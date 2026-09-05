package coolify

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeReadService struct {
	resourceUUID   string
	deploymentUUID string
	page           int
	pageSize       int
	err            error
}

func (service *fakeReadService) Servers(context.Context) ([]Server, error) {
	return []Server{{UUID: "server_1", Name: "lab"}}, service.err
}

func (service *fakeReadService) ServerResources(_ context.Context, uuid string) ([]Resource, error) {
	service.resourceUUID = uuid
	return []Resource{{UUID: "resource_1", Name: "api", Type: "application"}}, service.err
}

func (service *fakeReadService) Applications(context.Context) ([]Application, error) {
	return []Application{{UUID: "app_1", Name: "api"}}, service.err
}

func (service *fakeReadService) Services(context.Context) ([]Service, error) {
	return []Service{{UUID: "service_1", Name: "postgres"}}, service.err
}

func (service *fakeReadService) Deployments(_ context.Context, page, pageSize int) (DeploymentPage, error) {
	service.page, service.pageSize = page, pageSize
	return DeploymentPage{Items: []Deployment{{UUID: "deployment_1"}}, Page: page, PageSize: pageSize}, service.err
}

func (service *fakeReadService) Deployment(_ context.Context, uuid string) (Deployment, error) {
	service.deploymentUUID = uuid
	return Deployment{UUID: uuid}, service.err
}

func TestReadRoutesPreserveFlutterContract(t *testing.T) {
	service := &fakeReadService{}
	handler := NewHandler(service, slog.New(slog.DiscardHandler))
	for _, test := range []struct {
		path string
		want string
	}{
		{"/servers", `[{"uuid":"server_1","name":"lab","isReachable":null,"isUsable":null}]`},
		{"/servers/server_1/resources", `[{"uuid":"resource_1","name":"api","type":"application","status":null}]`},
		{"/applications", `[{"uuid":"app_1","name":"api","status":null,"fqdn":null,"gitBranch":null}]`},
		{"/services", `[{"uuid":"service_1","name":"postgres","status":null,"description":null}]`},
		{"/deployments?page=2&pageSize=99", `{"items":[{"uuid":"deployment_1","resourceUuid":"","resourceName":"","status":null,"branch":null,"commit":null,"commitMessage":null,"startedAt":null,"finishedAt":null,"duration":null,"logs":null}],"page":2,"pageSize":50,"hasMore":false}`},
		{"/deployments/deployment_1", `{"uuid":"deployment_1","resourceUuid":"","resourceName":"","status":null,"branch":null,"commit":null,"commitMessage":null,"startedAt":null,"finishedAt":null,"duration":null,"logs":null}`},
	} {
		t.Run(test.path, func(t *testing.T) {
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, test.path, nil))
			if response.Code != http.StatusOK || response.Header().Get("Content-Type") != "application/json; charset=utf-8" {
				t.Fatalf("response = %d %s", response.Code, response.Body.String())
			}
			assertJSON(t, response.Body.Bytes(), test.want)
		})
	}
	if service.resourceUUID != "server_1" || service.deploymentUUID != "deployment_1" || service.page != 2 || service.pageSize != 50 {
		t.Fatalf("captured values = %#v", service)
	}
}

func TestHandlerMapsSafeProblemDetails(t *testing.T) {
	for _, test := range []struct {
		name   string
		err    error
		status int
		detail string
	}{
		{"invalid", ErrInvalidIdentifier, http.StatusBadRequest, "Coolify identifier is invalid."},
		{"missing", ErrNotFound, http.StatusNotFound, "The requested Coolify resource does not exist."},
		{"not configured", ErrNotConfigured, http.StatusServiceUnavailable, "Coolify integration is not configured."},
		{"upstream secret", errors.New("token=must-not-leak"), http.StatusServiceUnavailable, "Coolify is unavailable."},
	} {
		t.Run(test.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			NewHandler(&fakeReadService{err: test.err}, slog.New(slog.DiscardHandler)).ServeHTTP(
				response, httptest.NewRequest(http.MethodGet, "/servers", nil))
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

func TestHandlerRejectsInvalidPaginationAndMethods(t *testing.T) {
	handler := NewHandler(&fakeReadService{}, slog.New(slog.DiscardHandler))
	for _, path := range []string{"/deployments?page=nope", "/deployments?page=1000001", "/deployments?pageSize=nope"} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusBadRequest {
			t.Fatalf("%s response = %d", path, response.Code)
		}
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/servers", nil))
	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("method response = %d", response.Code)
	}
}

func assertJSON(t *testing.T, actual []byte, expected string) {
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
