package monitoring

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/itsmangooo/flare/internal/alerts"
	"github.com/itsmangooo/flare/internal/coolify"
)

const (
	coolifyPollInterval = 30 * time.Second
	recentDeploymentAge = 5 * time.Minute
	maximumDeployments  = 4096
)

type coolifyMonitoringClient interface {
	Configured() bool
	Servers(context.Context) ([]coolify.Server, error)
	Deployments(context.Context, int, int) (coolify.DeploymentPage, error)
}

type CoolifyMonitor struct {
	client            coolifyMonitoringClient
	database          eventDatabase
	alerts            alertSink
	logger            *slog.Logger
	now               func() time.Time
	interval          time.Duration
	availabilityKnown bool
	available         bool
	seeded            bool
	seen              map[string]struct{}
	seenOrder         []string
	failures          map[string]bool
}

func NewCoolifyMonitor(client coolifyMonitoringClient, database eventDatabase, logger *slog.Logger, sink alertSink) *CoolifyMonitor {
	if logger == nil {
		logger = slog.Default()
	}
	return &CoolifyMonitor{
		client: client, database: database, alerts: sink, logger: logger,
		now: time.Now, interval: coolifyPollInterval, seen: make(map[string]struct{}), failures: make(map[string]bool),
	}
}

func (monitor *CoolifyMonitor) Run(ctx context.Context) {
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

func (monitor *CoolifyMonitor) pollAndLog(ctx context.Context) {
	if err := monitor.poll(ctx); err != nil && !errors.Is(err, context.Canceled) {
		monitor.logger.Warn("Coolify monitoring poll failed")
	}
}

func (monitor *CoolifyMonitor) poll(ctx context.Context) error {
	if _, err := monitor.client.Servers(ctx); err != nil {
		monitor.markUnavailable(ctx)
		return err
	}
	deployments, err := monitor.client.Deployments(ctx, 1, 50)
	if err != nil {
		monitor.markUnavailable(ctx)
		return err
	}
	monitor.markAvailable(ctx)
	monitor.processDeployments(ctx, deployments.Items)
	return nil
}

func (monitor *CoolifyMonitor) markUnavailable(ctx context.Context) {
	if !monitor.availabilityKnown || monitor.available {
		monitor.record(ctx, infrastructureEvent{
			action: "coolify.disconnected", target: "Coolify", failed: true,
			detail: "Coolify API is unavailable.", timestamp: monitor.now().UTC(),
		}, alerts.Signal{
			Fingerprint: "coolify.availability", Kind: "coolify.unavailable", Severity: "critical",
			Title: "Coolify unavailable", Message: "The Coolify API could not be reached.", Source: "coolify",
			ResourceType: "integration", ResourceID: "coolify", OccurredAt: monitor.now().UTC(),
		})
	}
	monitor.availabilityKnown = true
	monitor.available = false
}

func (monitor *CoolifyMonitor) markAvailable(ctx context.Context) {
	if monitor.availabilityKnown && !monitor.available {
		monitor.record(ctx, infrastructureEvent{
			action: "coolify.reconnected", target: "Coolify",
			detail: "Coolify API connectivity recovered.", timestamp: monitor.now().UTC(),
		}, alerts.Signal{
			Fingerprint: "coolify.availability", Kind: "coolify.unavailable", Severity: "info",
			Title: "Coolify reconnected", Message: "The Coolify API is reachable again.", Source: "coolify",
			ResourceType: "integration", ResourceID: "coolify", OccurredAt: monitor.now().UTC(), Recovery: true,
		})
	}
	monitor.availabilityKnown = true
	monitor.available = true
}

func (monitor *CoolifyMonitor) processDeployments(ctx context.Context, deployments []coolify.Deployment) {
	if !monitor.seeded {
		monitor.seedDeployments(ctx, deployments)
		return
	}
	for index := len(deployments) - 1; index >= 0; index-- {
		deployment := deployments[index]
		if deployment.UUID == "" || !monitor.remember(deployment.UUID) {
			continue
		}
		status := deploymentStatus(deployment.Status)
		switch status {
		case "failed":
			monitor.recordDeployment(ctx, deployment, false)
		case "succeeded":
			monitor.recordDeployment(ctx, deployment, true)
		}
	}
}

func (monitor *CoolifyMonitor) seedDeployments(ctx context.Context, deployments []coolify.Deployment) {
	latest := make(map[string]struct{})
	cutoff := monitor.now().Add(-recentDeploymentAge)
	for _, deployment := range deployments {
		if deployment.UUID == "" {
			continue
		}
		monitor.remember(deployment.UUID)
		status := deploymentStatus(deployment.Status)
		if status == "" {
			continue
		}
		fingerprint := deploymentFingerprint(deployment)
		if _, found := latest[fingerprint]; found {
			continue
		}
		latest[fingerprint] = struct{}{}
		if status == "failed" && !deploymentTime(deployment, monitor.now()).Before(cutoff) {
			monitor.recordDeployment(ctx, deployment, false)
		}
	}
	monitor.seeded = true
}

func (monitor *CoolifyMonitor) remember(id string) bool {
	if _, found := monitor.seen[id]; found {
		return false
	}
	monitor.seen[id] = struct{}{}
	monitor.seenOrder = append(monitor.seenOrder, id)
	if len(monitor.seenOrder) > maximumDeployments {
		oldest := monitor.seenOrder[0]
		monitor.seenOrder = monitor.seenOrder[1:]
		delete(monitor.seen, oldest)
	}
	return true
}

func (monitor *CoolifyMonitor) recordDeployment(ctx context.Context, deployment coolify.Deployment, recovery bool) {
	resourceID := deployment.ResourceUUID
	resourceType := "coolify_application"
	fingerprint := deploymentFingerprint(deployment)
	if resourceID == "" {
		resourceID = deployment.UUID
		resourceType = "coolify_deployment"
	}
	if recovery && !monitor.failures[fingerprint] {
		return
	}
	timestamp := deploymentTime(deployment, monitor.now())
	target := strings.TrimSpace(deployment.ResourceName)
	if target == "" {
		target = "Coolify application"
	}
	event := infrastructureEvent{
		action: "coolify.deployment_failed", target: target, failed: true,
		detail: "Coolify reported a failed deployment.", timestamp: timestamp,
		resourceType: resourceType, resourceID: resourceID,
	}
	signal := alerts.Signal{
		Fingerprint: fingerprint, Kind: "coolify.deployment_failed", Severity: "critical",
		Title: "Deployment failed", Message: target + " failed to deploy.", Source: "coolify",
		ResourceType: resourceType, ResourceID: resourceID, OccurredAt: timestamp,
	}
	if recovery {
		event.action = "coolify.deployment_recovered"
		event.failed = false
		event.detail = "A later Coolify deployment succeeded."
		signal.Severity = "info"
		signal.Title = "Deployment recovered"
		signal.Message = target + " deployed successfully."
		signal.Recovery = true
	}
	if monitor.record(ctx, event, signal) {
		if recovery {
			delete(monitor.failures, fingerprint)
		} else {
			monitor.failures[fingerprint] = true
		}
	}
}

func deploymentFingerprint(deployment coolify.Deployment) string {
	if deployment.ResourceUUID != "" {
		return "coolify.deployment:" + deployment.ResourceUUID
	}
	return "coolify.deployment:" + deployment.UUID
}

func (monitor *CoolifyMonitor) record(ctx context.Context, event infrastructureEvent, signal alerts.Signal) bool {
	result := 0
	if event.failed {
		result = 1
	}
	_, err := monitor.database.Exec(ctx, `INSERT INTO "InfrastructureEvents" ("Id","Action","Target","Timestamp","Result","Detail") VALUES ($1,$2,$3,$4,$5,$6)`,
		uuid.New(), event.action, truncate(event.target, 256), event.timestamp.UTC(), result, truncate(event.detail, 1000))
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			monitor.logger.Error("Coolify infrastructure event could not be recorded", "action", event.action)
		}
		return false
	}
	if err = monitor.alerts.Record(ctx, signal); err != nil && !errors.Is(err, context.Canceled) {
		monitor.logger.Error("Coolify alert could not be recorded", "kind", signal.Kind)
		return false
	}
	return err == nil
}

func deploymentStatus(value *string) string {
	if value == nil {
		return ""
	}
	switch strings.ToLower(strings.TrimSpace(*value)) {
	case "failed", "error":
		return "failed"
	case "finished", "succeeded", "success", "completed":
		return "succeeded"
	default:
		return ""
	}
}

func deploymentTime(deployment coolify.Deployment, fallback time.Time) time.Time {
	if deployment.FinishedAt != nil {
		return deployment.FinishedAt.UTC()
	}
	if deployment.StartedAt != nil {
		return deployment.StartedAt.UTC()
	}
	return fallback.UTC()
}
