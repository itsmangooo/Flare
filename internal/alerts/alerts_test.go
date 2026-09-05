package alerts

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestNtfyPublisherUsesDocumentedJSONAPIAndBearerAuth(t *testing.T) {
	var received publishRequest
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost || request.URL.Path != "/ntfy/" || request.Header.Get("Authorization") != "Bearer private-token" {
			t.Fatalf("request = %s %s auth=%q", request.Method, request.URL.Path, request.Header.Get("Authorization"))
		}
		if err := json.NewDecoder(request.Body).Decode(&received); err != nil {
			t.Fatal(err)
		}
		writer.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	publisher, err := NewNtfyPublisher(server.URL+"/ntfy", "flare-alerts_01", "private-token")
	if err != nil {
		t.Fatal(err)
	}
	publisher.client = server.Client()
	if err := publisher.Publish(context.Background(), Notification{
		ID: "event-1", Title: "Container stopped", Message: "api exited with code 137.",
		Priority: 5, Tags: []string{"warning", "container"},
	}); err != nil {
		t.Fatal(err)
	}
	if received.Topic != "flare-alerts_01" || received.SequenceID != "event-1" || received.Priority != 5 || received.Message != "api exited with code 137." {
		t.Fatalf("payload = %#v", received)
	}
}

func TestNtfyConfigurationIsOptionalAndStrict(t *testing.T) {
	disabled, err := NewNtfyPublisher("", "", "")
	if err != nil || disabled.Configured() {
		t.Fatalf("disabled publisher = %#v, %v", disabled, err)
	}
	for _, test := range []struct{ baseURL, topic, token string }{
		{"http://ntfy.example.test", "alerts", ""},
		{"https://user:pass@ntfy.example.test", "alerts", ""},
		{"https://ntfy.example.test?token=secret", "alerts", ""},
		{"https://ntfy.example.test", "bad/topic", ""},
		{"", "alerts", ""},
		{"", "", "orphan-token"},
	} {
		if _, err := NewNtfyPublisher(test.baseURL, test.topic, test.token); err == nil {
			t.Fatalf("configuration %#v was accepted", test)
		}
	}
}

type recordingPublisher struct {
	mu       sync.Mutex
	calls    int
	failures int
	done     chan Notification
}

func (publisher *recordingPublisher) Configured() bool { return true }
func (publisher *recordingPublisher) Publish(_ context.Context, notification Notification) error {
	publisher.mu.Lock()
	publisher.calls++
	call := publisher.calls
	publisher.mu.Unlock()
	if call <= publisher.failures {
		return io.ErrUnexpectedEOF
	}
	publisher.done <- notification
	return nil
}

func TestDispatcherRetriesWithoutBlockingProducer(t *testing.T) {
	publisher := &recordingPublisher{failures: 2, done: make(chan Notification, 1)}
	dispatcher := NewDispatcher(publisher, slog.New(slog.DiscardHandler))
	dispatcher.retryDelay = time.Millisecond
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go dispatcher.Run(ctx)
	started := time.Now()
	if err := dispatcher.Notify(ctx, Notification{ID: "event-1", Title: "Alert"}); err != nil {
		t.Fatal(err)
	}
	if time.Since(started) > 100*time.Millisecond {
		t.Fatal("producer was blocked by network delivery")
	}
	select {
	case notification := <-publisher.done:
		if notification.ID != "event-1" {
			t.Fatalf("notification = %#v", notification)
		}
	case <-time.After(time.Second):
		t.Fatal("notification was not delivered")
	}
	publisher.mu.Lock()
	defer publisher.mu.Unlock()
	if publisher.calls != 3 {
		t.Fatalf("publish calls = %d", publisher.calls)
	}
}

func TestPublisherBoundsUntrustedNotificationText(t *testing.T) {
	var received publishRequest
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_ = json.NewDecoder(request.Body).Decode(&received)
		writer.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	publisher, _ := NewNtfyPublisher(server.URL, "alerts", "")
	publisher.client = server.Client()
	err := publisher.Publish(context.Background(), Notification{
		Title: strings.Repeat("t", maximumTitleBytes+20), Message: strings.Repeat("m", maximumMessageBytes+20),
		Priority: 99, Tags: []string{"valid", strings.Repeat("x", 41)},
	})
	if err != nil || len(received.Title) != maximumTitleBytes || len(received.Message) != maximumMessageBytes ||
		received.Priority != 5 || len(received.Tags) != 1 {
		t.Fatalf("payload = %#v, err = %v", received, err)
	}
}

func TestNotificationTruncationPreservesUTF8(t *testing.T) {
	value := truncate(strings.Repeat("🔥", 60), 201)
	if !strings.HasSuffix(value, "🔥") || len(value) > 201 {
		t.Fatalf("truncated value is not valid at %d bytes: %q", len(value), value)
	}
}
