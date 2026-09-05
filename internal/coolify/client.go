package coolify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const maximumResponseBytes = 8 << 20

var (
	ErrNotConfigured = errors.New("Coolify integration is not configured")
	ErrNotFound      = errors.New("Coolify resource was not found")
	ErrUnavailable   = errors.New("Coolify is unavailable")
	identifier       = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)
)

type Server struct {
	UUID        string `json:"uuid"`
	Name        string `json:"name"`
	IsReachable *bool  `json:"isReachable"`
	IsUsable    *bool  `json:"isUsable"`
}

type Resource struct {
	UUID   string  `json:"uuid"`
	Name   string  `json:"name"`
	Type   string  `json:"type"`
	Status *string `json:"status"`
}

type Application struct {
	UUID      string  `json:"uuid"`
	Name      string  `json:"name"`
	Status    *string `json:"status"`
	FQDN      *string `json:"fqdn"`
	GitBranch *string `json:"gitBranch"`
}

type Service struct {
	UUID        string  `json:"uuid"`
	Name        string  `json:"name"`
	Status      *string `json:"status"`
	Description *string `json:"description"`
}

type Client struct {
	baseURL *url.URL
	token   string
	http    *http.Client
	logger  *slog.Logger
}

func NewClient(baseURL, token string, logger *slog.Logger) (*Client, error) {
	base, configured, err := parseBaseURL(baseURL, token)
	if err != nil {
		return nil, err
	}
	if logger == nil {
		logger = slog.Default()
	}
	client := &Client{baseURL: base, token: strings.TrimSpace(token), logger: logger}
	if configured {
		client.http = &http.Client{
			Timeout:       15 * time.Second,
			CheckRedirect: sameOriginRedirects(base),
		}
	}
	return client, nil
}

func (client *Client) Configured() bool {
	return client != nil && client.baseURL != nil && client.http != nil && client.token != ""
}

func (client *Client) Servers(ctx context.Context) ([]Server, error) {
	type rawServer struct {
		UUID     string `json:"uuid"`
		Name     string `json:"name"`
		Settings struct {
			IsReachable *bool `json:"is_reachable"`
			IsUsable    *bool `json:"is_usable"`
		} `json:"settings"`
	}
	var raw []rawServer
	if err := client.get(ctx, "servers", &raw); err != nil {
		return nil, err
	}
	result := make([]Server, len(raw))
	for index, item := range raw {
		result[index] = Server{UUID: item.UUID, Name: fallback(item.Name, "Unnamed server"),
			IsReachable: item.Settings.IsReachable, IsUsable: item.Settings.IsUsable}
	}
	return result, nil
}

func (client *Client) ServerResources(ctx context.Context, serverUUID string) ([]Resource, error) {
	if !identifier.MatchString(serverUUID) {
		return nil, errors.New("Coolify identifier is invalid")
	}
	type rawResource struct {
		UUID   string  `json:"uuid"`
		Name   string  `json:"name"`
		Type   string  `json:"type"`
		Status *string `json:"status"`
	}
	var raw []rawResource
	if err := client.get(ctx, "servers/"+url.PathEscape(serverUUID)+"/resources", &raw); err != nil {
		return nil, err
	}
	result := make([]Resource, len(raw))
	for index, item := range raw {
		result[index] = Resource{UUID: item.UUID, Name: fallback(item.Name, "Unnamed resource"),
			Type: fallback(item.Type, "unknown"), Status: item.Status}
	}
	return result, nil
}

func (client *Client) Applications(ctx context.Context) ([]Application, error) {
	type rawApplication struct {
		UUID      string  `json:"uuid"`
		Name      string  `json:"name"`
		Status    *string `json:"status"`
		FQDN      *string `json:"fqdn"`
		GitBranch *string `json:"git_branch"`
		Branch    *string `json:"branch"`
	}
	var raw []rawApplication
	if err := client.get(ctx, "applications", &raw); err != nil {
		return nil, err
	}
	result := make([]Application, len(raw))
	for index, item := range raw {
		branch := item.GitBranch
		if branch == nil {
			branch = item.Branch
		}
		result[index] = Application{UUID: item.UUID, Name: fallback(item.Name, "Unnamed application"),
			Status: item.Status, FQDN: item.FQDN, GitBranch: branch}
	}
	return result, nil
}

func (client *Client) Services(ctx context.Context) ([]Service, error) {
	type rawService struct {
		UUID        string  `json:"uuid"`
		Name        string  `json:"name"`
		Status      *string `json:"status"`
		Description *string `json:"description"`
	}
	var raw []rawService
	if err := client.get(ctx, "services", &raw); err != nil {
		return nil, err
	}
	result := make([]Service, len(raw))
	for index, item := range raw {
		result[index] = Service{UUID: item.UUID, Name: fallback(item.Name, "Unnamed service"),
			Status: item.Status, Description: item.Description}
	}
	return result, nil
}

func (client *Client) get(ctx context.Context, path string, target any) error {
	if !client.Configured() {
		return ErrNotConfigured
	}
	endpoint := *client.baseURL
	endpoint.Path = strings.TrimSuffix(client.baseURL.Path, "/") + "/" + path
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return fmt.Errorf("%w: request could not be created", ErrUnavailable)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", "Bearer "+client.token)
	response, err := client.http.Do(request)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return context.Canceled
		}
		client.logger.Warn("Coolify API request failed", "path", path)
		return ErrUnavailable
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound {
		return ErrNotFound
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		client.logger.Warn("Coolify API rejected request", "path", path, "status", response.StatusCode)
		return ErrUnavailable
	}
	payload, err := io.ReadAll(io.LimitReader(response.Body, maximumResponseBytes+1))
	if err != nil || len(payload) > maximumResponseBytes {
		return ErrUnavailable
	}
	if err := json.Unmarshal(payload, target); err != nil {
		client.logger.Warn("Coolify API returned invalid JSON", "path", path)
		return ErrUnavailable
	}
	return nil
}

func parseBaseURL(rawURL, token string) (*url.URL, bool, error) {
	rawURL, token = strings.TrimSpace(rawURL), strings.TrimSpace(token)
	if rawURL == "" && token == "" {
		return nil, false, nil
	}
	if rawURL == "" || token == "" {
		return nil, false, errors.New("COOLIFY_BASE_URL and COOLIFY_API_TOKEN must be set together")
	}
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, false, errors.New("COOLIFY_BASE_URL must be an absolute HTTPS URL without credentials, query, or fragment")
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/api/v1"
	return parsed, true, nil
}

func sameOriginRedirects(base *url.URL) func(*http.Request, []*http.Request) error {
	return func(request *http.Request, _ []*http.Request) error {
		if !strings.EqualFold(request.URL.Scheme, base.Scheme) || !strings.EqualFold(request.URL.Host, base.Host) {
			return http.ErrUseLastResponse
		}
		return nil
	}
}

func fallback(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
