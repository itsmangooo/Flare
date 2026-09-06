package httpapi

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/itsmangooo/flare/internal/config"
)

type pingFunc func(context.Context) error

func (fn pingFunc) Ping(ctx context.Context) error { return fn(ctx) }

func TestHealthContracts(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tests := []struct {
		name       string
		path       string
		ping       pingFunc
		wantStatus int
		wantBody   string
	}{
		{"live", "/health/live", func(context.Context) error { return nil }, http.StatusOK, `"checks":{}`},
		{"ready", "/health/ready", func(context.Context) error { return nil }, http.StatusOK, `"postgres":{"status":"healthy"}`},
		{"not ready", "/health/ready", func(context.Context) error { return errors.New("offline") }, http.StatusServiceUnavailable, `"status":"unhealthy"`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := New(config.Config{HTTPAddress: ":0"}, "test", logger, test.ping, Routes{})
			response := httptest.NewRecorder()
			server.Handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, test.path, nil))
			if response.Code != test.wantStatus || !strings.Contains(response.Body.String(), test.wantBody) {
				t.Fatalf("response = %d %s", response.Code, response.Body.String())
			}
		})
	}
}

func TestPublicPageIsUsefulAndSanitized(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := New(config.Config{HTTPAddress: ":0"}, "1.4.2", logger, pingFunc(func(context.Context) error { return nil }), Routes{})
	response := httptest.NewRecorder()
	server.Handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))

	body := response.Body.String()
	if response.Code != http.StatusOK || !strings.HasPrefix(response.Header().Get("Content-Type"), "text/html") {
		t.Fatalf("response = %d %q", response.Code, response.Header().Get("Content-Type"))
	}
	for _, expected := range []string{"Homelab Control Plane", "1.4.2", "Operational", "Backend</span><span class=\"value\">Go"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("page does not contain %q", expected)
		}
	}
	for _, forbidden := range []string{"COOLIFY_API_TOKEN", "CLOUDFLARE_API_TOKEN", "ConnectionStrings__Postgres", "DOCKER_HOST"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("page leaked sensitive configuration name %q", forbidden)
		}
	}
}

func TestDomainsRoutesAreMountedUnderVersionedAPI(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	domains := http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusNoContent)
	})
	server := New(config.Config{HTTPAddress: ":0"}, "test", logger,
		pingFunc(func(context.Context) error { return nil }), Routes{Domains: domains})
	response := httptest.NewRecorder()
	server.Handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/domains/status", nil))
	if response.Code != http.StatusNoContent {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
}

func TestTopologyRouteIsMountedUnderVersionedAPI(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	topology := http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusNoContent)
	})
	server := New(config.Config{HTTPAddress: ":0"}, "test", logger,
		pingFunc(func(context.Context) error { return nil }), Routes{Topology: topology})
	response := httptest.NewRecorder()
	server.Handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/topology", nil))
	if response.Code != http.StatusNoContent {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
}

func TestTelemetryRouteIsMountedOutsideBoundedRequests(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	telemetry := http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusNoContent)
	})
	server := New(config.Config{HTTPAddress: ":0"}, "test", logger,
		pingFunc(func(context.Context) error { return nil }), Routes{Telemetry: telemetry})
	response := httptest.NewRecorder()
	server.Handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/telemetry", nil))
	if response.Code != http.StatusNoContent {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
}

func TestPublicPageReportsUnavailableDatabase(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	server := New(config.Config{HTTPAddress: ":0"}, "test", logger, pingFunc(func(context.Context) error { return errors.New("offline") }), Routes{})
	response := httptest.NewRecorder()
	server.Handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "Unavailable") {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
}
