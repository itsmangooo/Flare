package cloudflare

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

const (
	testAccountID = "0123456789abcdef0123456789abcdef"
	testZoneID    = "fedcba9876543210fedcba9876543210"
	testTunnelID  = "f70ff985-a4ef-4643-bbbc-4a0ed4fc8415"
)

func TestReadOnlyMethodsUseOfficialEndpointsAndSafeModels(t *testing.T) {
	const token = "cloudflare-server-secret"
	var requests []string
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer "+token || request.Header.Get("Accept") != "application/json" {
			t.Errorf("missing Cloudflare authentication headers")
			writer.WriteHeader(http.StatusBadRequest)
			return
		}
		requests = append(requests, request.URL.RequestURI())
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/client/v4/zones":
			_, _ = io.WriteString(writer, `{"success":true,"result":[{"id":"`+testZoneID+`","name":"example.test","status":"active","type":"full","account":{"id":"must-not-leak"}}],"result_info":{"page":1,"total_pages":1}}`)
		case "/client/v4/zones/" + testZoneID + "/dns_records":
			_, _ = io.WriteString(writer, `{"success":true,"result":[{"id":"record-id","name":"app.example.test","type":"CNAME","content":"tunnel.example.test","ttl":1,"proxied":true,"proxiable":true,"comment":"must-not-leak"}],"result_info":{"page":1,"total_pages":1}}`)
		case "/client/v4/accounts/" + testAccountID + "/cfd_tunnel":
			_, _ = io.WriteString(writer, `{"success":true,"result":[{"id":"`+testTunnelID+`","name":"homelab","status":"healthy","config_src":"cloudflare","connections":[{"origin_ip":"must-not-leak"}]}],"result_info":{"page":1,"total_pages":1}}`)
		case "/client/v4/accounts/" + testAccountID + "/cfd_tunnel/" + testTunnelID + "/configurations":
			_, _ = io.WriteString(writer, `{"success":true,"result":{"config":{"ingress":[{"hostname":"app.example.test","service":"http://private-host:8080","path":"/api/*","originRequest":{"caPool":"must-not-leak"}},{"hostname":"","service":"http_status:404"}]}}}`)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	client := testClient(server, token, testAccountID)

	zones, err := client.Zones(context.Background())
	if err != nil || len(zones) != 1 || zones[0].Name != "example.test" {
		t.Fatalf("zones = %#v, %v", zones, err)
	}
	records, err := client.DNSRecords(context.Background(), testZoneID)
	if err != nil || len(records) != 1 || records[0].Proxied == nil || !*records[0].Proxied {
		t.Fatalf("records = %#v, %v", records, err)
	}
	tunnels, err := client.Tunnels(context.Background())
	if err != nil || len(tunnels) != 1 || tunnels[0].Status != "healthy" {
		t.Fatalf("tunnels = %#v, %v", tunnels, err)
	}
	routes, err := client.TunnelRoutes(context.Background(), testTunnelID)
	if err != nil || len(routes) != 1 || routes[0].OriginKind != "http" {
		t.Fatalf("routes = %#v, %v", routes, err)
	}
	encoded, _ := json.Marshal([]any{zones, records, tunnels, routes})
	for _, forbidden := range []string{token, "private-host", "8080", "origin_ip", "caPool", "must-not-leak"} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("mapped response leaked %q: %s", forbidden, encoded)
		}
	}
	wantPaths := []string{
		"/client/v4/zones?page=1&per_page=50",
		"/client/v4/zones/" + testZoneID + "/dns_records?page=1&per_page=500",
		"/client/v4/accounts/" + testAccountID + "/cfd_tunnel?page=1&per_page=1000",
		"/client/v4/accounts/" + testAccountID + "/cfd_tunnel/" + testTunnelID + "/configurations",
	}
	if strings.Join(requests, "\n") != strings.Join(wantPaths, "\n") {
		t.Fatalf("requests = %v", requests)
	}
}

func TestPaginationFollowsResultInfo(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		page := request.URL.Query().Get("page")
		_, _ = io.WriteString(writer, `{"success":true,"result":[{"id":"`+page+`","name":"zone-`+page+`"}],"result_info":{"page":`+page+`,"total_pages":2}}`)
	}))
	defer server.Close()
	client := testClient(server, "token", "")
	zones, err := client.Zones(context.Background())
	if err != nil || len(zones) != 2 || zones[1].Name != "zone-2" {
		t.Fatalf("zones = %#v, %v", zones, err)
	}
}

func TestOptionalConfigurationAndTunnelAccount(t *testing.T) {
	client, err := NewClient("", "", testLogger())
	if err != nil || client.Configured() || client.TunnelsConfigured() {
		t.Fatalf("optional client = %#v, %v", client, err)
	}
	if _, err := client.Zones(context.Background()); err != ErrNotConfigured {
		t.Fatalf("unconfigured zones error = %v", err)
	}
	zonesOnly, err := NewClient("token", "", testLogger())
	if err != nil || !zonesOnly.Configured() || zonesOnly.TunnelsConfigured() {
		t.Fatalf("zones-only client = %#v, %v", zonesOnly, err)
	}
	if _, err := zonesOnly.Tunnels(context.Background()); err != ErrNotConfigured {
		t.Fatalf("unconfigured tunnels error = %v", err)
	}
	for _, accountID := range []string{"too-short", strings.Repeat("z", 32)} {
		if _, err := NewClient("token", accountID, testLogger()); err == nil {
			t.Fatalf("account ID %q was accepted", accountID)
		}
	}
	if _, err := NewClient("", testAccountID, testLogger()); err == nil {
		t.Fatal("account ID without token was accepted")
	}
}

func TestIdentifiersAreValidatedBeforeRequests(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Error("invalid identifier reached Cloudflare")
	}))
	defer server.Close()
	client := testClient(server, "token", testAccountID)
	if _, err := client.DNSRecords(context.Background(), "../secrets"); err == nil {
		t.Fatal("invalid zone ID was accepted")
	}
	if _, err := client.TunnelRoutes(context.Background(), "../secrets"); err == nil {
		t.Fatal("invalid tunnel ID was accepted")
	}
}

func TestFailuresAreBoundedAndSanitized(t *testing.T) {
	for _, test := range []struct {
		name   string
		status int
		body   string
	}{
		{"rejected", http.StatusUnauthorized, `{"errors":[{"message":"token cloudflare-server-secret rejected"}]}`},
		{"unsuccessful envelope", http.StatusOK, `{"success":false,"errors":[{"message":"private failure"}]}`},
		{"invalid JSON", http.StatusOK, `{not-json`},
		{"oversized", http.StatusOK, strings.Repeat("x", maximumResponseBytes+1)},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.WriteHeader(test.status)
				_, _ = io.WriteString(writer, test.body)
			}))
			defer server.Close()
			client := testClient(server, "cloudflare-server-secret", "")
			_, err := client.Zones(context.Background())
			if err != ErrUnavailable || strings.Contains(err.Error(), "secret") || strings.Contains(err.Error(), "private") {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestRedirectPolicyBlocksDifferentOrigins(t *testing.T) {
	base, _ := url.Parse(defaultBaseURL)
	policy := sameOriginRedirects(base)
	if err := policy(&http.Request{URL: mustURL(t, defaultBaseURL+"/zones")}, nil); err != nil {
		t.Fatalf("same-origin redirect = %v", err)
	}
	if err := policy(&http.Request{URL: mustURL(t, "https://evil.example.test/steal")}, nil); err != http.ErrUseLastResponse {
		t.Fatalf("cross-origin redirect = %v", err)
	}
}

func testClient(server *httptest.Server, token, accountID string) *Client {
	base, _ := url.Parse(server.URL + "/client/v4")
	return &Client{baseURL: base, token: token, accountID: accountID, http: server.Client(), logger: testLogger()}
}

func mustURL(t *testing.T, value string) *url.URL {
	t.Helper()
	parsed, err := url.Parse(value)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
