package overview

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/itsmangooo/flare/internal/activity"
	"github.com/itsmangooo/flare/internal/telemetry"
	"github.com/jackc/pgx/v5"
	containertypes "github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

const historyQuery = `SELECT "Timestamp", "CpuPercent", "MemoryPercent"
FROM "MetricSamples"
WHERE "Timestamp" >= $1
ORDER BY "Timestamp"
LIMIT 3600`

type dockerAPI interface {
	ContainerList(context.Context, client.ContainerListOptions) (client.ContainerListResult, error)
}

type metricsCollector interface {
	Collect(context.Context) telemetry.Metrics
}

type database interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

type Handler struct {
	docker   dockerAPI
	metrics  metricsCollector
	database database
	activity *activity.Reader
	logger   *slog.Logger
	now      func() time.Time
}

type response struct {
	GeneratedAt    time.Time         `json:"generatedAt"`
	Freshness      string            `json:"freshness"`
	Host           telemetry.Metrics `json:"host"`
	Containers     containerTotals   `json:"containers"`
	History        []metricPoint     `json:"history"`
	RecentActivity []activity.Event  `json:"recentActivity"`
}

type containerTotals struct {
	Running   *int `json:"running"`
	Stopped   *int `json:"stopped"`
	Unhealthy *int `json:"unhealthy"`
}

type metricPoint struct {
	Timestamp     time.Time `json:"timestamp"`
	CPUPercent    *float64  `json:"cpuPercent"`
	MemoryPercent *float64  `json:"memoryPercent"`
}

func NewHandler(docker dockerAPI, metrics metricsCollector, database database, activityReader *activity.Reader, logger *slog.Logger) http.Handler {
	return newHandler(docker, metrics, database, activityReader, logger, time.Now)
}

func newHandler(docker dockerAPI, metrics metricsCollector, database database, activityReader *activity.Reader, logger *slog.Logger, now func() time.Time) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{docker: docker, metrics: metrics, database: database, activity: activityReader, logger: logger, now: now}
}

func (handler *Handler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writer.Header().Set("Allow", http.MethodGet)
		writeProblem(writer, http.StatusMethodNotAllowed, "Method not allowed.", "")
		return
	}
	now := handler.now().UTC()
	host := handler.metrics.Collect(request.Context())
	containers, dockerAvailable := handler.containerTotals(request.Context())
	history, err := handler.history(request.Context(), now.Add(-time.Hour))
	if errors.Is(err, context.Canceled) {
		return
	}
	if err != nil {
		handler.internalError(writer, request, err)
		return
	}
	recent, _, err := handler.activity.Read(request.Context(), 1, 5)
	if errors.Is(err, context.Canceled) {
		return
	}
	if err != nil {
		handler.internalError(writer, request, err)
		return
	}
	freshness := "Offline"
	if dockerAvailable || hostAvailable(host) {
		freshness = "Live"
	}
	writeJSON(writer, http.StatusOK, response{
		GeneratedAt: now, Freshness: freshness, Host: host, Containers: containers,
		History: history, RecentActivity: recent,
	})
}

func (handler *Handler) containerTotals(ctx context.Context) (containerTotals, bool) {
	listed, err := handler.docker.ContainerList(ctx, client.ContainerListOptions{All: true})
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			handler.logger.Warn("Docker overview unavailable", "error", err)
		}
		return containerTotals{}, false
	}
	running, stopped, unhealthy := 0, 0, 0
	for _, item := range listed.Items {
		if strings.EqualFold(string(item.State), "running") {
			running++
		} else {
			stopped++
		}
		if unhealthyContainer(item) {
			unhealthy++
		}
	}
	return containerTotals{Running: &running, Stopped: &stopped, Unhealthy: &unhealthy}, true
}

func unhealthyContainer(item containertypes.Summary) bool {
	return strings.Contains(strings.ToLower(item.Status), "(unhealthy)")
}

func (handler *Handler) history(ctx context.Context, cutoff time.Time) ([]metricPoint, error) {
	rows, err := handler.database.Query(ctx, historyQuery, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]metricPoint, 0)
	for rows.Next() {
		var item metricPoint
		var cpu, memory sql.NullFloat64
		if err := rows.Scan(&item.Timestamp, &cpu, &memory); err != nil {
			return nil, err
		}
		item.Timestamp = item.Timestamp.UTC()
		if cpu.Valid {
			item.CPUPercent = &cpu.Float64
		}
		if memory.Valid {
			item.MemoryPercent = &memory.Float64
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func hostAvailable(metrics telemetry.Metrics) bool {
	return metrics.CPUPercent != nil || metrics.MemoryTotalBytes != nil || metrics.Uptime != nil
}

func (handler *Handler) internalError(writer http.ResponseWriter, request *http.Request, err error) {
	handler.logger.Error("Overview query failed", "request_id", middleware.GetReqID(request.Context()), "error", err)
	writeProblem(writer, http.StatusInternalServerError, "Request failed.", "The overview could not be loaded.")
}

func writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.Header().Set("X-Content-Type-Options", "nosniff")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}

func writeProblem(writer http.ResponseWriter, status int, title, detail string) {
	value := map[string]any{"type": "about:blank", "title": title, "status": status}
	if detail != "" {
		value["detail"] = detail
	}
	writer.Header().Set("Content-Type", "application/problem+json; charset=utf-8")
	writer.Header().Set("X-Content-Type-Options", "nosniff")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}
