package cloudflare

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeDomainService struct {
	configured bool
	tunnels    bool
	zoneID     string
	tunnelID   string
	err        error
}

func (service *fakeDomainService) Configured() bool        { return service.configured }
func (service *fakeDomainService) TunnelsConfigured() bool { return service.tunnels }
func (service *fakeDomainService) Zones(context.Context) ([]Zone, error) {
	return []Zone{{ID: testZoneID, Name: "example.test", Status: "active", Type: "full"}}, service.err
}
func (service *fakeDomainService) DNSRecords(_ context.Context, zoneID string) ([]DNSRecord, error) {
	service.zoneID = zoneID
	return []DNSRecord{{ID: "record", ZoneID: zoneID, Name: "app.example.test", Type: "CNAME", Target: "tunnel.example.test", TTL: 1}}, service.err
}
func (service *fakeDomainService) Tunnels(context.Context) ([]Tunnel, error) {
	return []Tunnel{{ID: testTunnelID, Name: "homelab", Status: "healthy", ConfigSource: "cloudflare"}}, service.err
}
func (service *fakeDomainService) TunnelRoutes(_ context.Context, tunnelID string) ([]TunnelRoute, error) {
	service.tunnelID = tunnelID
	return []TunnelRoute{{TunnelID: tunnelID, Hostname: "app.example.test", OriginKind: "http"}}, service.err
}

func TestDomainRoutesReturnSafeReadModels(t *testing.T) {
	service := &fakeDomainService{configured: true, tunnels: true}
	handler := NewHandler(service, slog.New(slog.DiscardHandler))
	for _, test := range []struct {
		path string
		want string
	}{
		{"/status", `{"configured":true,"tunnelsConfigured":true}`},
		{"/zones", `[{"id":"` + testZoneID + `","name":"example.test","status":"active","type":"full"}]`},
		{"/zones/" + testZoneID + "/records", `[{"id":"record","zoneId":"` + testZoneID + `","name":"app.example.test","type":"CNAME","target":"tunnel.example.test","ttl":1,"proxied":null,"proxiable":false}]`},
		{"/tunnels", `[{"id":"` + testTunnelID + `","name":"homelab","status":"healthy","configSource":"cloudflare"}]`},
		{"/tunnels/" + testTunnelID + "/routes", `[{"tunnelId":"` + testTunnelID + `","hostname":"app.example.test","originKind":"http"}]`},
	} {
		t.Run(test.path, func(t *testing.T) {
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, test.path, nil))
			if response.Code != http.StatusOK || response.Header().Get("Content-Type") != "application/json; charset=utf-8" {
				t.Fatalf("response = %d %s", response.Code, response.Body.String())
			}
			assertCloudflareJSON(t, response.Body.Bytes(), test.want)
		})
	}
	if service.zoneID != testZoneID || service.tunnelID != testTunnelID {
		t.Fatalf("identifiers = zone %s, tunnel %s", service.zoneID, service.tunnelID)
	}
}

func TestDomainStatusWorksWhenIntegrationIsNotConfigured(t *testing.T) {
	response := httptest.NewRecorder()
	NewHandler(&fakeDomainService{}, slog.New(slog.DiscardHandler)).ServeHTTP(
		response, httptest.NewRequest(http.MethodGet, "/status", nil))
	if response.Code != http.StatusOK || response.Body.String() != "{\"configured\":false,\"tunnelsConfigured\":false}\n" {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
}

func TestDomainHandlerMapsSanitizedProblems(t *testing.T) {
	for _, test := range []struct {
		name   string
		err    error
		status int
		detail string
	}{
		{"invalid", ErrInvalidIdentifier, http.StatusBadRequest, "Cloudflare identifier is invalid."},
		{"missing", ErrNotFound, http.StatusNotFound, "The requested Cloudflare resource does not exist."},
		{"not configured", ErrNotConfigured, http.StatusServiceUnavailable, "Cloudflare integration is not configured."},
		{"upstream secret", errors.New("token=must-not-leak"), http.StatusServiceUnavailable, "Cloudflare is unavailable."},
	} {
		t.Run(test.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			NewHandler(&fakeDomainService{err: test.err}, slog.New(slog.DiscardHandler)).ServeHTTP(
				response, httptest.NewRequest(http.MethodGet, "/zones", nil))
			if response.Code != test.status || response.Header().Get("Content-Type") != "application/problem+json; charset=utf-8" {
				t.Fatalf("response = %d %s", response.Code, response.Body.String())
			}
			var problem map[string]any
			if err := json.Unmarshal(response.Body.Bytes(), &problem); err != nil {
				t.Fatal(err)
			}
			if problem["detail"] != test.detail || problem["detail"] == "token=must-not-leak" {
				t.Fatalf("problem = %#v", problem)
			}
		})
	}
}

func TestDomainRoutesRejectUnsupportedMethods(t *testing.T) {
	response := httptest.NewRecorder()
	NewHandler(&fakeDomainService{}, slog.New(slog.DiscardHandler)).ServeHTTP(
		response, httptest.NewRequest(http.MethodPost, "/zones", nil))
	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("response = %d", response.Code)
	}
}

func assertCloudflareJSON(t *testing.T, actual []byte, expected string) {
	t.Helper()
	var actualValue, expectedValue any
	if err := json.Unmarshal(actual, &actualValue); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(expected), &expectedValue); err != nil {
		t.Fatal(err)
	}
	actualJSON, _ := json.Marshal(actualValue)
	expectedJSON, _ := json.Marshal(expectedValue)
	if string(actualJSON) != string(expectedJSON) {
		t.Fatalf("JSON = %s, want %s", actualJSON, expectedJSON)
	}
}
