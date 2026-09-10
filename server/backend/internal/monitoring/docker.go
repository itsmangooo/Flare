package monitoring

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/itsmangooo/flare/server/backend/internal/alerts"
	"github.com/jackc/pgx/v5/pgconn"
	eventtypes "github.com/moby/moby/api/types/events"
	"github.com/moby/moby/client"
)

const (
	restartWindow    = 5 * time.Minute
	restartThreshold = 3
	maximumSeen      = 4096
)

type dockerEvents interface {
	Events(context.Context, client.EventsListOptions) client.EventsResult
}

type eventDatabase interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

type alertSink interface {
	Record(context.Context, alerts.Signal) error
}

type DockerMonitor struct {
	docker      dockerEvents
	database    eventDatabase
	logger      *slog.Logger
	now         func() time.Time
	retryDelay  time.Duration
	maxDelay    time.Duration
	unhealthy   map[string]bool
	deaths      map[string][]time.Time
	loopAlerted map[string]bool
	seen        map[string]struct{}
	alerts      alertSink
}

func NewDockerMonitor(docker dockerEvents, database eventDatabase, logger *slog.Logger, sinks ...alertSink) *DockerMonitor {
	if logger == nil {
		logger = slog.Default()
	}
	monitor := &DockerMonitor{
		docker: docker, database: database, logger: logger, now: time.Now,
		retryDelay: time.Second, maxDelay: 30 * time.Second,
		unhealthy: make(map[string]bool), deaths: make(map[string][]time.Time),
		loopAlerted: make(map[string]bool), seen: make(map[string]struct{}),
	}
	if len(sinks) > 0 {
		monitor.alerts = sinks[0]
	}
	return monitor
}

func (monitor *DockerMonitor) Run(ctx context.Context) {
	if monitor == nil || monitor.docker == nil || monitor.database == nil {
		return
	}
	filters := make(client.Filters).Add("type", string(eventtypes.ContainerEventType))
	var since string
	available := true
	delay := monitor.retryDelay
	for ctx.Err() == nil {
		stream := monitor.docker.Events(ctx, client.EventsListOptions{Since: since, Filters: filters})
		received, err := monitor.consume(ctx, stream, &since, &available)
		if ctx.Err() != nil {
			return
		}
		if received {
			delay = monitor.retryDelay
		}
		if err != nil && available {
			monitor.record(ctx, infrastructureEvent{
				action: "server.disconnected", target: "Docker host", failed: true,
				detail: "Docker event stream unavailable.", timestamp: monitor.now().UTC(),
			})
			available = false
		}
		monitor.logger.Warn("Docker event stream ended; reconnecting")
		if !wait(ctx, delay) {
			return
		}
		delay = min(delay*2, monitor.maxDelay)
	}
}

func (monitor *DockerMonitor) consume(ctx context.Context, stream client.EventsResult, since *string, available *bool) (bool, error) {
	messages, failures := stream.Messages, stream.Err
	received := false
	for messages != nil || failures != nil {
		select {
		case <-ctx.Done():
			return received, ctx.Err()
		case message, open := <-messages:
			if !open {
				messages = nil
				continue
			}
			received = true
			if !*available {
				monitor.record(ctx, infrastructureEvent{action: "server.reconnected", target: "Docker host", timestamp: monitor.eventTime(message)})
				*available = true
			}
			monitor.process(ctx, message)
			if message.Time > 0 {
				*since = strconv.FormatInt(message.Time, 10)
			}
		case err, open := <-failures:
			if !open {
				failures = nil
				continue
			}
			if err != nil {
				return received, err
			}
		}
	}
	return received, io.EOF
}

func (monitor *DockerMonitor) process(ctx context.Context, message eventtypes.Message) {
	if message.Type != eventtypes.ContainerEventType || monitor.duplicate(message) {
		return
	}
	id := message.Actor.ID
	target := strings.TrimSpace(message.Actor.Attributes["name"])
	if target == "" {
		target = shortID(id)
	}
	timestamp := monitor.eventTime(message)
	switch message.Action {
	case eventtypes.ActionDie:
		exitCode, _ := strconv.Atoi(message.Actor.Attributes["exitCode"])
		if exitCode == 0 {
			return
		}
		monitor.record(ctx, infrastructureEvent{action: "container.unexpected_stop", target: target, failed: true,
			detail: fmt.Sprintf("Container exited with code %d.", exitCode), timestamp: timestamp,
			resourceType: "container", resourceID: id})
		monitor.recordDeath(ctx, id, target, timestamp)
	case eventtypes.ActionOOM:
		monitor.record(ctx, infrastructureEvent{action: "container.out_of_memory", target: target, failed: true,
			detail: "Container was terminated by the out-of-memory killer.", timestamp: timestamp,
			resourceType: "container", resourceID: id})
	case eventtypes.ActionHealthStatusUnhealthy:
		if !monitor.unhealthy[id] {
			monitor.record(ctx, infrastructureEvent{action: "container.unhealthy", target: target, failed: true,
				detail: "Docker health check reported unhealthy.", timestamp: timestamp,
				resourceType: "container", resourceID: id})
			monitor.unhealthy[id] = true
		}
	case eventtypes.ActionHealthStatusHealthy:
		if monitor.unhealthy[id] {
			monitor.record(ctx, infrastructureEvent{action: "container.recovered", target: target,
				detail: "Docker health check recovered.", timestamp: timestamp,
				resourceType: "container", resourceID: id})
			delete(monitor.unhealthy, id)
		}
	case eventtypes.ActionDestroy:
		delete(monitor.unhealthy, id)
		delete(monitor.deaths, id)
		delete(monitor.loopAlerted, id)
	}
}

func (monitor *DockerMonitor) recordDeath(ctx context.Context, id, target string, timestamp time.Time) {
	cutoff := timestamp.Add(-restartWindow)
	values := monitor.deaths[id][:0]
	for _, value := range monitor.deaths[id] {
		if !value.Before(cutoff) {
			values = append(values, value)
		}
	}
	values = append(values, timestamp)
	monitor.deaths[id] = values
	if len(values) >= restartThreshold && !monitor.loopAlerted[id] {
		monitor.record(ctx, infrastructureEvent{action: "container.restart_loop", target: target, failed: true,
			detail: "Container exited repeatedly within five minutes.", timestamp: timestamp,
			resourceType: "container", resourceID: id})
		monitor.loopAlerted[id] = true
	}
	if len(values) < restartThreshold {
		monitor.loopAlerted[id] = false
	}
}

func (monitor *DockerMonitor) duplicate(message eventtypes.Message) bool {
	key := fmt.Sprintf("%s:%s:%s:%d", message.Type, message.Action, message.Actor.ID, message.TimeNano)
	if _, found := monitor.seen[key]; found {
		return true
	}
	if len(monitor.seen) >= maximumSeen {
		clear(monitor.seen)
	}
	monitor.seen[key] = struct{}{}
	return false
}

func (monitor *DockerMonitor) eventTime(message eventtypes.Message) time.Time {
	if message.TimeNano > 0 {
		return time.Unix(0, message.TimeNano).UTC()
	}
	if message.Time > 0 {
		return time.Unix(message.Time, 0).UTC()
	}
	return monitor.now().UTC()
}

type infrastructureEvent struct {
	action       string
	target       string
	detail       string
	timestamp    time.Time
	failed       bool
	resourceType string
	resourceID   string
}

func (monitor *DockerMonitor) record(ctx context.Context, event infrastructureEvent) {
	result := 0
	if event.failed {
		result = 1
	}
	target := truncate(event.target, 256)
	detail := truncate(event.detail, 1000)
	var detailValue any
	if detail != "" {
		detailValue = detail
	}
	eventID := uuid.New()
	_, err := monitor.database.Exec(ctx, `INSERT INTO "InfrastructureEvents" ("Id","Action","Target","Timestamp","Result","Detail") VALUES ($1,$2,$3,$4,$5,$6)`,
		eventID, event.action, target, event.timestamp.UTC(), result, detailValue)
	if err != nil && !errors.Is(err, context.Canceled) {
		monitor.logger.Error("Infrastructure event could not be recorded", "action", event.action, "error", err)
		return
	}
	if err == nil && monitor.alerts != nil {
		if signal, publish := alertSignal(event, target, detail); publish {
			if err := monitor.alerts.Record(ctx, signal); err != nil && !errors.Is(err, context.Canceled) {
				monitor.logger.Error("Infrastructure alert could not be recorded", "action", event.action)
			}
		}
	}
}

func alertSignal(event infrastructureEvent, target, detail string) (alerts.Signal, bool) {
	title, severity, fingerprint, kind, recovery := "", "warning", "", event.action, false
	switch event.action {
	case "container.unexpected_stop":
		title, severity, fingerprint = "Container stopped unexpectedly", "critical", "container.stop:"+event.resourceID
	case "container.out_of_memory":
		title, severity, fingerprint = "Container ran out of memory", "critical", "container.oom:"+event.resourceID
	case "container.unhealthy":
		title, fingerprint = "Container became unhealthy", "container.health:"+event.resourceID
	case "container.restart_loop":
		title, severity, fingerprint = "Container restart loop", "critical", "container.restart_loop:"+event.resourceID
	case "server.disconnected":
		title, severity, fingerprint, kind = "Docker unavailable", "critical", "docker.availability", "docker.unavailable"
	case "container.recovered":
		title, severity, fingerprint, kind, recovery = "Container recovered", "info", "container.health:"+event.resourceID, "container.unhealthy", true
	case "server.reconnected":
		title, severity, fingerprint, kind, recovery = "Docker reconnected", "info", "docker.availability", "docker.unavailable", true
	default:
		return alerts.Signal{}, false
	}
	message := target
	if detail != "" {
		message += " — " + detail
	}
	return alerts.Signal{
		Fingerprint: fingerprint, Kind: kind, Severity: severity, Title: title, Message: message,
		Source: "docker", ResourceType: event.resourceType, ResourceID: event.resourceID,
		OccurredAt: event.timestamp, Recovery: recovery,
	}, true
}

func wait(ctx context.Context, duration time.Duration) bool {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func shortID(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	if id == "" {
		return "Unknown container"
	}
	return id
}

func truncate(value string, maximum int) string {
	if len(value) <= maximum {
		return value
	}
	return value[:maximum]
}
