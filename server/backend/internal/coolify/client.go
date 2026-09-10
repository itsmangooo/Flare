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
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const maximumResponseBytes = 8 << 20

var (
	ErrNotConfigured     = errors.New("Coolify integration is not configured")
	ErrNotFound          = errors.New("Coolify resource was not found")
	ErrUnavailable       = errors.New("Coolify is unavailable")
	ErrInvalidIdentifier = errors.New("Coolify identifier is invalid")
	identifier           = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)
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

type Deployment struct {
	UUID          string     `json:"uuid"`
	ResourceUUID  string     `json:"resourceUuid"`
	ResourceName  string     `json:"resourceName"`
	Status        *string    `json:"status"`
	Branch        *string    `json:"branch"`
	Commit        *string    `json:"commit"`
	CommitMessage *string    `json:"commitMessage"`
	StartedAt     *time.Time `json:"startedAt"`
	FinishedAt    *time.Time `json:"finishedAt"`
	Duration      *string    `json:"duration"`
	Logs          *string    `json:"logs"`
}

type DeploymentPage struct {
	Items    []Deployment `json:"items"`
	Page     int          `json:"page"`
	PageSize int          `json:"pageSize"`
	HasMore  bool         `json:"hasMore"`
}

type ActionResponse struct {
	Message     string  `json:"message"`
	OperationID *string `json:"operationId"`
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
		return nil, ErrInvalidIdentifier
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

func (client *Client) Deployments(ctx context.Context, page, pageSize int) (DeploymentPage, error) {
	page = min(max(page, 1), 1_000_000)
	pageSize = min(max(pageSize, 1), 50)
	applications, err := client.Applications(ctx)
	if err != nil {
		return DeploymentPage{}, err
	}
	take := min(max(page*pageSize+1, 1), 100)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	type result struct {
		items []Deployment
		err   error
	}
	jobs := make(chan string)
	results := make(chan result, len(applications))
	workers := min(6, len(applications))
	var group sync.WaitGroup
	for range workers {
		group.Add(1)
		go func() {
			defer group.Done()
			for applicationUUID := range jobs {
				items, err := client.ApplicationDeployments(ctx, applicationUUID, 0, take)
				select {
				case results <- result{items: items, err: err}:
				case <-ctx.Done():
					return
				}
				if err != nil {
					return
				}
			}
		}()
	}
	go func() {
		defer close(jobs)
		for _, application := range applications {
			select {
			case jobs <- application.UUID:
			case <-ctx.Done():
				return
			}
		}
	}()
	go func() {
		group.Wait()
		close(results)
	}()

	all := make([]Deployment, 0)
	for result := range results {
		if result.err != nil {
			return DeploymentPage{}, result.err
		}
		all = append(all, result.items...)
	}
	sort.SliceStable(all, func(left, right int) bool {
		if all[left].StartedAt == nil {
			return false
		}
		if all[right].StartedAt == nil {
			return true
		}
		return all[left].StartedAt.After(*all[right].StartedAt)
	})
	offset := (page - 1) * pageSize
	if offset >= len(all) {
		return DeploymentPage{Items: []Deployment{}, Page: page, PageSize: pageSize}, nil
	}
	end := min(offset+pageSize, len(all))
	items := append([]Deployment(nil), all[offset:end]...)
	return DeploymentPage{Items: items, Page: page, PageSize: pageSize, HasMore: end < len(all)}, nil
}

func (client *Client) ApplicationDeployments(ctx context.Context, applicationUUID string, skip, take int) ([]Deployment, error) {
	if !identifier.MatchString(applicationUUID) {
		return nil, ErrInvalidIdentifier
	}
	skip = max(skip, 0)
	take = min(max(take, 1), 100)
	query := url.Values{"skip": {strconv.Itoa(skip)}, "take": {strconv.Itoa(take)}}
	var raw []rawDeployment
	path := "deployments/applications/" + url.PathEscape(applicationUUID)
	if err := client.getQuery(ctx, path, query, &raw); err != nil {
		return nil, err
	}
	result := make([]Deployment, len(raw))
	for index, item := range raw {
		result[index] = mapDeployment(item, applicationUUID)
	}
	return result, nil
}

func (client *Client) Deployment(ctx context.Context, deploymentUUID string) (Deployment, error) {
	if !identifier.MatchString(deploymentUUID) {
		return Deployment{}, ErrInvalidIdentifier
	}
	var raw rawDeployment
	if err := client.get(ctx, "deployments/"+url.PathEscape(deploymentUUID), &raw); err != nil {
		return Deployment{}, err
	}
	return mapDeployment(raw, ""), nil
}

func (client *Client) StartApplication(ctx context.Context, applicationUUID string) (ActionResponse, error) {
	return client.applicationAction(ctx, applicationUUID, "start")
}

func (client *Client) StopApplication(ctx context.Context, applicationUUID string) (ActionResponse, error) {
	return client.applicationAction(ctx, applicationUUID, "stop")
}

func (client *Client) RestartApplication(ctx context.Context, applicationUUID string) (ActionResponse, error) {
	return client.applicationAction(ctx, applicationUUID, "restart")
}

func (client *Client) RestartService(ctx context.Context, serviceUUID string) (ActionResponse, error) {
	if !identifier.MatchString(serviceUUID) {
		return ActionResponse{}, ErrInvalidIdentifier
	}
	query := url.Values{"latest": {"false"}}
	return client.post(ctx, "services/"+url.PathEscape(serviceUUID)+"/restart", query)
}

func (client *Client) RedeployApplication(ctx context.Context, applicationUUID string) (ActionResponse, error) {
	if !identifier.MatchString(applicationUUID) {
		return ActionResponse{}, ErrInvalidIdentifier
	}
	query := url.Values{"uuid": {applicationUUID}, "force": {"false"}}
	return client.post(ctx, "deploy", query)
}

func (client *Client) applicationAction(ctx context.Context, applicationUUID, action string) (ActionResponse, error) {
	if !identifier.MatchString(applicationUUID) {
		return ActionResponse{}, ErrInvalidIdentifier
	}
	return client.post(ctx, "applications/"+url.PathEscape(applicationUUID)+"/"+action, nil)
}

type rawDeployment struct {
	DeploymentUUID  string  `json:"deployment_uuid"`
	UUID            string  `json:"uuid"`
	ApplicationUUID string  `json:"application_uuid"`
	ResourceUUID    string  `json:"resource_uuid"`
	ApplicationName string  `json:"application_name"`
	ResourceName    string  `json:"resource_name"`
	Status          *string `json:"status"`
	GitBranch       *string `json:"git_branch"`
	Branch          *string `json:"branch"`
	Commit          *string `json:"commit"`
	CommitMessage   *string `json:"commit_message"`
	CreatedAt       *string `json:"created_at"`
	StartedAt       *string `json:"started_at"`
	FinishedAt      *string `json:"finished_at"`
	UpdatedAt       *string `json:"updated_at"`
	Logs            *string `json:"logs"`
}

func mapDeployment(raw rawDeployment, fallbackResourceUUID string) Deployment {
	started := parseTime(first(raw.CreatedAt, raw.StartedAt))
	finished := parseTime(first(raw.FinishedAt, raw.UpdatedAt))
	if raw.Status != nil && (strings.EqualFold(*raw.Status, "in_progress") || strings.EqualFold(*raw.Status, "queued")) {
		finished = nil
	}
	resourceUUID := firstString(raw.ApplicationUUID, raw.ResourceUUID, fallbackResourceUUID)
	result := Deployment{
		UUID: firstString(raw.DeploymentUUID, raw.UUID), ResourceUUID: resourceUUID,
		ResourceName: firstString(raw.ApplicationName, raw.ResourceName, "Application"),
		Status:       raw.Status, Branch: first(raw.GitBranch, raw.Branch), Commit: raw.Commit,
		CommitMessage: raw.CommitMessage, StartedAt: started, FinishedAt: finished, Logs: truncate(raw.Logs, 1_000_000),
	}
	if started != nil && finished != nil && !finished.Before(*started) {
		value := contractDuration(finished.Sub(*started))
		result.Duration = &value
	}
	return result
}

func (client *Client) get(ctx context.Context, path string, target any) error {
	return client.getQuery(ctx, path, nil, target)
}

func (client *Client) getQuery(ctx context.Context, path string, query url.Values, target any) error {
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

func (client *Client) post(ctx context.Context, path string, query url.Values) (ActionResponse, error) {
	if !client.Configured() {
		return ActionResponse{}, ErrNotConfigured
	}
	endpoint := *client.baseURL
	endpoint.Path = strings.TrimSuffix(client.baseURL.Path, "/") + "/" + path
	endpoint.RawQuery = query.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), nil)
	if err != nil {
		return ActionResponse{}, fmt.Errorf("%w: request could not be created", ErrUnavailable)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", "Bearer "+client.token)
	response, err := client.http.Do(request)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return ActionResponse{}, context.Canceled
		}
		client.logger.Warn("Coolify API action failed", "path", path)
		return ActionResponse{}, ErrUnavailable
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound {
		return ActionResponse{}, ErrNotFound
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		client.logger.Warn("Coolify API rejected action", "path", path, "status", response.StatusCode)
		return ActionResponse{}, ErrUnavailable
	}
	payload, err := io.ReadAll(io.LimitReader(response.Body, maximumResponseBytes+1))
	if err != nil || len(payload) > maximumResponseBytes {
		return ActionResponse{}, ErrUnavailable
	}
	if len(strings.TrimSpace(string(payload))) == 0 {
		return ActionResponse{Message: "Operation queued."}, nil
	}
	var raw struct {
		Message        string  `json:"message"`
		DeploymentUUID *string `json:"deployment_uuid"`
		Deployments    []struct {
			DeploymentUUID *string `json:"deployment_uuid"`
		} `json:"deployments"`
	}
	if err := json.Unmarshal(payload, &raw); err != nil {
		client.logger.Warn("Coolify API action returned invalid JSON", "path", path)
		return ActionResponse{}, ErrUnavailable
	}
	operationID := raw.DeploymentUUID
	if operationID == nil && len(raw.Deployments) > 0 {
		operationID = raw.Deployments[0].DeploymentUUID
	}
	return ActionResponse{Message: fallback(raw.Message, "Operation queued."), OperationID: operationID}, nil
}

func first(values ...*string) *string {
	for _, value := range values {
		if value != nil && strings.TrimSpace(*value) != "" {
			return value
		}
	}
	return nil
}

func firstString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func parseTime(value *string) *time.Time {
	if value == nil {
		return nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, *value)
	if err != nil {
		return nil
	}
	parsed = parsed.UTC()
	return &parsed
}

func truncate(value *string, maximum int) *string {
	if value == nil || len(*value) <= maximum {
		return value
	}
	result := (*value)[:maximum]
	return &result
}

func contractDuration(duration time.Duration) string {
	duration = duration.Round(time.Microsecond)
	days := duration / (24 * time.Hour)
	duration -= days * 24 * time.Hour
	hours := duration / time.Hour
	duration -= hours * time.Hour
	minutes := duration / time.Minute
	duration -= minutes * time.Minute
	seconds := duration / time.Second
	fraction := (duration - seconds*time.Second) / (100 * time.Nanosecond)
	prefix := ""
	if days > 0 {
		prefix = fmt.Sprintf("%d.", days)
	}
	if fraction == 0 {
		return fmt.Sprintf("%s%02d:%02d:%02d", prefix, hours, minutes, seconds)
	}
	return fmt.Sprintf("%s%02d:%02d:%02d.%07d", prefix, hours, minutes, seconds, fraction)
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
