package containers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
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
	containertypes "github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

type dockerAPI interface {
	ContainerList(context.Context, client.ContainerListOptions) (client.ContainerListResult, error)
	ContainerInspect(context.Context, string, client.ContainerInspectOptions) (client.ContainerInspectResult, error)
	ContainerStats(context.Context, string, client.ContainerStatsOptions) (client.ContainerStatsResult, error)
}

type Handler struct {
	docker dockerAPI
	logger *slog.Logger
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

var containerIDPattern = regexp.MustCompile(`^[a-fA-F0-9]{12,64}$`)
var sensitiveLabelPattern = regexp.MustCompile(`(?i)token|secret|pass(word|wd)?|credential|private[-_. ]?key|api[-_. ]?key`)

func NewDockerClient(host string) (*client.Client, error) {
	return client.NewClientWithOpts(client.WithHost(host), client.WithAPIVersionNegotiation())
}

func NewHandler(docker dockerAPI, logger *slog.Logger) http.Handler {
	h := &Handler{docker: docker, logger: logger}
	router := chi.NewRouter()
	router.Get("/", h.list)
	router.Get("/{id}", h.detail)
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
