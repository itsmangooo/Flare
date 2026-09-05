package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/itsmangooo/flare/internal/activity"
	"github.com/itsmangooo/flare/internal/auth"
	"github.com/itsmangooo/flare/internal/config"
	"github.com/itsmangooo/flare/internal/containers"
	"github.com/itsmangooo/flare/internal/database"
	"github.com/itsmangooo/flare/internal/httpapi"
	"github.com/itsmangooo/flare/internal/monitoring"
	"github.com/itsmangooo/flare/internal/overview"
	"github.com/itsmangooo/flare/internal/telemetry"
)

var version = "dev"

func main() {
	healthcheck := flag.Bool("healthcheck", false, "probe the local liveness endpoint")
	flag.Parse()
	if *healthcheck {
		os.Exit(runHealthcheck())
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, err := config.Load()
	if err != nil {
		logger.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	docker, err := containers.NewDockerClient(cfg.DockerHost)
	if err != nil {
		logger.Error("Docker client configuration failed", "error", err)
		os.Exit(1)
	}
	defer docker.Close()
	go monitoring.NewDockerMonitor(docker, db, logger).Run(ctx)
	authHandler := auth.NewHandler(cfg, db, logger)
	containerHandler := authHandler.Authenticate(containers.NewHandler(docker, db, logger))
	activityHandler := authHandler.Authenticate(activity.NewHandler(db, logger))
	activityReader := activity.NewReader(db)
	hostMetrics := telemetry.NewCollector(cfg.HostName, cfg.HostProcPath, cfg.HostRootFSPath, logger)
	metricSampler := telemetry.NewSampler(hostMetrics, db, logger)
	go metricSampler.Run(ctx)
	overviewHandler := authHandler.Authenticate(overview.NewHandler(docker, hostMetrics, db, activityReader, logger))
	server := httpapi.New(cfg, version, logger, db, httpapi.Routes{
		Auth: authHandler, Containers: containerHandler, Activity: activityHandler, Overview: overviewHandler,
	})
	errCh := make(chan error, 1)
	go func() {
		logger.Info("Flare Go API listening", "address", cfg.HTTPAddress, "version", version)
		errCh <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("graceful shutdown failed", "error", err)
		}
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Error("HTTP server stopped", "error", err)
			os.Exit(1)
		}
	}
}

func runHealthcheck() int {
	client := http.Client{Timeout: 3 * time.Second}
	response, err := client.Get("http://127.0.0.1:8080/health/live")
	if err != nil {
		return 1
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return 1
	}
	return 0
}
