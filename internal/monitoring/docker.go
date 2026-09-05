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
	"github.com/itsmangooo/flare/internal/alerts"
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
	Notify(context.Context, alerts.Notification) error
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
			detail: fmt.Sprintf("Container exited with code %d.", exitCode), timestamp: timestamp})
		monitor.recordDeath(ctx, id, target, timestamp)
	case eventtypes.ActionOOM:
		monitor.record(ctx, infrastructureEvent{action: "container.out_of_memory", target: target, failed: true,
			detail: "Container was terminated by the out-of-memory killer.", timestamp: timestamp})
	case eventtypes.ActionHealthStatusUnhealthy:
		if !monitor.unhealthy[id] {
			monitor.record(ctx, infrastructureEvent{action: "container.unhealthy", target: target, failed: true,
				detail: "Docker health check reported unhealthy.", timestamp: timestamp})
			monitor.unhealthy[id] = true
		}
	case eventtypes.ActionHealthStatusHealthy:
		if monitor.unhealthy[id] {
			monitor.record(ctx, infrastructureEvent{action: "container.recovered", target: target,
				detail: "Docker health check recovered.", timestamp: timestamp})
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
			detail: "Container exited repeatedly within five minutes.", timestamp: timestamp})
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
	action    string
	target    string
	detail    string
	timestamp time.Time
	failed    bool
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
		if notification, publish := alertNotification(eventID.String(), event, target, detail); publish {
			if err := monitor.alerts.Notify(ctx, notification); err != nil && !errors.Is(err, context.Canceled) {
				monitor.logger.Error("Infrastructure alert could not be queued", "action", event.action)
			}
		}
	}
}

func alertNotification(id string, event infrastructureEvent, target, detail string) (alerts.Notification, bool) {
	title, priority, tags := "", 4, []string{"warning", "flare"}
	switch event.action {
	case "container.unexpected_stop":
		title, priority, tags = "Container stopped unexpectedly", 5, []string{"warning", "container"}
	case "container.out_of_memory":
		title, priority, tags = "Container ran out of memory", 5, []string{"warning", "container"}
	case "container.unhealthy":
		title, tags = "Container became unhealthy", []string{"warning", "container"}
	case "container.restart_loop":
		title, priority, tags = "Container restart loop", 5, []string{"warning", "container"}
	case "server.disconnected":
		title, priority, tags = "Docker unavailable", 5, []string{"warning", "server"}
	case "container.recovered":
		title, priority, tags = "Container recovered", 2, []string{"white_check_mark", "container"}
	case "server.reconnected":
		title, priority, tags = "Docker reconnected", 2, []string{"white_check_mark", "server"}
	default:
		return alerts.Notification{}, false
	}
	message := target
	if detail != "" {
		message += " — " + detail
	}
	return alerts.Notification{ID: id, Title: title, Message: message, Priority: priority, Tags: tags}, true
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
