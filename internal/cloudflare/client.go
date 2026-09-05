package cloudflare

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
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	defaultBaseURL       = "https://api.cloudflare.com/client/v4"
	maximumResponseBytes = 8 << 20
	maximumPages         = 100
)

var (
	ErrNotConfigured = errors.New("Cloudflare integration is not configured")
	ErrNotFound      = errors.New("Cloudflare resource was not found")
	ErrUnavailable   = errors.New("Cloudflare is unavailable")
	cloudflareID     = regexp.MustCompile(`^[A-Fa-f0-9]{32}$`)
)

type Zone struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
	Type   string `json:"type"`
}

type DNSRecord struct {
	ID        string `json:"id"`
	ZoneID    string `json:"zoneId"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	Target    string `json:"target"`
	TTL       int    `json:"ttl"`
	Proxied   *bool  `json:"proxied"`
	Proxiable bool   `json:"proxiable"`
}

type Tunnel struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Status       string `json:"status"`
	ConfigSource string `json:"configSource"`
}

// TunnelRoute intentionally omits the private origin service address. The
// hostname and origin kind are sufficient for mobile topology views without
// disclosing internal hostnames, socket paths, or ports.
type TunnelRoute struct {
	TunnelID   string `json:"tunnelId"`
	Hostname   string `json:"hostname"`
	Path       string `json:"path,omitempty"`
	OriginKind string `json:"originKind"`
}

type Client struct {
	baseURL   *url.URL
	token     string
	accountID string
	http      *http.Client
	logger    *slog.Logger
}

func NewClient(token, accountID string, logger *slog.Logger) (*Client, error) {
	token, accountID = strings.TrimSpace(token), strings.TrimSpace(accountID)
	if token == "" && accountID != "" {
		return nil, errors.New("CLOUDFLARE_API_TOKEN is required when CLOUDFLARE_ACCOUNT_ID is set")
	}
	if accountID != "" && !cloudflareID.MatchString(accountID) {
		return nil, errors.New("CLOUDFLARE_ACCOUNT_ID must be a 32-character Cloudflare identifier")
	}
	base, _ := url.Parse(defaultBaseURL)
	if logger == nil {
		logger = slog.Default()
	}
	client := &Client{token: token, accountID: accountID, logger: logger}
	if token != "" {
		client.baseURL = base
		client.http = &http.Client{Timeout: 15 * time.Second, CheckRedirect: sameOriginRedirects(base)}
	}
	return client, nil
}

func (client *Client) Configured() bool {
	return client != nil && client.baseURL != nil && client.http != nil && client.token != ""
}

func (client *Client) TunnelsConfigured() bool {
	return client.Configured() && client.accountID != ""
}

func (client *Client) Zones(ctx context.Context) ([]Zone, error) {
	type rawZone struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Status string `json:"status"`
		Type   string `json:"type"`
	}
	raw, err := collectPages[rawZone](ctx, client, "zones", 50)
	if err != nil {
		return nil, err
	}
	result := make([]Zone, len(raw))
	for index, item := range raw {
		result[index] = Zone{ID: item.ID, Name: item.Name, Status: item.Status, Type: item.Type}
	}
	return result, nil
}

func (client *Client) DNSRecords(ctx context.Context, zoneID string) ([]DNSRecord, error) {
	if !cloudflareID.MatchString(zoneID) {
		return nil, errors.New("Cloudflare zone identifier is invalid")
	}
	type rawRecord struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		Type      string `json:"type"`
		Content   string `json:"content"`
		TTL       int    `json:"ttl"`
		Proxied   *bool  `json:"proxied"`
		Proxiable bool   `json:"proxiable"`
	}
	raw, err := collectPages[rawRecord](ctx, client, "zones/"+zoneID+"/dns_records", 500)
	if err != nil {
		return nil, err
	}
	result := make([]DNSRecord, len(raw))
	for index, item := range raw {
		result[index] = DNSRecord{ID: item.ID, ZoneID: zoneID, Name: item.Name, Type: item.Type,
			Target: item.Content, TTL: item.TTL, Proxied: item.Proxied, Proxiable: item.Proxiable}
	}
	return result, nil
}

func (client *Client) Tunnels(ctx context.Context) ([]Tunnel, error) {
	if !client.TunnelsConfigured() {
		return nil, ErrNotConfigured
	}
	type rawTunnel struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		Status    string `json:"status"`
		ConfigSrc string `json:"config_src"`
	}
	raw, err := collectPages[rawTunnel](ctx, client, "accounts/"+client.accountID+"/cfd_tunnel", 1000)
	if err != nil {
		return nil, err
	}
	result := make([]Tunnel, len(raw))
	for index, item := range raw {
		result[index] = Tunnel{ID: item.ID, Name: item.Name, Status: item.Status, ConfigSource: item.ConfigSrc}
	}
	return result, nil
}

func (client *Client) TunnelRoutes(ctx context.Context, tunnelID string) ([]TunnelRoute, error) {
	if !client.TunnelsConfigured() {
		return nil, ErrNotConfigured
	}
	if _, err := uuid.Parse(tunnelID); err != nil {
		return nil, errors.New("Cloudflare tunnel identifier is invalid")
	}
	type rawConfiguration struct {
		Config struct {
			Ingress []struct {
				Hostname string `json:"hostname"`
				Service  string `json:"service"`
				Path     string `json:"path"`
			} `json:"ingress"`
		} `json:"config"`
	}
	var response envelope[rawConfiguration]
	path := "accounts/" + client.accountID + "/cfd_tunnel/" + tunnelID + "/configurations"
	if err := client.get(ctx, path, nil, &response); err != nil {
		return nil, err
	}
	if !response.Success {
		return nil, ErrUnavailable
	}
	result := make([]TunnelRoute, 0, len(response.Result.Config.Ingress))
	for _, item := range response.Result.Config.Ingress {
		if strings.TrimSpace(item.Hostname) == "" {
			continue
		}
		result = append(result, TunnelRoute{TunnelID: tunnelID, Hostname: item.Hostname,
			Path: item.Path, OriginKind: originKind(item.Service)})
	}
	return result, nil
}

type envelope[T any] struct {
	Success    bool `json:"success"`
	Result     T    `json:"result"`
	ResultInfo struct {
		Page       int `json:"page"`
		TotalPages int `json:"total_pages"`
	} `json:"result_info"`
}

func collectPages[T any](ctx context.Context, client *Client, path string, perPage int) ([]T, error) {
	result := make([]T, 0)
	for page := 1; page <= maximumPages; page++ {
		query := url.Values{"page": {strconv.Itoa(page)}, "per_page": {strconv.Itoa(perPage)}}
		var response envelope[[]T]
		if err := client.get(ctx, path, query, &response); err != nil {
			return nil, err
		}
		if !response.Success {
			return nil, ErrUnavailable
		}
		result = append(result, response.Result...)
		if response.ResultInfo.TotalPages > 0 {
			if page >= response.ResultInfo.TotalPages {
				return result, nil
			}
			continue
		}
		if len(response.Result) < perPage {
			return result, nil
		}
	}
	return nil, ErrUnavailable
}

func (client *Client) get(ctx context.Context, path string, query url.Values, target any) error {
	if !client.Configured() {
		return ErrNotConfigured
	}
	endpoint := *client.baseURL
	endpoint.Path = strings.TrimSuffix(client.baseURL.Path, "/") + "/" + path
	endpoint.RawQuery = query.Encode()
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
		client.logger.Warn("Cloudflare API request failed", "path", path)
		return ErrUnavailable
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound {
		return ErrNotFound
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		client.logger.Warn("Cloudflare API rejected request", "path", path, "status", response.StatusCode)
		return ErrUnavailable
	}
	payload, err := io.ReadAll(io.LimitReader(response.Body, maximumResponseBytes+1))
	if err != nil || len(payload) > maximumResponseBytes {
		return ErrUnavailable
	}
	if err := json.Unmarshal(payload, target); err != nil {
		client.logger.Warn("Cloudflare API returned invalid JSON", "path", path)
		return ErrUnavailable
	}
	return nil
}

func sameOriginRedirects(base *url.URL) func(*http.Request, []*http.Request) error {
	return func(request *http.Request, _ []*http.Request) error {
		if !strings.EqualFold(request.URL.Scheme, base.Scheme) || !strings.EqualFold(request.URL.Host, base.Host) {
			return http.ErrUseLastResponse
		}
		return nil
	}
}

func originKind(service string) string {
	parsed, err := url.Parse(strings.TrimSpace(service))
	if err == nil && parsed.Scheme != "" {
		return strings.ToLower(parsed.Scheme)
	}
	if strings.HasPrefix(strings.ToLower(service), "http_status:") {
		return "http_status"
	}
	return "unknown"
}
