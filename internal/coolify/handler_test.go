package coolify

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/itsmangooo/flare/internal/auth"
	"github.com/itsmangooo/flare/internal/config"
	"github.com/pashagolub/pgxmock/v4"
)

type fakeReadService struct {
	resourceUUID   string
	deploymentUUID string
	page           int
	pageSize       int
	action         string
	target         string
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

func (service *fakeReadService) StartApplication(_ context.Context, target string) (ActionResponse, error) {
	service.action, service.target = "start", target
	return ActionResponse{Message: "queued"}, service.err
}

func (service *fakeReadService) StopApplication(_ context.Context, target string) (ActionResponse, error) {
	service.action, service.target = "stop", target
	return ActionResponse{Message: "queued"}, service.err
}

func (service *fakeReadService) RestartApplication(_ context.Context, target string) (ActionResponse, error) {
	service.action, service.target = "restart", target
	return ActionResponse{Message: "queued"}, service.err
}

func (service *fakeReadService) RestartService(_ context.Context, target string) (ActionResponse, error) {
	service.action, service.target = "service-restart", target
	return ActionResponse{Message: "queued"}, service.err
}

func (service *fakeReadService) RedeployApplication(_ context.Context, target string) (ActionResponse, error) {
	service.action, service.target = "redeploy", target
	return ActionResponse{Message: "queued"}, service.err
}

func TestReadRoutesPreserveFlutterContract(t *testing.T) {
	service := &fakeReadService{}
	handler := NewHandler(service, nil, slog.New(slog.DiscardHandler))
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
			NewHandler(&fakeReadService{err: test.err}, nil, slog.New(slog.DiscardHandler)).ServeHTTP(
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
	handler := NewHandler(&fakeReadService{}, nil, slog.New(slog.DiscardHandler))
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

func TestAdministrativeActionsAreAllowlistedAndAudited(t *testing.T) {
	for _, test := range []struct {
		path        string
		serviceCall string
		auditAction string
		target      string
	}{
		{"/applications/app_1/start", "start", "coolify.application.start", "app_1"},
		{"/applications/app_1/stop", "stop", "coolify.application.stop", "app_1"},
		{"/applications/app_1/restart", "restart", "coolify.application.restart", "app_1"},
		{"/applications/app_1/redeploy", "redeploy", "coolify.application.redeploy", "app_1"},
		{"/services/service_1/restart", "service-restart", "coolify.service.restart", "service_1"},
	} {
		t.Run(test.serviceCall, func(t *testing.T) {
			database, err := pgxmock.NewPool()
			if err != nil {
				t.Fatal(err)
			}
			defer database.Close()
			userID := uuid.New()
			database.ExpectExec(`INSERT INTO "AuditEvents"`).WithArgs(
				pgxmock.AnyArg(), userID, "admin@example.com", test.auditAction, test.target,
				pgxmock.AnyArg(), 0, "", nil).WillReturnResult(pgxmock.NewResult("INSERT", 1))
			service := &fakeReadService{}
			response := serveCoolifyAuthenticated(t, database, service, userID, []string{"Administrator"}, test.path)
			if response.Code != http.StatusAccepted || service.action != test.serviceCall || service.target != test.target {
				t.Fatalf("response=%d %s service=%#v", response.Code, response.Body.String(), service)
			}
			if err := database.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestActionRequiresAdministrator(t *testing.T) {
	database, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	service := &fakeReadService{}
	response := serveCoolifyAuthenticated(t, database, service, uuid.New(), nil, "/applications/app_1/restart")
	if response.Code != http.StatusForbidden || service.action != "" {
		t.Fatalf("response=%d %s service=%#v", response.Code, response.Body.String(), service)
	}
}

func TestFailedActionIsAuditedWithoutLeakingUpstreamError(t *testing.T) {
	database, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	userID := uuid.New()
	database.ExpectExec(`INSERT INTO "AuditEvents"`).WithArgs(
		pgxmock.AnyArg(), userID, "admin@example.com", "coolify.application.start", "app_1",
		pgxmock.AnyArg(), 1, "", "Coolify operation failed.").WillReturnResult(pgxmock.NewResult("INSERT", 1))
	service := &fakeReadService{err: errors.New("upstream token=must-not-leak")}
	response := serveCoolifyAuthenticated(t, database, service, userID, []string{"Administrator"}, "/applications/app_1/start")
	if response.Code != http.StatusServiceUnavailable || strings.Contains(response.Body.String(), "token") || strings.Contains(response.Body.String(), "must-not-leak") {
		t.Fatalf("response=%d %s", response.Code, response.Body.String())
	}
	if err := database.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestActionLimiterAllowsThirtyRequestsPerMinute(t *testing.T) {
	limiter := actionLimiter{windows: make(map[string]actionWindow)}
	now := time.Now()
	for attempt := 1; attempt <= 31; attempt++ {
		if allowed := limiter.allow("user:address", now); allowed != (attempt <= 30) {
			t.Fatalf("attempt %d allowed=%v", attempt, allowed)
		}
	}
	if !limiter.allow("user:address", now.Add(time.Minute)) {
		t.Fatal("new window should allow an action")
	}
}

func serveCoolifyAuthenticated(t *testing.T, database pgxmock.PgxPoolIface, service readService,
	userID uuid.UUID, roles []string, path string) *httptest.ResponseRecorder {
	t.Helper()
	cfg := config.Config{JWTSigningKey: "test-signing-key-that-is-at-least-32-bytes", JWTIssuer: "Flare.Api", JWTAudience: "Flare.Mobile"}
	claims := jwt.MapClaims{"sub": userID.String(), "email": "admin@example.com", "iss": cfg.JWTIssuer,
		"aud": cfg.JWTAudience, "exp": time.Now().Add(time.Minute).Unix()}
	if len(roles) == 1 {
		claims["http://schemas.microsoft.com/ws/2008/06/identity/claims/role"] = roles[0]
	} else if len(roles) > 1 {
		claims["http://schemas.microsoft.com/ws/2008/06/identity/claims/role"] = roles
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(cfg.JWTSigningKey))
	if err != nil {
		t.Fatal(err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := auth.NewHandler(cfg, database, logger).Authenticate(NewHandler(service, database, logger))
	request := httptest.NewRequest(http.MethodPost, path, nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
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
