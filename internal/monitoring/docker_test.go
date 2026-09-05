package monitoring

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	eventtypes "github.com/moby/moby/api/types/events"
	"github.com/moby/moby/client"
)

type recordedEvent struct {
	action string
	target string
	result int
	detail any
}

type fakeDatabase struct {
	mu     sync.Mutex
	events []recordedEvent
	notify chan struct{}
}

func (database *fakeDatabase) Exec(_ context.Context, _ string, arguments ...any) (pgconn.CommandTag, error) {
	database.mu.Lock()
	database.events = append(database.events, recordedEvent{
		action: arguments[1].(string), target: arguments[2].(string),
		result: arguments[4].(int), detail: arguments[5],
	})
	database.mu.Unlock()
	if database.notify != nil {
		select {
		case database.notify <- struct{}{}:
		default:
		}
	}
	return pgconn.NewCommandTag("INSERT 0 1"), nil
}

func (database *fakeDatabase) snapshot() []recordedEvent {
	database.mu.Lock()
	defer database.mu.Unlock()
	return slices.Clone(database.events)
}

func TestDockerEventsCreateAlertsRecoveriesAndRestartLoop(t *testing.T) {
	database := &fakeDatabase{}
	monitor := NewDockerMonitor(nil, database, slog.New(slog.DiscardHandler))
	base := time.Date(2026, 9, 5, 10, 0, 0, 0, time.UTC)
	id := "0123456789abcdef"

	monitor.process(context.Background(), dockerMessage(id, "api", eventtypes.ActionDie, base.Add(-time.Second), map[string]string{"exitCode": "0"}))
	for index := range 3 {
		monitor.process(context.Background(), dockerMessage(id, "api", eventtypes.ActionDie, base.Add(time.Duration(index)*time.Minute), map[string]string{"exitCode": "137", "secret": "must-not-leak"}))
	}
	duplicate := dockerMessage(id, "api", eventtypes.ActionDie, base.Add(2*time.Minute), map[string]string{"exitCode": "137"})
	monitor.process(context.Background(), duplicate)
	monitor.process(context.Background(), dockerMessage(id, "api", eventtypes.ActionHealthStatusUnhealthy, base.Add(3*time.Minute), nil))
	monitor.process(context.Background(), dockerMessage(id, "api", eventtypes.ActionHealthStatusUnhealthy, base.Add(4*time.Minute), nil))
	monitor.process(context.Background(), dockerMessage(id, "api", eventtypes.ActionHealthStatusHealthy, base.Add(5*time.Minute), nil))

	events := database.snapshot()
	if len(events) != 6 {
		t.Fatalf("events = %#v", events)
	}
	wantActions := []string{
		"container.unexpected_stop", "container.unexpected_stop", "container.unexpected_stop",
		"container.restart_loop", "container.unhealthy", "container.recovered",
	}
	for index, want := range wantActions {
		if events[index].action != want {
			t.Fatalf("event %d action = %s, want %s", index, events[index].action, want)
		}
		if events[index].target != "api" {
			t.Fatalf("event %d target = %s", index, events[index].target)
		}
	}
	if events[0].result != 1 || events[len(events)-1].result != 0 {
		t.Fatalf("results = %#v", events)
	}
}

func TestDockerMonitorRecordsAvailabilityTransitionsAndReconnects(t *testing.T) {
	database := &fakeDatabase{notify: make(chan struct{}, 4)}
	docker := &fakeDockerEvents{}
	monitor := NewDockerMonitor(docker, database, slog.New(slog.DiscardHandler))
	monitor.retryDelay = time.Millisecond
	monitor.maxDelay = 2 * time.Millisecond
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		monitor.Run(ctx)
		close(done)
	}()

	deadline := time.After(2 * time.Second)
	for len(database.snapshot()) < 2 {
		select {
		case <-database.notify:
		case <-deadline:
			t.Fatal("availability events were not recorded")
		}
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("monitor did not stop after cancellation")
	}

	events := database.snapshot()
	if events[0].action != "server.disconnected" || events[0].detail != "Docker event stream unavailable." {
		t.Fatalf("disconnect = %#v", events[0])
	}
	if events[1].action != "server.reconnected" || events[1].result != 0 {
		t.Fatalf("reconnect = %#v", events[1])
	}
	docker.mu.Lock()
	defer docker.mu.Unlock()
	if docker.calls < 2 || !docker.filterObserved {
		t.Fatalf("Docker Events calls = %d, filter = %v", docker.calls, docker.filterObserved)
	}
}

type fakeDockerEvents struct {
	mu             sync.Mutex
	calls          int
	filterObserved bool
}

func (docker *fakeDockerEvents) Events(ctx context.Context, options client.EventsListOptions) client.EventsResult {
	docker.mu.Lock()
	docker.calls++
	call := docker.calls
	docker.filterObserved = options.Filters["type"][string(eventtypes.ContainerEventType)]
	docker.mu.Unlock()
	messages := make(chan eventtypes.Message)
	failures := make(chan error, 1)
	if call == 1 {
		close(messages)
		failures <- errors.New("socket path and token must not leak")
		close(failures)
		return client.EventsResult{Messages: messages, Err: failures}
	}
	go func() {
		defer close(messages)
		defer close(failures)
		select {
		case messages <- dockerMessage("0123456789abcdef", "api", eventtypes.ActionStart, time.Now(), nil):
		case <-ctx.Done():
			return
		}
		<-ctx.Done()
	}()
	return client.EventsResult{Messages: messages, Err: failures}
}

func dockerMessage(id, name string, action eventtypes.Action, timestamp time.Time, attributes map[string]string) eventtypes.Message {
	values := map[string]string{"name": name}
	for key, value := range attributes {
		values[key] = value
	}
	return eventtypes.Message{Type: eventtypes.ContainerEventType, Action: action,
		Actor: eventtypes.Actor{ID: id, Attributes: values}, Time: timestamp.Unix(), TimeNano: timestamp.UnixNano()}
}
