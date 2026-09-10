package topology

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	containertypes "github.com/moby/moby/api/types/container"
	networktypes "github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
)

type fakeDocker struct {
	networks     []networktypes.Summary
	containers   []containertypes.Summary
	networkError error
	listError    error
}

func (docker fakeDocker) NetworkList(context.Context, client.NetworkListOptions) (client.NetworkListResult, error) {
	return client.NetworkListResult{Items: docker.networks}, docker.networkError
}

func (docker fakeDocker) ContainerList(context.Context, client.ContainerListOptions) (client.ContainerListResult, error) {
	return client.ContainerListResult{Items: docker.containers}, docker.listError
}

func TestTopologyBuildsOnlyObservedRelationships(t *testing.T) {
	bridgeID := strings.Repeat("a", 64)
	containerID := strings.Repeat("b", 64)
	settings := &containertypes.NetworkSettingsSummary{Networks: map[string]*networktypes.EndpointSettings{
		"flare": {NetworkID: bridgeID},
	}}
	response := httptest.NewRecorder()
	handler := NewHandler(fakeDocker{
		networks: []networktypes.Summary{{Network: networktypes.Network{
			ID: bridgeID, Name: "flare", Driver: "bridge", Scope: "local",
		}}},
		containers: []containertypes.Summary{{
			ID: containerID, Names: []string{"/api"}, Image: "flare-api:1", State: "running",
			NetworkSettings: settings,
			Ports:           []containertypes.PortSummary{{PrivatePort: 8080, PublicPort: 8443, Type: "tcp"}},
		}},
	}, "lab-01", slog.New(slog.NewTextHandler(io.Discard, nil)))
	handler.(*Handler).now = func() time.Time { return time.Date(2026, 9, 5, 20, 0, 0, 0, time.UTC) }
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))

	body := response.Body.String()
	for _, expected := range []string{
		`"name":"lab-01"`, `"driver":"bridge"`, `"containerIds":["` + containerID + `"]`,
		`"shortId":"bbbbbbbbbbbb"`, `"networkIds":["` + bridgeID + `"]`,
		`"privatePort":8080`, `"publicPort":8443`, `"protocol":"tcp"`,
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("response missing %s: %s", expected, body)
		}
	}
	for _, privateValue := range []string{"172.20.0.2", "MacAddress", "Aliases", "Labels"} {
		if strings.Contains(body, privateValue) {
			t.Fatalf("response exposed %q: %s", privateValue, body)
		}
	}
}

func TestTopologyPreservesUnlistedObservedNetwork(t *testing.T) {
	containerID := strings.Repeat("c", 64)
	response := httptest.NewRecorder()
	NewHandler(fakeDocker{containers: []containertypes.Summary{{
		ID: containerID, Names: []string{"/worker"},
		NetworkSettings: &containertypes.NetworkSettingsSummary{Networks: map[string]*networktypes.EndpointSettings{
			"runtime-network": {NetworkID: "runtime-id"},
		}},
	}}}, "lab", nil).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"id":"runtime-id","name":"runtime-network"`) {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
}

func TestTopologyReturnsSanitizedUnavailableProblem(t *testing.T) {
	response := httptest.NewRecorder()
	NewHandler(fakeDocker{networkError: errors.New("dial /private/docker.sock token=secret")}, "lab", nil).
		ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	body := response.Body.String()
	if response.Code != http.StatusServiceUnavailable || !strings.Contains(body, "Docker topology is unavailable") ||
		strings.Contains(body, "private") || strings.Contains(body, "secret") {
		t.Fatalf("response = %d %s", response.Code, body)
	}
}

func TestTopologyIsReadOnly(t *testing.T) {
	response := httptest.NewRecorder()
	NewHandler(fakeDocker{}, "lab", nil).ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/", nil))
	if response.Code != http.StatusMethodNotAllowed || response.Header().Get("Allow") != http.MethodGet {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
}
