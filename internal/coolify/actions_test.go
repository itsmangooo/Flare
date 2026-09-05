package coolify

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestActionsUseAllowlistedOfficialEndpoints(t *testing.T) {
	const token = "server-side-secret"
	want := map[string]string{
		"/api/v1/applications/app_1/start":   "",
		"/api/v1/applications/app_1/stop":    "",
		"/api/v1/applications/app_1/restart": "",
		"/api/v1/services/service_1/restart": "latest=false",
		"/api/v1/deploy":                     "force=false&uuid=app_1",
	}
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		query, found := want[request.URL.Path]
		if !found || request.Method != http.MethodPost || request.URL.RawQuery != query {
			t.Errorf("unexpected request %s %s", request.Method, request.URL.RequestURI())
			http.NotFound(writer, request)
			return
		}
		if request.Header.Get("Authorization") != "Bearer "+token {
			t.Error("missing server-side bearer token")
		}
		if request.URL.Path == "/api/v1/deploy" {
			_, _ = io.WriteString(writer, `{"deployments":[{"deployment_uuid":"deployment_2"}]}`)
			return
		}
		_, _ = io.WriteString(writer, `{"message":"Operation queued.","deployment_uuid":"deployment_1"}`)
	}))
	defer server.Close()
	client := testClient(t, server, token)

	operations := []func(context.Context) (ActionResponse, error){
		func(ctx context.Context) (ActionResponse, error) { return client.StartApplication(ctx, "app_1") },
		func(ctx context.Context) (ActionResponse, error) { return client.StopApplication(ctx, "app_1") },
		func(ctx context.Context) (ActionResponse, error) { return client.RestartApplication(ctx, "app_1") },
		func(ctx context.Context) (ActionResponse, error) { return client.RestartService(ctx, "service_1") },
	}
	for _, operation := range operations {
		response, err := operation(context.Background())
		if err != nil || response.OperationID == nil || *response.OperationID != "deployment_1" {
			t.Fatalf("action response = %#v, %v", response, err)
		}
	}
	redeploy, err := client.RedeployApplication(context.Background(), "app_1")
	if err != nil || redeploy.OperationID == nil || *redeploy.OperationID != "deployment_2" {
		t.Fatalf("redeploy response = %#v, %v", redeploy, err)
	}
}

func TestActionsValidateIdentifiersBeforeRequest(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Error("invalid action reached Coolify")
	}))
	defer server.Close()
	client := testClient(t, server, "token")
	for _, operation := range []func(context.Context) (ActionResponse, error){
		func(ctx context.Context) (ActionResponse, error) { return client.StartApplication(ctx, "../secrets") },
		func(ctx context.Context) (ActionResponse, error) { return client.StopApplication(ctx, "../secrets") },
		func(ctx context.Context) (ActionResponse, error) { return client.RestartApplication(ctx, "../secrets") },
		func(ctx context.Context) (ActionResponse, error) { return client.RestartService(ctx, "../secrets") },
		func(ctx context.Context) (ActionResponse, error) {
			return client.RedeployApplication(ctx, "../secrets")
		},
	} {
		if _, err := operation(context.Background()); err != ErrInvalidIdentifier {
			t.Fatalf("validation error = %v", err)
		}
	}
}

func TestActionsSanitizeFailuresAndHandleEmptyResponses(t *testing.T) {
	for _, test := range []struct {
		name   string
		status int
		body   string
		wantOK bool
	}{
		{"empty success", http.StatusAccepted, "", true},
		{"rejected", http.StatusBadGateway, `{"message":"token must-not-leak"}`, false},
		{"invalid JSON", http.StatusOK, `{not-json`, false},
		{"oversized", http.StatusOK, strings.Repeat("x", maximumResponseBytes+1), false},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.WriteHeader(test.status)
				_, _ = io.WriteString(writer, test.body)
			}))
			defer server.Close()
			client := testClient(t, server, "token")
			response, err := client.StartApplication(context.Background(), "app_1")
			if test.wantOK {
				if err != nil || response.Message != "Operation queued." {
					t.Fatalf("response = %#v, %v", response, err)
				}
				return
			}
			if err != ErrUnavailable || strings.Contains(err.Error(), "must-not-leak") {
				t.Fatalf("error = %v", err)
			}
		})
	}
}
