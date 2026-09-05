package coolify

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type readService interface {
	Servers(context.Context) ([]Server, error)
	ServerResources(context.Context, string) ([]Resource, error)
	Applications(context.Context) ([]Application, error)
	Services(context.Context) ([]Service, error)
	Deployments(context.Context, int, int) (DeploymentPage, error)
	Deployment(context.Context, string) (Deployment, error)
}

type Handler struct {
	service readService
	logger  *slog.Logger
}

func NewHandler(service readService, logger *slog.Logger) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	handler := &Handler{service: service, logger: logger}
	router := chi.NewRouter()
	router.Get("/servers", handler.servers)
	router.Get("/servers/{uuid}/resources", handler.serverResources)
	router.Get("/applications", handler.applications)
	router.Get("/services", handler.services)
	router.Get("/deployments", handler.deployments)
	router.Get("/deployments/{uuid}", handler.deployment)
	return router
}

func (handler *Handler) servers(writer http.ResponseWriter, request *http.Request) {
	items, err := handler.service.Servers(request.Context())
	handler.respond(writer, request, items, err)
}

func (handler *Handler) serverResources(writer http.ResponseWriter, request *http.Request) {
	items, err := handler.service.ServerResources(request.Context(), chi.URLParam(request, "uuid"))
	handler.respond(writer, request, items, err)
}

func (handler *Handler) applications(writer http.ResponseWriter, request *http.Request) {
	items, err := handler.service.Applications(request.Context())
	handler.respond(writer, request, items, err)
}

func (handler *Handler) services(writer http.ResponseWriter, request *http.Request) {
	items, err := handler.service.Services(request.Context())
	handler.respond(writer, request, items, err)
}

func (handler *Handler) deployments(writer http.ResponseWriter, request *http.Request) {
	page, pageSize, err := coolifyPagination(request)
	if err != nil {
		writeProblem(writer, http.StatusBadRequest, "Invalid request.", "Page and pageSize must be integers in the supported range.")
		return
	}
	items, err := handler.service.Deployments(request.Context(), page, pageSize)
	handler.respond(writer, request, items, err)
}

func (handler *Handler) deployment(writer http.ResponseWriter, request *http.Request) {
	item, err := handler.service.Deployment(request.Context(), chi.URLParam(request, "uuid"))
	handler.respond(writer, request, item, err)
}

func (handler *Handler) respond(writer http.ResponseWriter, request *http.Request, value any, err error) {
	if err == nil {
		writeJSON(writer, http.StatusOK, value)
		return
	}
	switch {
	case errors.Is(err, context.Canceled):
		return
	case errors.Is(err, ErrInvalidIdentifier):
		writeProblem(writer, http.StatusBadRequest, "Invalid request.", "Coolify identifier is invalid.")
	case errors.Is(err, ErrNotFound):
		writeProblem(writer, http.StatusNotFound, "Resource not found.", "The requested Coolify resource does not exist.")
	case errors.Is(err, ErrNotConfigured):
		writeProblem(writer, http.StatusServiceUnavailable, "Integration not configured.", "Coolify integration is not configured.")
	default:
		handler.logger.Warn("Coolify request failed", "request_id", middleware.GetReqID(request.Context()))
		writeProblem(writer, http.StatusServiceUnavailable, "Infrastructure unavailable.", "Coolify is unavailable.")
	}
}

func coolifyPagination(request *http.Request) (int, int, error) {
	page, pageSize := 1, 30
	var err error
	if raw := request.URL.Query().Get("page"); raw != "" {
		page, err = strconv.Atoi(raw)
		if err != nil || page > 1_000_000 {
			return 0, 0, errors.New("page is invalid")
		}
	}
	if raw := request.URL.Query().Get("pageSize"); raw != "" {
		pageSize, err = strconv.Atoi(raw)
		if err != nil {
			return 0, 0, errors.New("pageSize is invalid")
		}
	}
	return max(page, 1), min(max(pageSize, 1), 50), nil
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
