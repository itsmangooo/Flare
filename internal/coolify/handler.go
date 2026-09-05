package coolify

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/itsmangooo/flare/internal/auth"
	"github.com/jackc/pgx/v5/pgconn"
)

type readService interface {
	Servers(context.Context) ([]Server, error)
	ServerResources(context.Context, string) ([]Resource, error)
	Applications(context.Context) ([]Application, error)
	Services(context.Context) ([]Service, error)
	Deployments(context.Context, int, int) (DeploymentPage, error)
	Deployment(context.Context, string) (Deployment, error)
	StartApplication(context.Context, string) (ActionResponse, error)
	StopApplication(context.Context, string) (ActionResponse, error)
	RestartApplication(context.Context, string) (ActionResponse, error)
	RestartService(context.Context, string) (ActionResponse, error)
	RedeployApplication(context.Context, string) (ActionResponse, error)
}

type auditDatabase interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

type Handler struct {
	service readService
	audits  auditDatabase
	logger  *slog.Logger
	limiter *actionLimiter
}

type actionWindow struct {
	started  time.Time
	requests int
}

type actionLimiter struct {
	mu      sync.Mutex
	windows map[string]actionWindow
}

func NewHandler(service readService, audits auditDatabase, logger *slog.Logger) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	handler := &Handler{service: service, audits: audits, logger: logger,
		limiter: &actionLimiter{windows: make(map[string]actionWindow)}}
	router := chi.NewRouter()
	router.Get("/servers", handler.servers)
	router.Get("/servers/{uuid}/resources", handler.serverResources)
	router.Get("/applications", handler.applications)
	router.Get("/services", handler.services)
	router.Get("/deployments", handler.deployments)
	router.Get("/deployments/{uuid}", handler.deployment)
	router.Post("/applications/{uuid}/start", handler.startApplication)
	router.Post("/applications/{uuid}/stop", handler.stopApplication)
	router.Post("/applications/{uuid}/restart", handler.restartApplication)
	router.Post("/applications/{uuid}/redeploy", handler.redeployApplication)
	router.Post("/services/{uuid}/restart", handler.restartService)
	return router
}

func (handler *Handler) startApplication(writer http.ResponseWriter, request *http.Request) {
	handler.perform(writer, request, "coolify.application.start", handler.service.StartApplication)
}

func (handler *Handler) stopApplication(writer http.ResponseWriter, request *http.Request) {
	handler.perform(writer, request, "coolify.application.stop", handler.service.StopApplication)
}

func (handler *Handler) restartApplication(writer http.ResponseWriter, request *http.Request) {
	handler.perform(writer, request, "coolify.application.restart", handler.service.RestartApplication)
}

func (handler *Handler) redeployApplication(writer http.ResponseWriter, request *http.Request) {
	handler.perform(writer, request, "coolify.application.redeploy", handler.service.RedeployApplication)
}

func (handler *Handler) restartService(writer http.ResponseWriter, request *http.Request) {
	handler.perform(writer, request, "coolify.service.restart", handler.service.RestartService)
}

func (handler *Handler) perform(writer http.ResponseWriter, request *http.Request, action string,
	operation func(context.Context, string) (ActionResponse, error)) {
	user, authenticated := auth.UserFromContext(request.Context())
	if !authenticated || !auth.HasRole(request.Context(), "Administrator") {
		writeProblem(writer, http.StatusForbidden, "Forbidden.", "Administrator access is required.")
		return
	}
	if handler.audits == nil {
		handler.internalError(writer, request, errors.New("audit database is unavailable"))
		return
	}
	if !handler.limiter.allow(user.ID.String()+":"+request.RemoteAddr, time.Now()) {
		writer.Header().Set("Retry-After", "60")
		writeProblem(writer, http.StatusTooManyRequests, "Too many requests.", "Try again later.")
		return
	}
	target := chi.URLParam(request, "uuid")
	response, err := operation(request.Context(), target)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return
		}
		if auditErr := handler.recordAudit(request.Context(), user, action, target, 1, "Coolify operation failed."); auditErr != nil {
			handler.internalError(writer, request, auditErr)
			return
		}
		handler.respond(writer, request, nil, err)
		return
	}
	if err := handler.recordAudit(request.Context(), user, action, target, 0, ""); err != nil {
		handler.internalError(writer, request, err)
		return
	}
	writeJSON(writer, http.StatusAccepted, response)
}

func (handler *Handler) recordAudit(ctx context.Context, user auth.User, action, target string, result int, detail string) error {
	var detailValue any
	if detail != "" {
		detailValue = detail
	}
	_, err := handler.audits.Exec(ctx, `INSERT INTO "AuditEvents" ("Id","UserId","Actor","Action","Target","Timestamp","Result","CorrelationId","Detail") VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		uuid.New(), user.ID, user.Email, action, target, time.Now().UTC(), result, middleware.GetReqID(ctx), detailValue)
	return err
}

func (handler *Handler) internalError(writer http.ResponseWriter, request *http.Request, err error) {
	handler.logger.Error("Coolify operation could not be audited", "request_id", middleware.GetReqID(request.Context()), "error", err)
	writeProblem(writer, http.StatusInternalServerError, "Request failed.", "The operation could not be completed.")
}

func (limiter *actionLimiter) allow(key string, now time.Time) bool {
	limiter.mu.Lock()
	defer limiter.mu.Unlock()
	window := limiter.windows[key]
	if window.started.IsZero() || now.Sub(window.started) >= time.Minute {
		window = actionWindow{started: now}
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
