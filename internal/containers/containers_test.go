package containers

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	containertypes "github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

type fakeDocker struct {
	items   []containertypes.Summary
	err     error
	inspect containertypes.InspectResponse
	stats   string
}

func (f fakeDocker) ContainerList(context.Context, client.ContainerListOptions) (client.ContainerListResult, error) {
	return client.ContainerListResult{Items: f.items}, f.err
}
func (f fakeDocker) ContainerInspect(context.Context, string, client.ContainerInspectOptions) (client.ContainerInspectResult, error) {
	return client.ContainerInspectResult{Container: f.inspect}, nil
}
func (f fakeDocker) ContainerStats(context.Context, string, client.ContainerStatsOptions) (client.ContainerStatsResult, error) {
	return client.ContainerStatsResult{Body: io.NopCloser(strings.NewReader(f.stats))}, nil
}

func TestListCalculatesBoundedOneShotStats(t *testing.T) {
	docker := fakeDocker{
		items:   []containertypes.Summary{{ID: strings.Repeat("a", 64), Names: []string{"/api"}, Image: "api:2", State: "running"}},
		inspect: containertypes.InspectResponse{State: &containertypes.State{StartedAt: "2026-09-05T07:00:00Z", Health: &containertypes.Health{Status: "healthy"}}},
		stats:   `{"cpu_stats":{"cpu_usage":{"total_usage":200},"system_cpu_usage":2000,"online_cpus":2},"precpu_stats":{"cpu_usage":{"total_usage":100},"system_cpu_usage":1000},"memory_stats":{"usage":1000,"stats":{"inactive_file":100}}}`,
	}
	response := httptest.NewRecorder()
	NewHandler(docker, slog.New(slog.NewTextHandler(io.Discard, nil))).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	body := response.Body.String()
	for _, expected := range []string{`"cpuPercent":20`, `"memoryBytes":900`, `"startedAt":"2026-09-05T07:00:00Z"`} {
		if !strings.Contains(body, expected) {
			t.Fatalf("response missing %s: %s", expected, body)
		}
	}
}

func TestListPreservesFlutterContractAndStableOrder(t *testing.T) {
	docker := fakeDocker{items: []containertypes.Summary{{ID: strings.Repeat("b", 64), Names: []string{"/zeta"}, Image: "worker:1", State: "exited"}, {ID: strings.Repeat("a", 64), Names: []string{"/alpha"}, Image: "api:2", State: "running", Status: "Up 2 minutes (healthy)"}}}
	response := httptest.NewRecorder()
	NewHandler(docker, slog.New(slog.NewTextHandler(io.Discard, nil))).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	body := response.Body.String()
	if response.Code != http.StatusOK || !strings.Contains(body, `"state":"Running"`) || !strings.Contains(body, `"health":"Healthy"`) || strings.Index(body, "alpha") > strings.Index(body, "zeta") {
		t.Fatalf("response = %d %s", response.Code, body)
	}
}

func TestListReturnsSanitizedUnavailableProblem(t *testing.T) {
	response := httptest.NewRecorder()
	NewHandler(fakeDocker{err: errors.New("dial unix /private/socket: token=secret")}, slog.New(slog.NewTextHandler(io.Discard, nil))).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if response.Code != http.StatusServiceUnavailable || strings.Contains(response.Body.String(), "private/socket") || strings.Contains(response.Body.String(), "secret") {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
}
