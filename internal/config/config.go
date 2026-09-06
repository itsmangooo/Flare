package config

import (
	"errors"
	"fmt"
	"math"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var cloudflareIdentifier = regexp.MustCompile(`^[A-Fa-f0-9]{32}$`)
var ntfyTopic = regexp.MustCompile(`^[-_A-Za-z0-9]{1,64}$`)

type Config struct {
	HTTPAddress         string
	DatabaseURL         string
	DockerHost          string
	HostName            string
	HostProcPath        string
	HostRootFSPath      string
	CoolifyBaseURL      string
	CoolifyToken        string
	CloudflareToken     string
	CloudflareAccountID string
	NtfyBaseURL         string
	NtfyTopic           string
	NtfyToken           string
	BootstrapToken      string
	JWTSigningKey       string
	JWTIssuer           string
	JWTAudience         string
	AccessTokenTTL      time.Duration
	RefreshTokenTTL     time.Duration
	HostCPUAlertPercent float64
	HostRAMAlertPercent float64
	HostAlertSamples    int
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddress:         value("FLARE_HTTP_ADDRESS", ":8080"),
		DockerHost:          value("DOCKER_HOST", "unix:///var/run/docker.sock"),
		HostName:            value("FLARE_HOST_NAME", "homelab"),
		HostProcPath:        value("HOST_PROC_PATH", "/host/proc"),
		HostRootFSPath:      value("HOST_ROOTFS_PATH", "/host/rootfs"),
		CoolifyBaseURL:      strings.TrimRight(strings.TrimSpace(os.Getenv("COOLIFY_BASE_URL")), "/"),
		CoolifyToken:        strings.TrimSpace(os.Getenv("COOLIFY_API_TOKEN")),
		CloudflareToken:     strings.TrimSpace(os.Getenv("CLOUDFLARE_API_TOKEN")),
		CloudflareAccountID: strings.TrimSpace(os.Getenv("CLOUDFLARE_ACCOUNT_ID")),
		NtfyBaseURL:         strings.TrimRight(strings.TrimSpace(os.Getenv("NTFY_BASE_URL")), "/"),
		NtfyTopic:           strings.TrimSpace(os.Getenv("NTFY_TOPIC")),
		NtfyToken:           strings.TrimSpace(os.Getenv("NTFY_TOKEN")),
		BootstrapToken:      strings.TrimSpace(os.Getenv("FLARE_BOOTSTRAP_TOKEN")),
		JWTSigningKey:       os.Getenv("FLARE_JWT_SIGNING_KEY"),
		JWTIssuer:           value("FLARE_JWT_ISSUER", "Flare.Api"),
		JWTAudience:         value("FLARE_JWT_AUDIENCE", "Flare.Mobile"),
	}
	accessMinutes, err := boundedInt("FLARE_ACCESS_TOKEN_MINUTES", 15, 5, 60)
	if err != nil {
		return Config{}, err
	}
	refreshDays, err := boundedInt("FLARE_REFRESH_TOKEN_DAYS", 30, 1, 90)
	if err != nil {
		return Config{}, err
	}
	cfg.AccessTokenTTL = time.Duration(accessMinutes) * time.Minute
	cfg.RefreshTokenTTL = time.Duration(refreshDays) * 24 * time.Hour
	cfg.HostCPUAlertPercent, err = boundedFloat("FLARE_ALERT_CPU_PERCENT", 90, 1, 100)
	if err != nil {
		return Config{}, err
	}
	cfg.HostRAMAlertPercent, err = boundedFloat("FLARE_ALERT_MEMORY_PERCENT", 90, 1, 100)
	if err != nil {
		return Config{}, err
	}
	cfg.HostAlertSamples, err = boundedInt("FLARE_ALERT_SUSTAINED_SAMPLES", 5, 2, 20)
	if err != nil {
		return Config{}, err
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
	if cfg.CloudflareToken == "" && cfg.CloudflareAccountID != "" {
		return Config{}, errors.New("CLOUDFLARE_API_TOKEN is required when CLOUDFLARE_ACCOUNT_ID is set")
	}
	if cfg.CloudflareAccountID != "" && !cloudflareIdentifier.MatchString(cfg.CloudflareAccountID) {
		return Config{}, errors.New("CLOUDFLARE_ACCOUNT_ID must be a 32-character Cloudflare identifier")
	}
	if (cfg.NtfyBaseURL == "") != (cfg.NtfyTopic == "") {
		return Config{}, errors.New("NTFY_BASE_URL and NTFY_TOPIC must be set together")
	}
	if cfg.NtfyToken != "" && cfg.NtfyBaseURL == "" {
		return Config{}, errors.New("NTFY_BASE_URL and NTFY_TOPIC are required when NTFY_TOKEN is set")
	}
	if cfg.NtfyBaseURL != "" {
		parsed, err := url.Parse(cfg.NtfyBaseURL)
		if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
			return Config{}, errors.New("NTFY_BASE_URL must be an absolute HTTPS origin without credentials, query, or fragment")
		}
		if !ntfyTopic.MatchString(cfg.NtfyTopic) {
			return Config{}, errors.New("NTFY_TOPIC must be 1-64 letters, numbers, underscores, or dashes")
		}
	}
	if len([]byte(cfg.JWTSigningKey)) < 32 {
		return Config{}, errors.New("FLARE_JWT_SIGNING_KEY must contain at least 32 UTF-8 bytes")
	}
	return cfg, nil
}

func boundedInt(name string, fallback, minimum, maximum int) (int, error) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback, nil
	}
	result, err := strconv.Atoi(raw)
	if err != nil || result < minimum || result > maximum {
		return 0, fmt.Errorf("%s must be between %d and %d", name, minimum, maximum)
	}
	return result, nil
}

func boundedFloat(name string, fallback, minimum, maximum float64) (float64, error) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback, nil
	}
	result, err := strconv.ParseFloat(raw, 64)
	if err != nil || math.IsNaN(result) || math.IsInf(result, 0) || result < minimum || result > maximum {
		return 0, fmt.Errorf("%s must be between %g and %g", name, minimum, maximum)
	}
	return result, nil
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
