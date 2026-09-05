package cloudflare

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type domainService interface {
	Configured() bool
	TunnelsConfigured() bool
	Zones(context.Context) ([]Zone, error)
	DNSRecords(context.Context, string) ([]DNSRecord, error)
	Tunnels(context.Context) ([]Tunnel, error)
	TunnelRoutes(context.Context, string) ([]TunnelRoute, error)
}

type Handler struct {
	service domainService
	logger  *slog.Logger
}

type integrationStatus struct {
	Configured        bool `json:"configured"`
	TunnelsConfigured bool `json:"tunnelsConfigured"`
}

func NewHandler(service domainService, logger *slog.Logger) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	handler := &Handler{service: service, logger: logger}
	router := chi.NewRouter()
	router.Get("/status", handler.status)
	router.Get("/zones", handler.zones)
	router.Get("/zones/{zoneID}/records", handler.records)
	router.Get("/tunnels", handler.tunnels)
	router.Get("/tunnels/{tunnelID}/routes", handler.tunnelRoutes)
	return router
}

func (handler *Handler) status(writer http.ResponseWriter, _ *http.Request) {
	writeCloudflareJSON(writer, http.StatusOK, integrationStatus{
		Configured: handler.service.Configured(), TunnelsConfigured: handler.service.TunnelsConfigured(),
	})
}

func (handler *Handler) zones(writer http.ResponseWriter, request *http.Request) {
	items, err := handler.service.Zones(request.Context())
	handler.respond(writer, request, items, err)
}

func (handler *Handler) records(writer http.ResponseWriter, request *http.Request) {
	items, err := handler.service.DNSRecords(request.Context(), chi.URLParam(request, "zoneID"))
	handler.respond(writer, request, items, err)
}

func (handler *Handler) tunnels(writer http.ResponseWriter, request *http.Request) {
	items, err := handler.service.Tunnels(request.Context())
	handler.respond(writer, request, items, err)
}

func (handler *Handler) tunnelRoutes(writer http.ResponseWriter, request *http.Request) {
	items, err := handler.service.TunnelRoutes(request.Context(), chi.URLParam(request, "tunnelID"))
	handler.respond(writer, request, items, err)
}

func (handler *Handler) respond(writer http.ResponseWriter, request *http.Request, value any, err error) {
	if err == nil {
		writeCloudflareJSON(writer, http.StatusOK, value)
		return
	}
	switch {
	case errors.Is(err, context.Canceled):
		return
	case errors.Is(err, ErrInvalidIdentifier):
		writeCloudflareProblem(writer, http.StatusBadRequest, "Invalid request.", "Cloudflare identifier is invalid.")
	case errors.Is(err, ErrNotFound):
		writeCloudflareProblem(writer, http.StatusNotFound, "Resource not found.", "The requested Cloudflare resource does not exist.")
	case errors.Is(err, ErrNotConfigured):
		writeCloudflareProblem(writer, http.StatusServiceUnavailable, "Integration not configured.", "Cloudflare integration is not configured.")
	default:
		handler.logger.Warn("Cloudflare request failed", "request_id", middleware.GetReqID(request.Context()))
		writeCloudflareProblem(writer, http.StatusServiceUnavailable, "Infrastructure unavailable.", "Cloudflare is unavailable.")
	}
}

func writeCloudflareJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.Header().Set("X-Content-Type-Options", "nosniff")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}

func writeCloudflareProblem(writer http.ResponseWriter, status int, title, detail string) {
	value := map[string]any{"type": "about:blank", "title": title, "status": status}
	if detail != "" {
		value["detail"] = detail
	}
	writer.Header().Set("Content-Type", "application/problem+json; charset=utf-8")
	writer.Header().Set("X-Content-Type-Options", "nosniff")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}
