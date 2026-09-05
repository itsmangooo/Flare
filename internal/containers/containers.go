package containers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

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

func NewDockerClient(host string) (*client.Client, error) {
	return client.NewClientWithOpts(client.WithHost(host), client.WithAPIVersionNegotiation())
}

func NewHandler(docker dockerAPI, logger *slog.Logger) http.Handler {
	h := &Handler{docker: docker, logger: logger}
	router := chi.NewRouter()
	router.Get("/", h.list)
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
		statsCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
		stats, statsErr := h.docker.ContainerStats(statsCtx, item.ID, client.ContainerStatsOptions{Stream: false, IncludePreviousSample: true})
		if statsErr == nil {
			defer stats.Body.Close()
			var value containertypes.StatsResponse
			if json.NewDecoder(io.LimitReader(stats.Body, 2<<20)).Decode(&value) == nil {
				cpuDelta := subtract(value.CPUStats.CPUUsage.TotalUsage, value.PreCPUStats.CPUUsage.TotalUsage)
				systemDelta := subtract(value.CPUStats.SystemUsage, value.PreCPUStats.SystemUsage)
				cpus := value.CPUStats.OnlineCPUs
				if cpus == 0 {
					cpus = uint32(max(len(value.CPUStats.CPUUsage.PercpuUsage), 1))
				}
				if systemDelta > 0 {
					cpu := float64(cpuDelta) / float64(systemDelta) * float64(cpus) * 100
					result.CPUPercent = &cpu
				}
				cache := value.MemoryStats.Stats["inactive_file"]
				if cache == 0 {
					cache = value.MemoryStats.Stats["cache"]
				}
				memory := subtract(value.MemoryStats.Usage, cache)
				if memory <= uint64(^uint64(0)>>1) {
					converted := int64(memory)
					result.MemoryBytes = &converted
				}
			}
		}
	}
	return result
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
	h.logger.Error("Docker container list failed", "request_id", middleware.GetReqID(r.Context()), "error", err)
	w.Header().Set("Content-Type", "application/problem+json; charset=utf-8")
	w.WriteHeader(http.StatusServiceUnavailable)
	_ = json.NewEncoder(w).Encode(map[string]any{"type": "about:blank", "title": "Infrastructure unavailable.", "detail": "Docker Engine is unavailable.", "status": http.StatusServiceUnavailable})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
