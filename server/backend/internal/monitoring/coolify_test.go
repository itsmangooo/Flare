package monitoring

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/itsmangooo/flare/server/backend/internal/coolify"
)

type fakeCoolifyMonitoringClient struct {
	configured    bool
	serversError  error
	deploymentErr error
	deployments   []coolify.Deployment
}

func (client *fakeCoolifyMonitoringClient) Configured() bool { return client.configured }

func (client *fakeCoolifyMonitoringClient) Servers(context.Context) ([]coolify.Server, error) {
	return []coolify.Server{{UUID: "server_1", Name: "primary"}}, client.serversError
}

func (client *fakeCoolifyMonitoringClient) Deployments(context.Context, int, int) (coolify.DeploymentPage, error) {
	return coolify.DeploymentPage{Items: client.deployments}, client.deploymentErr
}

func TestCoolifyMonitorRecordsAvailabilityTransitionsOnce(t *testing.T) {
	client := &fakeCoolifyMonitoringClient{configured: true, serversError: errors.New("private upstream failure")}
	database := &fakeDatabase{}
	sink := &fakeAlertSink{}
	monitor := NewCoolifyMonitor(client, database, slog.New(slog.DiscardHandler), sink)
	now := time.Date(2026, 9, 6, 14, 0, 0, 0, time.UTC)
	monitor.now = func() time.Time { return now }

	if err := monitor.poll(context.Background()); err == nil {
		t.Fatal("unavailable Coolify poll succeeded")
	}
	_ = monitor.poll(context.Background())
	if got := len(database.snapshot()); got != 1 {
		t.Fatalf("repeated outage events = %d, want 1", got)
	}

	client.serversError = nil
	if err := monitor.poll(context.Background()); err != nil {
		t.Fatal(err)
	}
	events := database.snapshot()
	if len(events) != 2 || events[0].action != "coolify.disconnected" || events[1].action != "coolify.reconnected" {
		t.Fatalf("availability events = %#v", events)
	}
	if len(sink.signals) != 2 || sink.signals[0].Fingerprint != "coolify.availability" || !sink.signals[1].Recovery {
		t.Fatalf("availability signals = %#v", sink.signals)
	}
	for _, signal := range sink.signals {
		if strings.Contains(signal.Message, "private upstream failure") {
			t.Fatal("upstream failure detail leaked into alert")
		}
	}
}

func TestCoolifyMonitorSkipsStaleHistoryAndRecoversNewFailure(t *testing.T) {
	now := time.Date(2026, 9, 6, 15, 0, 0, 0, time.UTC)
	failed := "failed"
	finished := "finished"
	oldTime := now.Add(-time.Hour)
	recentTime := now.Add(-time.Minute)
	client := &fakeCoolifyMonitoringClient{
		configured: true,
		deployments: []coolify.Deployment{
			{UUID: "deploy_already_recovered", ResourceUUID: "app_4", ResourceName: "site", Status: &finished, FinishedAt: &recentTime},
			{UUID: "deploy_failed_before_recovery", ResourceUUID: "app_4", ResourceName: "site", Status: &failed, FinishedAt: &recentTime},
			{UUID: "deploy_recent", ResourceUUID: "app_1", ResourceName: "api", Status: &failed, FinishedAt: &recentTime, Logs: stringPointer("token=must-not-leak")},
			{UUID: "deploy_old", ResourceUUID: "app_2", ResourceName: "worker", Status: &failed, FinishedAt: &oldTime},
		},
	}
	database := &fakeDatabase{}
	sink := &fakeAlertSink{}
	monitor := NewCoolifyMonitor(client, database, slog.New(slog.DiscardHandler), sink)
	monitor.now = func() time.Time { return now }

	if err := monitor.poll(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(sink.signals) != 1 || sink.signals[0].Kind != "coolify.deployment_failed" || sink.signals[0].ResourceID != "app_1" {
		t.Fatalf("initial deployment signals = %#v", sink.signals)
	}
	if strings.Contains(sink.signals[0].Message, "must-not-leak") {
		t.Fatal("deployment logs leaked into alert")
	}
	if err := monitor.poll(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(sink.signals) != 1 {
		t.Fatalf("unchanged deployment was emitted again: %#v", sink.signals)
	}

	successTime := now.Add(time.Minute)
	client.deployments = append([]coolify.Deployment{
		{UUID: "deploy_unrelated", ResourceUUID: "app_3", ResourceName: "web", Status: &finished, FinishedAt: &successTime},
		{UUID: "deploy_success", ResourceUUID: "app_1", ResourceName: "api", Status: &finished, FinishedAt: &successTime},
	}, client.deployments...)
	if err := monitor.poll(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(sink.signals) != 2 || !sink.signals[1].Recovery || sink.signals[1].Fingerprint != sink.signals[0].Fingerprint {
		t.Fatalf("deployment recovery signals = %#v", sink.signals)
	}
	events := database.snapshot()
	if len(events) != 2 || events[0].action != "coolify.deployment_failed" || events[1].action != "coolify.deployment_recovered" {
		t.Fatalf("deployment events = %#v", events)
	}
}

func stringPointer(value string) *string { return &value }
