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
	"github.com/itsmangooo/flare/internal/alerts"
	"github.com/itsmangooo/flare/internal/auth"
	"github.com/itsmangooo/flare/internal/cloudflare"
	"github.com/itsmangooo/flare/internal/config"
	"github.com/itsmangooo/flare/internal/containers"
	"github.com/itsmangooo/flare/internal/coolify"
	"github.com/itsmangooo/flare/internal/database"
	"github.com/itsmangooo/flare/internal/httpapi"
	"github.com/itsmangooo/flare/internal/monitoring"
	"github.com/itsmangooo/flare/internal/overview"
	"github.com/itsmangooo/flare/internal/realtime"
	"github.com/itsmangooo/flare/internal/systeminfo"
	"github.com/itsmangooo/flare/internal/telemetry"
	"github.com/itsmangooo/flare/internal/topology"
)

var version = "dev"

func main() {
	healthcheck := flag.Bool("healthcheck", false, "probe the local liveness endpoint")
	migrate := flag.Bool("migrate", false, "apply pending PostgreSQL migrations and exit")
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
	if *migrate {
		if err = database.Migrate(ctx, db); err != nil {
			logger.Error("database migration failed", "error", err)
			os.Exit(1)
		}
		logger.Info("database migrations complete")
		return
	}

	docker, err := containers.NewDockerClient(cfg.DockerHost)
	if err != nil {
		logger.Error("Docker client configuration failed", "error", err)
		os.Exit(1)
	}
	defer docker.Close()
	ntfyPublisher, err := alerts.NewNtfyPublisher(cfg.NtfyBaseURL, cfg.NtfyTopic, cfg.NtfyToken)
	if err != nil {
		logger.Error("ntfy configuration failed", "error", err)
		os.Exit(1)
	}
	alertDispatcher := alerts.NewDispatcher(ntfyPublisher, logger)
	go alertDispatcher.Run(ctx)
	go monitoring.NewDockerMonitor(docker, db, logger, alertDispatcher).Run(ctx)
	cloudflareClient, err := cloudflare.NewClient(cfg.CloudflareToken, cfg.CloudflareAccountID, logger)
	if err != nil {
		logger.Error("Cloudflare client configuration failed", "error", err)
		os.Exit(1)
	}
	coolifyClient, err := coolify.NewClient(cfg.CoolifyBaseURL, cfg.CoolifyToken, logger)
	if err != nil {
		logger.Error("Coolify client configuration failed", "error", err)
		os.Exit(1)
	}
	authHandler := auth.NewHandler(cfg, db, logger)
	containerHandler := authHandler.Authenticate(containers.NewHandler(docker, db, logger))
	activityHandler := authHandler.Authenticate(activity.NewHandler(db, logger))
	alertHistoryHandler := authHandler.Authenticate(alerts.NewHistoryHandler(db, logger))
	domainHandler := authHandler.Authenticate(cloudflare.NewHandler(cloudflareClient, logger))
	topologyHandler := authHandler.Authenticate(topology.NewHandler(docker, cfg.HostName, logger))
	coolifyHandler := authHandler.Authenticate(coolify.NewHandler(coolifyClient, db, logger))
	systemHandler := authHandler.Authenticate(systeminfo.NewHandler(version, time.Now))
	activityReader := activity.NewReader(db)
	hostMetrics := telemetry.NewCollector(cfg.HostName, cfg.HostProcPath, cfg.HostRootFSPath, logger)
	metricSampler := telemetry.NewSampler(hostMetrics, db, logger)
	go metricSampler.Run(ctx)
	overviewSource := overview.NewHandler(docker, hostMetrics, db, activityReader, logger)
	overviewHandler := authHandler.Authenticate(overviewSource)
	realtimeHandler := authHandler.Authenticate(realtime.NewHandler(overviewSource, logger))
	server := httpapi.New(cfg, version, logger, db, httpapi.Routes{
		Auth: authHandler, Alerts: alertHistoryHandler, Containers: containerHandler, Activity: activityHandler, Overview: overviewHandler,
		Domains: domainHandler, Topology: topologyHandler, Coolify: coolifyHandler, System: systemHandler,
		Telemetry: realtimeHandler,
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
