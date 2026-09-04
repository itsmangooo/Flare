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
			server := New(config.Config{HTTPAddress: ":0"}, "test", logger, test.ping)
			response := httptest.NewRecorder()
			server.Handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, test.path, nil))
			if response.Code != test.wantStatus || !strings.Contains(response.Body.String(), test.wantBody) {
				t.Fatalf("response = %d %s", response.Code, response.Body.String())
			}
		})
	}
}
