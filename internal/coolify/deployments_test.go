package coolify

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestApplicationDeploymentsMapsCompatibleContract(t *testing.T) {
	logs := strings.Repeat("x", 1_000_001)
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/v1/deployments/applications/app_1" || request.URL.Query().Get("skip") != "2" || request.URL.Query().Get("take") != "100" {
			t.Errorf("request = %s", request.URL.RequestURI())
			http.NotFound(writer, request)
			return
		}
		_, _ = io.WriteString(writer, `[{"deployment_uuid":"deploy_1","application_name":"api","status":"finished","git_branch":"main","commit":"abc123","commit_message":"ship it","created_at":"2026-09-05T10:00:00Z","finished_at":"2026-09-05T10:01:24.5Z","logs":"`+logs+`"}]`)
	}))
	defer server.Close()
	client := testClient(t, server, "token")

	items, err := client.ApplicationDeployments(context.Background(), "app_1", 2, 999)
	if err != nil || len(items) != 1 {
		t.Fatalf("deployments = %#v, %v", items, err)
	}
	item := items[0]
	if item.UUID != "deploy_1" || item.ResourceUUID != "app_1" || item.ResourceName != "api" {
		t.Fatalf("identity = %#v", item)
	}
	if item.Duration == nil || *item.Duration != "00:01:24.5000000" {
		t.Fatalf("duration = %v", item.Duration)
	}
	if item.Logs == nil || len(*item.Logs) != 1_000_000 {
		t.Fatalf("logs length = %v", item.Logs)
	}
}

func TestDeploymentDetailUsesDocumentedEndpointAndRunningHasNoFinish(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/v1/deployments/deploy_1" {
			http.NotFound(writer, request)
			return
		}
		_, _ = io.WriteString(writer, `{"uuid":"deploy_1","resource_uuid":"app_1","resource_name":"api","status":"in_progress","started_at":"2026-09-05T10:00:00Z","updated_at":"2026-09-05T10:01:00Z"}`)
	}))
	defer server.Close()
	client := testClient(t, server, "token")

	item, err := client.Deployment(context.Background(), "deploy_1")
	if err != nil {
		t.Fatal(err)
	}
	if item.FinishedAt != nil || item.Duration != nil || item.StartedAt == nil {
		t.Fatalf("running deployment times = %#v", item)
	}
}

func TestDeploymentPageAggregatesApplicationsAndSortsNewestFirst(t *testing.T) {
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/api/v1/applications":
			_, _ = io.WriteString(writer, `[{"uuid":"app_1","name":"one"},{"uuid":"app_2","name":"two"}]`)
		case "/api/v1/deployments/applications/app_1":
			_, _ = io.WriteString(writer, `[{"deployment_uuid":"old","application_name":"one","created_at":"2026-09-05T08:00:00Z"}]`)
		case "/api/v1/deployments/applications/app_2":
			_, _ = io.WriteString(writer, `[{"deployment_uuid":"new","application_name":"two","created_at":"2026-09-05T09:00:00Z"},{"deployment_uuid":"middle","application_name":"two","created_at":"2026-09-05T08:30:00Z"}]`)
		default:
			http.NotFound(writer, request)
		}
	}))
	server.Config.ErrorLog = log.New(io.Discard, "", 0)
	server.StartTLS()
	defer server.Close()
	client := testClient(t, server, "token")

	first, err := client.Deployments(context.Background(), 1, 2)
	if err != nil || len(first.Items) != 2 || first.Items[0].UUID != "new" || first.Items[1].UUID != "middle" || !first.HasMore {
		t.Fatalf("first page = %#v, %v", first, err)
	}
	second, err := client.Deployments(context.Background(), 2, 2)
	if err != nil || len(second.Items) != 1 || second.Items[0].UUID != "old" || second.HasMore {
		t.Fatalf("second page = %#v, %v", second, err)
	}
}

func TestDeploymentValidationAndNotFound(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if strings.Contains(request.URL.Path, "missing") {
			http.NotFound(writer, request)
			return
		}
		t.Error("invalid identifier reached Coolify")
	}))
	defer server.Close()
	client := testClient(t, server, "token")
	if _, err := client.Deployment(context.Background(), "../secrets"); err == nil {
		t.Fatal("invalid deployment ID was accepted")
	}
	if _, err := client.ApplicationDeployments(context.Background(), "../secrets", 0, 30); err == nil {
		t.Fatal("invalid application ID was accepted")
	}
	if _, err := client.Deployment(context.Background(), "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("not found error = %v", err)
	}
}

func TestDotNetDuration(t *testing.T) {
	if value := dotNetDuration(26*time.Hour + 3*time.Minute + 4*time.Second + 500*time.Millisecond); value != "1.02:03:04.5000000" {
		t.Fatalf("duration = %s", value)
	}
	if value := dotNetDuration(4 * time.Second); value != "00:00:04" {
		t.Fatalf("whole duration = %s", value)
	}
}
