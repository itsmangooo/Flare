package httpapi

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/itsmangooo/flare/internal/config"
)

type databasePinger interface {
	Ping(context.Context) error
}

type healthResponse struct {
	Status string                 `json:"status"`
	Checks map[string]healthCheck `json:"checks"`
}

type healthCheck struct {
	Status string `json:"status"`
}

type Routes struct {
	Auth       http.Handler
	Containers http.Handler
	Activity   http.Handler
	Overview   http.Handler
	Domains    http.Handler
	Coolify    http.Handler
	System     http.Handler
}

func New(cfg config.Config, version string, logger *slog.Logger, database databasePinger, routes Routes) *http.Server {
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Recoverer)
	router.Use(middleware.Timeout(30 * time.Second))
	router.Use(requestLogger(logger))
	registerPublicRoutes(router, version, database, time.Now())
	if routes.Auth != nil {
		router.Mount("/api/v1/auth", routes.Auth)
	}
	if routes.Containers != nil {
		router.Mount("/api/v1/containers", routes.Containers)
	}
	if routes.Activity != nil {
		router.Mount("/api/v1/activity", routes.Activity)
	}
	if routes.Overview != nil {
		router.Mount("/api/v1/overview", routes.Overview)
	}
	if routes.Domains != nil {
		router.Mount("/api/v1/domains", routes.Domains)
	}
	if routes.Coolify != nil {
		router.Mount("/api/v1/coolify", routes.Coolify)
	}
	if routes.System != nil {
		router.Mount("/api/v1/system", routes.System)
	}
	router.Get("/health/live", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, healthResponse{Status: "healthy", Checks: map[string]healthCheck{}})
	})
	router.Get("/health/ready", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		if err := database.Ping(ctx); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, healthResponse{
				Status: "unhealthy",
				Checks: map[string]healthCheck{"postgres": {Status: "unhealthy"}},
			})
			return
		}
		writeJSON(w, http.StatusOK, healthResponse{
			Status: "healthy",
			Checks: map[string]healthCheck{"postgres": {Status: "healthy"}},
		})
	})

	return &http.Server{
		Addr:              cfg.HTTPAddress,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      35 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func requestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			started := time.Now()
			next.ServeHTTP(w, r)
			logger.Info("HTTP request", "method", r.Method, "path", r.URL.Path,
				"request_id", middleware.GetReqID(r.Context()), "duration_ms", time.Since(started).Milliseconds())
		})
	}
}
