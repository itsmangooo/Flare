package containers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/containerd/errdefs"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/itsmangooo/flare/server/backend/internal/auth"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/moby/moby/api/pkg/stdcopy"
	containertypes "github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

type dockerAPI interface {
	ContainerList(context.Context, client.ContainerListOptions) (client.ContainerListResult, error)
	ContainerInspect(context.Context, string, client.ContainerInspectOptions) (client.ContainerInspectResult, error)
	ContainerStats(context.Context, string, client.ContainerStatsOptions) (client.ContainerStatsResult, error)
	ContainerLogs(context.Context, string, client.ContainerLogsOptions) (client.ContainerLogsResult, error)
	ContainerStart(context.Context, string, client.ContainerStartOptions) (client.ContainerStartResult, error)
	ContainerStop(context.Context, string, client.ContainerStopOptions) (client.ContainerStopResult, error)
	ContainerRestart(context.Context, string, client.ContainerRestartOptions) (client.ContainerRestartResult, error)
}

type auditDatabase interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

type Handler struct {
	docker  dockerAPI
	audits  auditDatabase
	logger  *slog.Logger
	limiter *operationLimiter
}

type operationWindow struct {
	started  time.Time
	requests int
}

type operationLimiter struct {
	mu      sync.Mutex
	windows map[string]operationWindow
}

type summaryResponse struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Image       string     `json:"image"`
	State       string     `json:"state"`
	Health      string     `json:"health"`
	StartedAt   *time.Time `json:"startedAt"`
	CPUPercent  *float64   `json:"cpuPercent"`
	MemoryBytes *int64     `json:"memoryBytes"`
}

type detailResponse struct {
	ID                   string            `json:"id"`
	ShortID              string            `json:"shortId"`
	Name                 string            `json:"name"`
	Image                string            `json:"image"`
	State                string            `json:"state"`
	Health               string            `json:"health"`
	CreatedAt            time.Time         `json:"createdAt"`
	StartedAt            *time.Time        `json:"startedAt"`
	RestartCount         int               `json:"restartCount"`
	CPUPercent           *float64          `json:"cpuPercent"`
	MemoryBytes          *int64            `json:"memoryBytes"`
	MemoryLimitBytes     *int64            `json:"memoryLimitBytes"`
	NetworkReceiveBytes  *int64            `json:"networkReceiveBytes"`
	NetworkTransmitBytes *int64            `json:"networkTransmitBytes"`
	Ports                []portResponse    `json:"ports"`
	Labels               map[string]string `json:"labels"`
}

type portResponse struct {
	PrivatePort int     `json:"privatePort"`
	PublicPort  *int    `json:"publicPort"`
	Protocol    string  `json:"protocol"`
	HostIP      *string `json:"hostIp"`
}

type statsResponse struct {
	CPUPercent           *float64
	MemoryBytes          int64
	MemoryLimitBytes     int64
	NetworkReceiveBytes  int64
	NetworkTransmitBytes int64
}

type logPageResponse struct {
	Lines           []string   `json:"lines"`
	RetrievedAt     time.Time  `json:"retrievedAt"`
	Truncated       bool       `json:"truncated"`
	RequestedTail   int        `json:"requestedTail"`
	OldestTimestamp *time.Time `json:"oldestTimestamp"`
	NewestTimestamp *time.Time `json:"newestTimestamp"`
}

type parsedLogLine struct {
	text      string
	timestamp *time.Time
	index     int
}

type actionResponse struct {
	Message     string  `json:"message"`
	OperationID *string `json:"operationId"`
}

var containerIDPattern = regexp.MustCompile(`^[a-fA-F0-9]{12,64}$`)
var sensitiveLabelPattern = regexp.MustCompile(`(?i)token|secret|pass(word|wd)?|credential|private[-_. ]?key|api[-_. ]?key`)

func NewDockerClient(host string) (*client.Client, error) {
	return client.NewClientWithOpts(client.WithHost(host), client.WithAPIVersionNegotiation())
}

func NewHandler(docker dockerAPI, audits auditDatabase, logger *slog.Logger) http.Handler {
	h := &Handler{docker: docker, audits: audits, logger: logger, limiter: &operationLimiter{windows: make(map[string]operationWindow)}}
	router := chi.NewRouter()
	router.Get("/", h.list)
	router.Get("/{id}", h.detail)
	router.Get("/{id}/logs", h.logs)
	router.Post("/{id}/start", h.start)
	router.Post("/{id}/stop", h.stop)
	router.Post("/{id}/restart", h.restart)
	return router
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	listed, err := h.docker.ContainerList(r.Context(), client.ContainerListOptions{All: true})
	if err != nil {
		h.unavailable(w, r, err)
		return
	}
	items := listed.Items
	results := make([]summaryResponse, len(items))
	workers := make(chan struct{}, 6)
	var group sync.WaitGroup
	for index := range items {
		index := index
		group.Add(1)
		go func() {
			defer group.Done()
			workers <- struct{}{}
			defer func() { <-workers }()
			results[index] = h.toSummary(r.Context(), items[index])
		}()
	}
	group.Wait()
	sort.Slice(results, func(i, j int) bool {
		left, right := strings.ToLower(results[i].Name), strings.ToLower(results[j].Name)
		if left == right {
			return results[i].ID < results[j].ID
		}
		return left < right
	})
	writeJSON(w, http.StatusOK, results)
}

func (h *Handler) detail(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !containerIDPattern.MatchString(id) {
		writeProblem(w, http.StatusBadRequest, "Invalid request.", "Container identifier is invalid.")
		return
	}
	inspected, err := h.docker.ContainerInspect(r.Context(), id, client.ContainerInspectOptions{})
	if err != nil {
		if errdefs.IsNotFound(err) {
			writeProblem(w, http.StatusNotFound, "Container not found.", "The requested container does not exist.")
			return
		}
		h.unavailable(w, r, err)
		return
	}
	value := inspected.Container
	result := detailResponse{
		ID: value.ID, ShortID: shortID(value.ID), Name: strings.TrimLeft(value.Name, "/"),
		State: "Unknown", Health: "None", Ports: []portResponse{}, Labels: map[string]string{},
		RestartCount: max(value.RestartCount, 0),
	}
	if value.Config != nil {
		result.Image = value.Config.Image
		result.Labels = sanitizeLabels(value.Config.Labels)
	}
	if parsed := timestamp(value.Created); parsed != nil {
		result.CreatedAt = *parsed
	}
	if value.State != nil {
		result.State = state(value.State.Status)
		result.StartedAt = timestamp(value.State.StartedAt)
		if value.State.Health != nil {
			result.Health = health(string(value.State.Health.Status))
		}
		if result.State == "Running" {
			if stats := h.readStats(r.Context(), id); stats != nil {
				result.CPUPercent = stats.CPUPercent
				result.MemoryBytes = &stats.MemoryBytes
				result.MemoryLimitBytes = &stats.MemoryLimitBytes
				result.NetworkReceiveBytes = &stats.NetworkReceiveBytes
				result.NetworkTransmitBytes = &stats.NetworkTransmitBytes
			}
		}
	}
	result.Ports = ports(value.NetworkSettings)
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) logs(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !containerIDPattern.MatchString(id) {
		writeProblem(w, http.StatusBadRequest, "Invalid request.", "Container identifier is invalid.")
		return
	}
	tail, err := parseTail(r.URL.Query().Get("tail"))
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "Invalid request.", "Tail must be an integer.")
		return
	}
	until, err := parseBefore(r.URL.Query().Get("before"))
	if err != nil {
		writeProblem(w, http.StatusBadRequest, "Invalid request.", "Before must be an RFC 3339 timestamp.")
		return
	}
	inspected, err := h.docker.ContainerInspect(r.Context(), id, client.ContainerInspectOptions{})
	if err != nil {
		h.dockerError(w, r, err)
		return
	}
	tty := inspected.Container.Config != nil && inspected.Container.Config.Tty
	stream, err := h.docker.ContainerLogs(r.Context(), id, client.ContainerLogsOptions{
		ShowStdout: true, ShowStderr: true, Timestamps: true, Tail: strconv.Itoa(tail), Until: until,
	})
	if err != nil {
		h.dockerError(w, r, err)
		return
	}
	defer stream.Close()
	output, byteTruncated, err := readLogOutput(stream, tty)
	if err != nil {
		h.unavailable(w, r, err)
		return
	}
	lines := parseLogLines(output, tail)
	response := logPageResponse{
		Lines: make([]string, len(lines)), RetrievedAt: time.Now().UTC(),
		Truncated: byteTruncated || len(lines) >= tail, RequestedTail: tail,
	}
	for index, line := range lines {
		response.Lines[index] = line.text
	}
	if len(lines) > 0 {
		response.OldestTimestamp = lines[0].timestamp
		response.NewestTimestamp = lines[len(lines)-1].timestamp
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) start(w http.ResponseWriter, r *http.Request) {
	h.perform(w, r, "container.start", func(ctx context.Context, id string) error {
		_, err := h.docker.ContainerStart(ctx, id, client.ContainerStartOptions{})
		return err
	})
}

func (h *Handler) stop(w http.ResponseWriter, r *http.Request) {
	h.perform(w, r, "container.stop", func(ctx context.Context, id string) error {
		timeout := 20
		_, err := h.docker.ContainerStop(ctx, id, client.ContainerStopOptions{Timeout: &timeout})
		return err
	})
}

func (h *Handler) restart(w http.ResponseWriter, r *http.Request) {
	h.perform(w, r, "container.restart", func(ctx context.Context, id string) error {
		timeout := 20
		_, err := h.docker.ContainerRestart(ctx, id, client.ContainerRestartOptions{Timeout: &timeout})
		return err
	})
}

func (h *Handler) perform(w http.ResponseWriter, r *http.Request, action string, operation func(context.Context, string) error) {
	user, authenticated := auth.UserFromContext(r.Context())
	if !authenticated || !auth.HasRole(r.Context(), "Administrator") {
		writeProblem(w, http.StatusForbidden, "Forbidden.", "Administrator access is required.")
		return
	}
	if !h.limiter.allow(operationKey(user.ID, r.RemoteAddr), time.Now()) {
		w.Header().Set("Retry-After", "60")
		writeProblem(w, http.StatusTooManyRequests, "Too many requests.", "Try again later.")
		return
	}
	id := chi.URLParam(r, "id")
	if !containerIDPattern.MatchString(id) {
		writeProblem(w, http.StatusBadRequest, "Invalid request.", "Container identifier is invalid.")
		return
	}
	if err := operation(r.Context(), id); err != nil {
		if errors.Is(err, context.Canceled) {
			return
		}
		if auditErr := h.recordAudit(r.Context(), user, action, id, 1, "Docker operation failed."); auditErr != nil {
			h.internalError(w, r, auditErr)
			return
		}
		h.unavailable(w, r, err)
		return
	}
	if err := h.recordAudit(r.Context(), user, action, id, 0, ""); err != nil {
		h.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusAccepted, actionResponse{Message: "Container operation accepted."})
}

func (h *Handler) recordAudit(ctx context.Context, user auth.User, action, target string, result int, detail string) error {
	var detailValue any
	if detail != "" {
		detailValue = detail
	}
	_, err := h.audits.Exec(ctx, `INSERT INTO "AuditEvents" ("Id","UserId","Actor","Action","Target","Timestamp","Result","CorrelationId","Detail") VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		uuid.New(), user.ID, user.Email, action, target, time.Now().UTC(), result, middleware.GetReqID(ctx), detailValue)
	return err
}

func (limiter *operationLimiter) allow(key string, now time.Time) bool {
	limiter.mu.Lock()
	defer limiter.mu.Unlock()
	window := limiter.windows[key]
	if window.started.IsZero() || now.Sub(window.started) >= time.Minute {
		window = operationWindow{started: now}
	}
	window.requests++
	limiter.windows[key] = window
	if len(limiter.windows) > 1024 {
		for id, candidate := range limiter.windows {
			if now.Sub(candidate.started) >= time.Minute {
				delete(limiter.windows, id)
			}
		}
	}
	return window.requests <= 30
}

func operationKey(userID uuid.UUID, remoteAddress string) string {
	host := remoteAddress
	if parsed, _, err := net.SplitHostPort(remoteAddress); err == nil {
		host = parsed
	}
	if len(host) > 64 {
		host = host[:64]
	}
	return userID.String() + ":" + host
}

func (h *Handler) toSummary(ctx context.Context, item containertypes.Summary) summaryResponse {
	name := shortID(item.ID)
	if len(item.Names) > 0 && strings.Trim(item.Names[0], "/") != "" {
		name = strings.TrimLeft(item.Names[0], "/")
	}
	result := summaryResponse{ID: item.ID, Name: name, Image: item.Image, State: state(item.State), Health: healthFromText(item.Status)}
	inspected, err := h.docker.ContainerInspect(ctx, item.ID, client.ContainerInspectOptions{})
	if err == nil && inspected.Container.State != nil {
		result.StartedAt = timestamp(inspected.Container.State.StartedAt)
		if inspected.Container.State.Health != nil {
			result.Health = health(string(inspected.Container.State.Health.Status))
		}
	}
	if result.State == "Running" {
		if stats := h.readStats(ctx, item.ID); stats != nil {
			result.CPUPercent = stats.CPUPercent
			result.MemoryBytes = &stats.MemoryBytes
		}
	}
	return result
}

func (h *Handler) readStats(ctx context.Context, id string) *statsResponse {
	statsCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	response, err := h.docker.ContainerStats(statsCtx, id, client.ContainerStatsOptions{Stream: false, IncludePreviousSample: true})
	if err != nil {
		return nil
	}
	defer response.Body.Close()
	var value containertypes.StatsResponse
	if json.NewDecoder(io.LimitReader(response.Body, 2<<20)).Decode(&value) != nil {
		return nil
	}
	cpuDelta := subtract(value.CPUStats.CPUUsage.TotalUsage, value.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := subtract(value.CPUStats.SystemUsage, value.PreCPUStats.SystemUsage)
	cpus := value.CPUStats.OnlineCPUs
	if cpus == 0 {
		cpus = uint32(max(len(value.CPUStats.CPUUsage.PercpuUsage), 1))
	}
	var cpu *float64
	if systemDelta > 0 {
		percentage := float64(cpuDelta) / float64(systemDelta) * float64(cpus) * 100
		cpu = &percentage
	}
	cache := value.MemoryStats.Stats["inactive_file"]
	if cache == 0 {
		cache = value.MemoryStats.Stats["cache"]
	}
	memory := subtract(value.MemoryStats.Usage, cache)
	return &statsResponse{
		CPUPercent:           cpu,
		MemoryBytes:          safeInt64(memory),
		MemoryLimitBytes:     safeInt64(value.MemoryStats.Limit),
		NetworkReceiveBytes:  sumNetwork(value, true),
		NetworkTransmitBytes: sumNetwork(value, false),
	}
}

func ports(settings *containertypes.NetworkSettings) []portResponse {
	result := []portResponse{}
	if settings == nil {
		return result
	}
	for port, bindings := range settings.Ports {
		privatePort := int(port.Num())
		protocol := string(port.Proto())
		if len(bindings) == 0 {
			result = append(result, portResponse{PrivatePort: privatePort, Protocol: protocol})
			continue
		}
		for _, binding := range bindings {
			var publicPort *int
			if parsed, err := strconv.Atoi(binding.HostPort); err == nil {
				publicPort = &parsed
			}
			var hostIP *string
			if binding.HostIP.IsValid() {
				text := binding.HostIP.String()
				hostIP = &text
			}
			result = append(result, portResponse{PrivatePort: privatePort, PublicPort: publicPort, Protocol: protocol, HostIP: hostIP})
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].PrivatePort != result[j].PrivatePort {
			return result[i].PrivatePort < result[j].PrivatePort
		}
		return result[i].Protocol < result[j].Protocol
	})
	return result
}

func sanitizeLabels(labels map[string]string) map[string]string {
	keys := make([]string, 0, len(labels))
	for key := range labels {
		if len(key) <= 256 && !sensitiveLabelPattern.MatchString(key) {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	if len(keys) > 30 {
		keys = keys[:30]
	}
	result := make(map[string]string, len(keys))
	for _, key := range keys {
		value := labels[key]
		if len(value) > 512 {
			value = value[:512]
		}
		result[key] = value
	}
	return result
}

func sumNetwork(value containertypes.StatsResponse, receive bool) int64 {
	var total uint64
	for _, network := range value.Networks {
		part := network.TxBytes
		if receive {
			part = network.RxBytes
		}
		if ^uint64(0)-total < part {
			return int64(^uint64(0) >> 1)
		}
		total += part
	}
	return safeInt64(total)
}

func safeInt64(value uint64) int64 {
	if value > uint64(^uint64(0)>>1) {
		return int64(^uint64(0) >> 1)
	}
	return int64(value)
}

func parseTail(value string) (int, error) {
	if value == "" {
		return 300, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}
	return min(max(parsed, 1), 2000), nil
}

func parseBefore(value string) (string, error) {
	if value == "" {
		return "", nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return "", err
	}
	return parsed.UTC().Format(time.RFC3339Nano), nil
}

func readLogOutput(stream io.Reader, tty bool) ([]byte, bool, error) {
	const maximumReadBytes = 8 * 1024 * 1024
	const maximumResponseBytes = 2 * 1024 * 1024
	raw, err := io.ReadAll(io.LimitReader(stream, maximumReadBytes+1))
	if err != nil {
		return nil, false, err
	}
	truncated := len(raw) > maximumReadBytes
	if truncated {
		raw = raw[:maximumReadBytes]
	}
	output := raw
	if !tty {
		var decoded bytes.Buffer
		_, decodeErr := stdcopy.StdCopy(&decoded, &decoded, bytes.NewReader(raw))
		if decodeErr != nil && !truncated {
			return nil, false, decodeErr
		}
		output = decoded.Bytes()
	}
	if len(output) > maximumResponseBytes {
		truncated = true
		output = output[len(output)-maximumResponseBytes:]
		if newline := bytes.IndexByte(output, '\n'); newline >= 0 {
			output = output[newline+1:]
		}
	}
	return output, truncated, nil
}

func parseLogLines(output []byte, tail int) []parsedLogLine {
	parts := strings.FieldsFunc(string(output), func(r rune) bool { return r == '\r' || r == '\n' })
	lines := make([]parsedLogLine, len(parts))
	for index, text := range parts {
		lines[index] = parsedLogLine{text: text, timestamp: logTimestamp(text), index: index}
	}
	sort.SliceStable(lines, func(i, j int) bool {
		left, right := lines[i].timestamp, lines[j].timestamp
		if left == nil && right != nil {
			return true
		}
		if left != nil && right == nil {
			return false
		}
		if left != nil && right != nil && !left.Equal(*right) {
			return left.Before(*right)
		}
		return lines[i].index < lines[j].index
	})
	if len(lines) > tail {
		lines = lines[len(lines)-tail:]
	}
	return lines
}

func logTimestamp(line string) *time.Time {
	value := line
	if separator := strings.IndexByte(line, ' '); separator >= 0 {
		value = line[:separator]
	}
	return timestamp(value)
}

func state(value containertypes.ContainerState) string {
	switch strings.ToLower(string(value)) {
	case "running":
		return "Running"
	case "exited", "created":
		return "Stopped"
	case "paused":
		return "Paused"
	case "restarting":
		return "Restarting"
	case "dead", "removing":
		return "Dead"
	default:
		return "Unknown"
	}
}
func health(value string) string {
	switch strings.ToLower(value) {
	case "healthy":
		return "Healthy"
	case "unhealthy":
		return "Unhealthy"
	case "starting":
		return "Starting"
	case "", "none":
		return "None"
	default:
		return "Unknown"
	}
}
func healthFromText(value string) string {
	lower := strings.ToLower(value)
	if strings.Contains(lower, "(unhealthy)") {
		return "Unhealthy"
	}
	if strings.Contains(lower, "(healthy)") {
		return "Healthy"
	}
	if strings.Contains(lower, "(health: starting)") {
		return "Starting"
	}
	return "None"
}
func timestamp(value string) *time.Time {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil || parsed.Year() <= 1 {
		return nil
	}
	return &parsed
}
func shortID(value string) string {
	if len(value) > 12 {
		return value[:12]
	}
	return value
}
func subtract(value, deduction uint64) uint64 {
	if value >= deduction {
		return value - deduction
	}
	return 0
}

func (h *Handler) unavailable(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, context.Canceled) {
		return
	}
	h.logger.Error("Docker operation failed", "request_id", middleware.GetReqID(r.Context()), "error", err)
	writeProblem(w, http.StatusServiceUnavailable, "Infrastructure unavailable.", "Docker Engine is unavailable.")
}

func (h *Handler) dockerError(w http.ResponseWriter, r *http.Request, err error) {
	if errdefs.IsNotFound(err) {
		writeProblem(w, http.StatusNotFound, "Container not found.", "The requested container does not exist.")
		return
	}
	h.unavailable(w, r, err)
}

func (h *Handler) internalError(w http.ResponseWriter, r *http.Request, err error) {
	h.logger.Error("container request failed", "request_id", middleware.GetReqID(r.Context()), "error", err)
	writeProblem(w, http.StatusInternalServerError, "Request failed.", "The server could not complete the request.")
}

func writeProblem(w http.ResponseWriter, status int, title, detail string) {
	w.Header().Set("Content-Type", "application/problem+json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"type": "about:blank", "title": title, "detail": detail, "status": status})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
