package containers

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"

	"github.com/containerd/errdefs"
	"github.com/moby/moby/api/pkg/stdcopy"
	containertypes "github.com/moby/moby/api/types/container"
	networktypes "github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
)

type fakeDocker struct {
	items      []containertypes.Summary
	err        error
	inspect    containertypes.InspectResponse
	inspectErr error
	stats      string
	logs       string
	logsErr    error
	logOptions *client.ContainerLogsOptions
}

func (f fakeDocker) ContainerList(context.Context, client.ContainerListOptions) (client.ContainerListResult, error) {
	return client.ContainerListResult{Items: f.items}, f.err
}
func (f fakeDocker) ContainerInspect(context.Context, string, client.ContainerInspectOptions) (client.ContainerInspectResult, error) {
	return client.ContainerInspectResult{Container: f.inspect}, f.inspectErr
}

func TestDetailPreservesFlutterContractAndSanitizesLabels(t *testing.T) {
	port := networktypes.MustParsePort("8080/tcp")
	docker := fakeDocker{
		inspect: containertypes.InspectResponse{
			ID: strings.Repeat("a", 64), Name: "/api", Created: "2026-09-05T06:00:00Z", RestartCount: 3,
			Config:          &containertypes.Config{Image: "api:2", Labels: map[string]string{"team": "flare", "api_token": "do-not-return"}},
			State:           &containertypes.State{Status: "running", StartedAt: "2026-09-05T07:00:00Z", Health: &containertypes.Health{Status: "healthy"}},
			NetworkSettings: &containertypes.NetworkSettings{Ports: networktypes.PortMap{port: {{HostIP: netip.MustParseAddr("127.0.0.1"), HostPort: "18080"}}}},
		},
		stats: `{"cpu_stats":{"cpu_usage":{"total_usage":200},"system_cpu_usage":2000,"online_cpus":2},"precpu_stats":{"cpu_usage":{"total_usage":100},"system_cpu_usage":1000},"memory_stats":{"usage":1000,"limit":4096,"stats":{"inactive_file":100}},"networks":{"eth0":{"rx_bytes":40,"tx_bytes":20}}}`,
	}
	response := httptest.NewRecorder()
	NewHandler(docker, slog.New(slog.NewTextHandler(io.Discard, nil))).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/"+strings.Repeat("a", 64), nil))
	body := response.Body.String()
	for _, expected := range []string{`"shortId":"aaaaaaaaaaaa"`, `"name":"api"`, `"state":"Running"`, `"health":"Healthy"`, `"memoryLimitBytes":4096`, `"networkReceiveBytes":40`, `"privatePort":8080`, `"publicPort":18080`, `"team":"flare"`} {
		if !strings.Contains(body, expected) {
			t.Fatalf("response missing %s: %s", expected, body)
		}
	}
	if response.Code != http.StatusOK || strings.Contains(body, "api_token") || strings.Contains(body, "do-not-return") {
		t.Fatalf("response = %d %s", response.Code, body)
	}
}

func TestDetailRejectsInvalidIdentifier(t *testing.T) {
	response := httptest.NewRecorder()
	NewHandler(fakeDocker{}, slog.New(slog.NewTextHandler(io.Discard, nil))).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/not-an-id", nil))
	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "Container identifier is invalid") {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
}

func TestDetailReturnsNotFoundProblem(t *testing.T) {
	response := httptest.NewRecorder()
	docker := fakeDocker{inspectErr: errdefs.ErrNotFound}
	NewHandler(docker, slog.New(slog.NewTextHandler(io.Discard, nil))).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/"+strings.Repeat("b", 12), nil))
	if response.Code != http.StatusNotFound || strings.Contains(response.Body.String(), "missing") {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
}
func (f fakeDocker) ContainerStats(context.Context, string, client.ContainerStatsOptions) (client.ContainerStatsResult, error) {
	return client.ContainerStatsResult{Body: io.NopCloser(strings.NewReader(f.stats))}, nil
}
func (f fakeDocker) ContainerLogs(_ context.Context, _ string, options client.ContainerLogsOptions) (client.ContainerLogsResult, error) {
	if f.logOptions != nil {
		*f.logOptions = options
	}
	return io.NopCloser(strings.NewReader(f.logs)), f.logsErr
}

func TestLogsDemultiplexesBoundsAndPaginates(t *testing.T) {
	var multiplexed bytes.Buffer
	writeMultiplexed(&multiplexed, stdcopy.Stdout, "2026-09-05T08:00:00Z first\n2026-09-05T08:02:00Z third\n")
	writeMultiplexed(&multiplexed, stdcopy.Stderr, "2026-09-05T08:01:00Z second\n")
	options := client.ContainerLogsOptions{}
	docker := fakeDocker{
		inspect: containertypes.InspectResponse{Config: &containertypes.Config{Tty: false}},
		logs:    multiplexed.String(), logOptions: &options,
	}
	response := httptest.NewRecorder()
	target := "/" + strings.Repeat("a", 12) + "/logs?tail=2&before=2026-09-05T09%3A00%3A00%2B01%3A00"
	NewHandler(docker, slog.New(slog.NewTextHandler(io.Discard, nil))).ServeHTTP(response, httptest.NewRequest(http.MethodGet, target, nil))
	body := response.Body.String()
	if response.Code != http.StatusOK || strings.Contains(body, "first") || !strings.Contains(body, "second") || !strings.Contains(body, "third") {
		t.Fatalf("response = %d %s", response.Code, body)
	}
	for _, expected := range []string{`"requestedTail":2`, `"truncated":true`, `"oldestTimestamp":"2026-09-05T08:01:00Z"`, `"newestTimestamp":"2026-09-05T08:02:00Z"`} {
		if !strings.Contains(body, expected) {
			t.Fatalf("response missing %s: %s", expected, body)
		}
	}
	if options.Tail != "2" || options.Until != "2026-09-05T08:00:00Z" || !options.ShowStdout || !options.ShowStderr || !options.Timestamps || options.Follow {
		t.Fatalf("unexpected Docker log options: %+v", options)
	}
}

func writeMultiplexed(destination *bytes.Buffer, stream stdcopy.StdType, text string) {
	header := [8]byte{byte(stream)}
	binary.BigEndian.PutUint32(header[4:], uint32(len(text)))
	destination.Write(header[:])
	destination.WriteString(text)
}

func TestLogsRejectsInvalidPagination(t *testing.T) {
	for _, target := range []string{"/" + strings.Repeat("a", 12) + "/logs?tail=many", "/" + strings.Repeat("a", 12) + "/logs?before=yesterday"} {
		response := httptest.NewRecorder()
		NewHandler(fakeDocker{}, slog.New(slog.NewTextHandler(io.Discard, nil))).ServeHTTP(response, httptest.NewRequest(http.MethodGet, target, nil))
		if response.Code != http.StatusBadRequest || response.Header().Get("Content-Type") != "application/problem+json; charset=utf-8" {
			t.Fatalf("response = %d %s", response.Code, response.Body.String())
		}
	}
}

func TestLogOutputCapsMemoryAndDropsPartialFirstLine(t *testing.T) {
	prefix := strings.Repeat("x", 2*1024*1024)
	output, truncated, err := readLogOutput(strings.NewReader(prefix+"\nlast line\n"), true)
	if err != nil || !truncated || string(output) != "last line\n" {
		t.Fatalf("truncated=%v err=%v length=%d suffix=%q", truncated, err, len(output), string(output[max(len(output)-20, 0):]))
	}
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
