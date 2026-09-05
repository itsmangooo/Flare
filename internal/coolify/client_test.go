package coolify

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

func TestReadOnlyMethodsUseDocumentedEndpointsAndSanitizePayloads(t *testing.T) {
	token := "server-side-secret"
	requests := make([]string, 0, 4)
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer "+token || request.Header.Get("Accept") != "application/json" {
			t.Errorf("missing Coolify authentication headers")
			writer.WriteHeader(http.StatusBadRequest)
			return
		}
		requests = append(requests, request.URL.Path)
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/api/v1/servers":
			_, _ = io.WriteString(writer, `[{"uuid":"srv_1","name":"lab","ip":"10.0.0.2","settings":{"is_reachable":true,"is_usable":false,"sentinel_token":"must-not-leak"}}]`)
		case "/api/v1/servers/srv_1/resources":
			_, _ = io.WriteString(writer, `[{"uuid":"res_1","name":"api","type":"application","status":"running","docker_compose":"must-not-leak"}]`)
		case "/api/v1/applications":
			_, _ = io.WriteString(writer, `[{"uuid":"app_1","name":"web","status":"running","fqdn":"https://web.example.test","git_branch":"main","manual_webhook_secret_github":"must-not-leak"}]`)
		case "/api/v1/services":
			_, _ = io.WriteString(writer, `[{"uuid":"svc_1","name":"database","status":"running","description":"PostgreSQL","docker_compose_raw":"must-not-leak"}]`)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	client := testClient(t, server, token)

	servers, err := client.Servers(context.Background())
	if err != nil || len(servers) != 1 || servers[0].IsReachable == nil || !*servers[0].IsReachable {
		t.Fatalf("servers = %#v, %v", servers, err)
	}
	resources, err := client.ServerResources(context.Background(), "srv_1")
	if err != nil || len(resources) != 1 || resources[0].Type != "application" {
		t.Fatalf("resources = %#v, %v", resources, err)
	}
	applications, err := client.Applications(context.Background())
	if err != nil || len(applications) != 1 || applications[0].GitBranch == nil || *applications[0].GitBranch != "main" {
		t.Fatalf("applications = %#v, %v", applications, err)
	}
	services, err := client.Services(context.Background())
	if err != nil || len(services) != 1 || services[0].Description == nil || *services[0].Description != "PostgreSQL" {
		t.Fatalf("services = %#v, %v", services, err)
	}
	encoded, _ := json.Marshal([]any{servers, resources, applications, services})
	for _, forbidden := range []string{"10.0.0.2", "sentinel_token", "must-not-leak", "docker_compose", token} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("mapped response leaked %q: %s", forbidden, encoded)
		}
	}
	if strings.Join(requests, ",") != "/api/v1/servers,/api/v1/servers/srv_1/resources,/api/v1/applications,/api/v1/services" {
		t.Fatalf("paths = %v", requests)
	}
}

func TestClientHandlesOptionalConfigurationAndValidation(t *testing.T) {
	client, err := NewClient("", "", testLogger())
	if err != nil || client.Configured() {
		t.Fatalf("optional client = %#v, %v", client, err)
	}
	if _, err := client.Servers(context.Background()); err != ErrNotConfigured {
		t.Fatalf("unconfigured error = %v", err)
	}
	invalid := []struct{ base, token string }{
		{"http://coolify.example.test", "token"},
		{"https://user@coolify.example.test", "token"},
		{"https://coolify.example.test?token=value", "token"},
		{"https://coolify.example.test", ""},
	}
	for _, item := range invalid {
		if _, err := NewClient(item.base, item.token, testLogger()); err == nil {
			t.Fatalf("configuration %q was accepted", item.base)
		}
	}
}

func TestClientRejectsInvalidIdentifiersWithoutRequest(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Error("invalid identifier reached Coolify")
	}))
	defer server.Close()
	client := testClient(t, server, "token")
	if _, err := client.ServerResources(context.Background(), "../secrets"); err == nil {
		t.Fatal("invalid identifier was accepted")
	}
}

func TestClientCapsAndSanitizesUpstreamFailures(t *testing.T) {
	for _, test := range []struct {
		name   string
		status int
		body   string
	}{
		{"rejected", http.StatusUnauthorized, `{"message":"token server-side-secret rejected"}`},
		{"invalid JSON", http.StatusOK, `{not-json`},
		{"oversized", http.StatusOK, strings.Repeat("x", maximumResponseBytes+1)},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.WriteHeader(test.status)
				_, _ = io.WriteString(writer, test.body)
			}))
			defer server.Close()
			client := testClient(t, server, "server-side-secret")
			_, err := client.Servers(context.Background())
			if err != ErrUnavailable || strings.Contains(err.Error(), "secret") {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestRedirectPolicyBlocksDifferentOrigins(t *testing.T) {
	base, _ := url.Parse("https://coolify.example.test/api/v1")
	policy := sameOriginRedirects(base)
	if err := policy(&http.Request{URL: mustURL(t, "https://coolify.example.test/login")}, nil); err != nil {
		t.Fatalf("same-origin redirect = %v", err)
	}
	if err := policy(&http.Request{URL: mustURL(t, "https://evil.example.test/steal")}, nil); err != http.ErrUseLastResponse {
		t.Fatalf("cross-origin redirect = %v", err)
	}
}

func testClient(t *testing.T, server *httptest.Server, token string) *Client {
	t.Helper()
	base, configured, err := parseBaseURL(server.URL, token)
	if err != nil || !configured {
		t.Fatalf("test configuration = %v", err)
	}
	return &Client{baseURL: base, token: token, http: server.Client(), logger: testLogger()}
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
