package monitoring

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/itsmangooo/flare/server/backend/internal/alerts"
	"github.com/itsmangooo/flare/server/backend/internal/cloudflare"
)

const cloudflarePollInterval = time.Minute

type cloudflareMonitoringClient interface {
	Configured() bool
	TunnelsConfigured() bool
	Zones(context.Context) ([]cloudflare.Zone, error)
	Tunnels(context.Context) ([]cloudflare.Tunnel, error)
}

type CloudflareMonitor struct {
	client            cloudflareMonitoringClient
	database          eventDatabase
	alerts            alertSink
	logger            *slog.Logger
	now               func() time.Time
	interval          time.Duration
	availabilityKnown bool
	available         bool
	unhealthyTunnels  map[string]bool
}

func NewCloudflareMonitor(client cloudflareMonitoringClient, database eventDatabase, logger *slog.Logger, sink alertSink) *CloudflareMonitor {
	if logger == nil {
		logger = slog.Default()
	}
	return &CloudflareMonitor{
		client: client, database: database, alerts: sink, logger: logger,
		now: time.Now, interval: cloudflarePollInterval, unhealthyTunnels: make(map[string]bool),
	}
}

func (monitor *CloudflareMonitor) Run(ctx context.Context) {
	if monitor == nil || monitor.client == nil || !monitor.client.Configured() || monitor.database == nil || monitor.alerts == nil {
		return
	}
	monitor.pollAndLog(ctx)
	ticker := time.NewTicker(monitor.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			monitor.pollAndLog(ctx)
		}
	}
}

func (monitor *CloudflareMonitor) pollAndLog(ctx context.Context) {
	if err := monitor.poll(ctx); err != nil && !errors.Is(err, context.Canceled) {
		monitor.logger.Warn("Cloudflare monitoring poll failed")
	}
}

func (monitor *CloudflareMonitor) poll(ctx context.Context) error {
	if _, err := monitor.client.Zones(ctx); err != nil {
		monitor.markCloudflareUnavailable(ctx)
		return err
	}
	var tunnels []cloudflare.Tunnel
	if monitor.client.TunnelsConfigured() {
		var err error
		tunnels, err = monitor.client.Tunnels(ctx)
		if err != nil {
			monitor.markCloudflareUnavailable(ctx)
			return err
		}
	}
	monitor.markCloudflareAvailable(ctx)
	monitor.processTunnels(ctx, tunnels)
	return nil
}

func (monitor *CloudflareMonitor) markCloudflareUnavailable(ctx context.Context) {
	if !monitor.availabilityKnown || monitor.available {
		now := monitor.now().UTC()
		monitor.recordCloudflare(ctx, infrastructureEvent{
			action: "cloudflare.disconnected", target: "Cloudflare", failed: true,
			detail: "Cloudflare API is unavailable.", timestamp: now,
		}, alerts.Signal{
			Fingerprint: "cloudflare.availability", Kind: "cloudflare.unavailable", Severity: "critical",
			Title: "Cloudflare unavailable", Message: "The Cloudflare API could not be reached.", Source: "cloudflare",
			ResourceType: "integration", ResourceID: "cloudflare", OccurredAt: now,
		})
	}
	monitor.availabilityKnown = true
	monitor.available = false
}

func (monitor *CloudflareMonitor) markCloudflareAvailable(ctx context.Context) {
	if monitor.availabilityKnown && !monitor.available {
		now := monitor.now().UTC()
		monitor.recordCloudflare(ctx, infrastructureEvent{
			action: "cloudflare.reconnected", target: "Cloudflare",
			detail: "Cloudflare API connectivity recovered.", timestamp: now,
		}, alerts.Signal{
			Fingerprint: "cloudflare.availability", Kind: "cloudflare.unavailable", Severity: "info",
			Title: "Cloudflare reconnected", Message: "The Cloudflare API is reachable again.", Source: "cloudflare",
			ResourceType: "integration", ResourceID: "cloudflare", OccurredAt: now, Recovery: true,
		})
	}
	monitor.availabilityKnown = true
	monitor.available = true
}

func (monitor *CloudflareMonitor) processTunnels(ctx context.Context, tunnels []cloudflare.Tunnel) {
	for _, tunnel := range tunnels {
		unavailable, known := tunnelUnavailable(tunnel.Status)
		if tunnel.ID == "" || !known {
			continue
		}
		active := monitor.unhealthyTunnels[tunnel.ID]
		if unavailable == active {
			continue
		}
		if monitor.recordTunnel(ctx, tunnel, !unavailable) {
			if unavailable {
				monitor.unhealthyTunnels[tunnel.ID] = true
			} else {
				delete(monitor.unhealthyTunnels, tunnel.ID)
			}
		}
	}
}

func (monitor *CloudflareMonitor) recordTunnel(ctx context.Context, tunnel cloudflare.Tunnel, recovery bool) bool {
	now := monitor.now().UTC()
	target := strings.TrimSpace(tunnel.Name)
	if target == "" {
		target = "Cloudflare Tunnel"
	}
	event := infrastructureEvent{
		action: "cloudflare.tunnel.down", target: target, failed: true,
		detail: "Cloudflare reported the tunnel as unavailable.", timestamp: now,
		resourceType: "cloudflare_tunnel", resourceID: tunnel.ID,
	}
	signal := alerts.Signal{
		Fingerprint: "cloudflare.tunnel:" + tunnel.ID, Kind: "cloudflare.tunnel_unavailable", Severity: "critical",
		Title: "Tunnel unavailable", Message: target + " is not healthy.", Source: "cloudflare",
		ResourceType: "cloudflare_tunnel", ResourceID: tunnel.ID, OccurredAt: now,
	}
	if recovery {
		event.action = "cloudflare.tunnel.recovered"
		event.failed = false
		event.detail = "Cloudflare reported the tunnel as healthy."
		signal.Severity = "info"
		signal.Title = "Tunnel recovered"
		signal.Message = target + " is healthy again."
		signal.Recovery = true
	}
	return monitor.recordCloudflare(ctx, event, signal)
}

func (monitor *CloudflareMonitor) recordCloudflare(ctx context.Context, event infrastructureEvent, signal alerts.Signal) bool {
	result := 0
	if event.failed {
		result = 1
	}
	_, err := monitor.database.Exec(ctx, `INSERT INTO "InfrastructureEvents" ("Id","Action","Target","Timestamp","Result","Detail") VALUES ($1,$2,$3,$4,$5,$6)`,
		uuid.New(), event.action, truncate(event.target, 256), event.timestamp.UTC(), result, truncate(event.detail, 1000))
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			monitor.logger.Error("Cloudflare infrastructure event could not be recorded", "action", event.action)
		}
		return false
	}
	if err = monitor.alerts.Record(ctx, signal); err != nil {
		if !errors.Is(err, context.Canceled) {
			monitor.logger.Error("Cloudflare alert could not be recorded", "kind", signal.Kind)
		}
		return false
	}
	return true
}

func tunnelUnavailable(status string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "healthy":
		return false, true
	case "down", "degraded", "inactive", "unhealthy":
		return true, true
	default:
		return false, false
	}
}
