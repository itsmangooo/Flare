package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	HTTPAddress    string
	DatabaseURL    string
	DockerHost     string
	CoolifyBaseURL string
	CoolifyToken   string
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddress:    value("FLARE_HTTP_ADDRESS", ":8080"),
		DockerHost:     value("DOCKER_HOST", "unix:///var/run/docker.sock"),
		CoolifyBaseURL: strings.TrimRight(strings.TrimSpace(os.Getenv("COOLIFY_BASE_URL")), "/"),
		CoolifyToken:   strings.TrimSpace(os.Getenv("COOLIFY_API_TOKEN")),
	}

	databaseURL, err := postgresURL(strings.TrimSpace(os.Getenv("ConnectionStrings__Postgres")))
	if err != nil {
		return Config{}, err
	}
	cfg.DatabaseURL = databaseURL

	if (cfg.CoolifyBaseURL == "") != (cfg.CoolifyToken == "") {
		return Config{}, errors.New("COOLIFY_BASE_URL and COOLIFY_API_TOKEN must be set together")
	}
	if cfg.CoolifyBaseURL != "" {
		parsed, err := url.Parse(cfg.CoolifyBaseURL)
		if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
			return Config{}, errors.New("COOLIFY_BASE_URL must be an absolute HTTPS URL")
		}
	}
	return cfg, nil
}

func value(name, fallback string) string {
	if result := strings.TrimSpace(os.Getenv(name)); result != "" {
		return result
	}
	return fallback
}

// postgresURL accepts both PostgreSQL URLs and the Npgsql connection-string
// format already documented and deployed by Flare.
func postgresURL(input string) (string, error) {
	if input == "" {
		return "", errors.New("ConnectionStrings__Postgres is required")
	}
	if parsed, err := url.Parse(input); err == nil && (parsed.Scheme == "postgres" || parsed.Scheme == "postgresql") {
		return input, nil
	}

	parts, err := splitNpgsql(input)
	if err != nil {
		return "", err
	}
	values := make(map[string]string)
	for _, part := range parts {
		key, raw, found := strings.Cut(part, "=")
		if !found || strings.TrimSpace(key) == "" {
			continue
		}
		value, err := unquote(strings.TrimSpace(raw))
		if err != nil {
			return "", err
		}
		values[strings.ToLower(strings.TrimSpace(key))] = value
	}
	host := values["host"]
	database := values["database"]
	user := values["username"]
	if user == "" {
		user = values["user id"]
	}
	if host == "" || database == "" || user == "" {
		return "", errors.New("PostgreSQL connection string requires Host, Database, and Username")
	}
	port := 5432
	if raw := values["port"]; raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 65535 {
			return "", fmt.Errorf("invalid PostgreSQL port %q", raw)
		}
		port = parsed
	}

	tlsMode := "prefer"
	switch strings.ToLower(values["ssl mode"]) {
	case "", "prefer":
	case "disable":
		tlsMode = "disable"
	case "require":
		tlsMode = "require"
	case "verify-ca":
		tlsMode = "verify-ca"
	case "verify-full":
		tlsMode = "verify-full"
	default:
		return "", errors.New("unsupported PostgreSQL SSL Mode")
	}

	result := &url.URL{
		Scheme: "postgresql",
		Host:   fmt.Sprintf("%s:%d", host, port),
		Path:   database,
	}
	if password := values["password"]; password != "" {
		result.User = url.UserPassword(user, password)
	} else {
		result.User = url.User(user)
	}
	query := result.Query()
	query.Set("sslmode", tlsMode)
	result.RawQuery = query.Encode()
	return result.String(), nil
}

func splitNpgsql(input string) ([]string, error) {
	var parts []string
	start := 0
	var quote byte
	valueStart := false
	for index := 0; index < len(input); index++ {
		character := input[index]
		if quote != 0 {
			if character == quote {
				if index+1 < len(input) && input[index+1] == quote {
					index++
					continue
				}
				quote = 0
			}
			continue
		}
		if character == '=' {
			valueStart = true
			continue
		}
		if valueStart && (character == ' ' || character == '\t') {
			continue
		}
		if valueStart && (character == '\'' || character == '"') {
			quote = character
			valueStart = false
			continue
		}
		valueStart = false
		if character == ';' {
			parts = append(parts, input[start:index])
			start = index + 1
		}
	}
	if quote != 0 {
		return nil, errors.New("PostgreSQL connection string contains an unterminated quoted value")
	}
	return append(parts, input[start:]), nil
}

func unquote(value string) (string, error) {
	if value == "" || (value[0] != '\'' && value[0] != '"') {
		return value, nil
	}
	if len(value) < 2 || value[len(value)-1] != value[0] {
		return "", errors.New("PostgreSQL connection string contains an invalid quoted value")
	}
	quote := string(value[0])
	return strings.ReplaceAll(value[1:len(value)-1], quote+quote, quote), nil
}
