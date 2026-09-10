package monitoring

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/itsmangooo/flare/server/backend/internal/cloudflare"
)

type fakeCloudflareMonitoringClient struct {
	configured        bool
	tunnelsConfigured bool
	zonesError        error
	tunnelsError      error
	tunnels           []cloudflare.Tunnel
	tunnelCalls       int
}

func (client *fakeCloudflareMonitoringClient) Configured() bool { return client.configured }

func (client *fakeCloudflareMonitoringClient) TunnelsConfigured() bool {
	return client.tunnelsConfigured
}

func (client *fakeCloudflareMonitoringClient) Zones(context.Context) ([]cloudflare.Zone, error) {
	return []cloudflare.Zone{{ID: "zone_1", Name: "example.test", Status: "active"}}, client.zonesError
}

func (client *fakeCloudflareMonitoringClient) Tunnels(context.Context) ([]cloudflare.Tunnel, error) {
	client.tunnelCalls++
	return client.tunnels, client.tunnelsError
}

func TestCloudflareMonitorRecordsAvailabilityTransitionsOnce(t *testing.T) {
	client := &fakeCloudflareMonitoringClient{configured: true, zonesError: errors.New("token=must-not-leak")}
	database := &fakeDatabase{}
	sink := &fakeAlertSink{}
	monitor := NewCloudflareMonitor(client, database, slog.New(slog.DiscardHandler), sink)
	now := time.Date(2026, 9, 6, 16, 0, 0, 0, time.UTC)
	monitor.now = func() time.Time { return now }

	if err := monitor.poll(context.Background()); err == nil {
		t.Fatal("unavailable Cloudflare poll succeeded")
	}
	_ = monitor.poll(context.Background())
	if got := len(database.snapshot()); got != 1 {
		t.Fatalf("repeated outage events = %d, want 1", got)
	}
	client.zonesError = nil
	if err := monitor.poll(context.Background()); err != nil {
		t.Fatal(err)
	}
	events := database.snapshot()
	if len(events) != 2 || events[0].action != "cloudflare.disconnected" || events[1].action != "cloudflare.reconnected" {
		t.Fatalf("availability events = %#v", events)
	}
	if len(sink.signals) != 2 || sink.signals[0].Fingerprint != "cloudflare.availability" || !sink.signals[1].Recovery {
		t.Fatalf("availability signals = %#v", sink.signals)
	}
	for _, signal := range sink.signals {
		if strings.Contains(signal.Message, "must-not-leak") {
			t.Fatal("Cloudflare error detail leaked into alert")
		}
	}
}

func TestCloudflareMonitorDeduplicatesTunnelTransitions(t *testing.T) {
	client := &fakeCloudflareMonitoringClient{
		configured: true, tunnelsConfigured: true,
		tunnels: []cloudflare.Tunnel{
			{ID: "tunnel_1", Name: "homelab", Status: "down"},
			{ID: "tunnel_2", Name: "unknown", Status: "future-state"},
		},
	}
	database := &fakeDatabase{}
	sink := &fakeAlertSink{}
	monitor := NewCloudflareMonitor(client, database, slog.New(slog.DiscardHandler), sink)
	monitor.now = func() time.Time { return time.Date(2026, 9, 6, 17, 0, 0, 0, time.UTC) }

	if err := monitor.poll(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := monitor.poll(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(sink.signals) != 1 || sink.signals[0].Kind != "cloudflare.tunnel_unavailable" || sink.signals[0].ResourceID != "tunnel_1" {
		t.Fatalf("tunnel failure signals = %#v", sink.signals)
	}

	client.tunnels = []cloudflare.Tunnel{{ID: "tunnel_1", Name: "homelab", Status: "healthy"}}
	if err := monitor.poll(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(sink.signals) != 2 || !sink.signals[1].Recovery || sink.signals[1].Fingerprint != sink.signals[0].Fingerprint {
		t.Fatalf("tunnel recovery signals = %#v", sink.signals)
	}
	events := database.snapshot()
	if len(events) != 2 || events[0].action != "cloudflare.tunnel.down" || events[1].action != "cloudflare.tunnel.recovered" {
		t.Fatalf("tunnel events = %#v", events)
	}
}

func TestCloudflareMonitorDoesNotRequireTunnelConfiguration(t *testing.T) {
	client := &fakeCloudflareMonitoringClient{configured: true}
	monitor := NewCloudflareMonitor(client, &fakeDatabase{}, slog.New(slog.DiscardHandler), &fakeAlertSink{})
	if err := monitor.poll(context.Background()); err != nil {
		t.Fatal(err)
	}
	if client.tunnelCalls != 0 {
		t.Fatalf("tunnel calls = %d, want 0", client.tunnelCalls)
	}
}
