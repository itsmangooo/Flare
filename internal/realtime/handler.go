package realtime

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/itsmangooo/flare/internal/overview"
)

const defaultInterval = 3 * time.Second
const snapshotTimeout = 10 * time.Second

type snapshotSource interface {
	Snapshot(context.Context) (overview.Snapshot, error)
}

type Handler struct {
	source   snapshotSource
	logger   *slog.Logger
	interval time.Duration
}

func NewHandler(source snapshotSource, logger *slog.Logger) http.Handler {
	return newHandler(source, logger, defaultInterval)
}

func newHandler(source snapshotSource, logger *slog.Logger, interval time.Duration) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{source: source, logger: logger, interval: interval}
}

func (handler *Handler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		writer.Header().Set("Allow", http.MethodGet)
		writeProblem(writer, http.StatusMethodNotAllowed, "Method not allowed.")
		return
	}
	flusher, ok := writer.(http.Flusher)
	if !ok {
		writeProblem(writer, http.StatusInternalServerError, "Realtime telemetry is unavailable.")
		return
	}

	snapshot, err := handler.readSnapshot(request.Context())
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			handler.logger.Error("Initial realtime snapshot failed", "request_id", middleware.GetReqID(request.Context()), "error", err)
			writeProblem(writer, http.StatusServiceUnavailable, "Realtime telemetry is unavailable.")
		}
		return
	}

	writer.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	writer.Header().Set("Cache-Control", "no-cache, no-store")
	writer.Header().Set("X-Accel-Buffering", "no")
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write([]byte("retry: 3000\n"))
	if !writeSnapshot(writer, snapshot) {
		return
	}
	flusher.Flush()

	ticker := time.NewTicker(handler.interval)
	defer ticker.Stop()
	for {
		select {
		case <-request.Context().Done():
			return
		case <-ticker.C:
			snapshot, err = handler.readSnapshot(request.Context())
			if err != nil {
				if errors.Is(err, context.Canceled) {
					return
				}
				handler.logger.Warn("Realtime snapshot failed", "request_id", middleware.GetReqID(request.Context()), "error", err)
				if _, err = writer.Write([]byte(": snapshot unavailable\n\n")); err != nil {
					return
				}
			} else if !writeSnapshot(writer, snapshot) {
				return
			}
			flusher.Flush()
		}
	}
}

func (handler *Handler) readSnapshot(parent context.Context) (overview.Snapshot, error) {
	ctx, cancel := context.WithTimeout(parent, snapshotTimeout)
	defer cancel()
	return handler.source.Snapshot(ctx)
}

func writeSnapshot(writer http.ResponseWriter, snapshot overview.Snapshot) bool {
	payload, err := json.Marshal(snapshot)
	if err != nil {
		return false
	}
	message := append([]byte("event: snapshot\ndata: "), payload...)
	message = append(message, '\n', '\n')
	_, err = writer.Write(message)
	return err == nil
}

func writeProblem(writer http.ResponseWriter, status int, title string) {
	writer.Header().Set("Content-Type", "application/problem+json; charset=utf-8")
	writer.Header().Set("X-Content-Type-Options", "nosniff")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(map[string]any{
		"type": "about:blank", "title": title, "status": status,
	})
}
