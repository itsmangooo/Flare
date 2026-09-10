package alerts

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/itsmangooo/flare/server/backend/internal/auth"
	"github.com/pashagolub/pgxmock/v4"
)

func TestAlertHistoryIsPerUserPaginatedAndIncludesUnreadCount(t *testing.T) {
	database, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	userID := uuid.New()
	now := time.Date(2026, time.September, 6, 12, 0, 0, 0, time.UTC)
	rows := pgxmock.NewRows([]string{
		"Id", "Kind", "Severity", "Title", "Message", "Source", "ResourceType", "ResourceId",
		"Status", "FirstSeenAt", "LastSeenAt", "RecoveredAt", "OccurrenceCount", "ReadAt",
	}).
		AddRow(uuid.New(), "container.unhealthy", "critical", "Container unhealthy", "Health check failed.", "docker", "container", "abc", "active", now, now, nil, 3, nil).
		AddRow(uuid.New(), "docker.unavailable", "critical", "Docker unavailable", "Docker could not be reached.", "docker", nil, nil, "recovered", now.Add(-time.Hour), now.Add(-time.Minute), now, 1, nil).
		AddRow(uuid.New(), "host.threshold", "warning", "High memory", "Memory exceeded threshold.", "host", "host", "homelab", "active", now, now, nil, 1, nil)
	database.ExpectQuery(regexp.QuoteMeta(listAlertsQuery)).
		WithArgs(userID, true, 3, int64(2)).WillReturnRows(rows)
	database.ExpectQuery(regexp.QuoteMeta(unreadCountQuery)).
		WithArgs(userID).WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(int64(7)))

	var logs bytes.Buffer
	handler := NewHistoryHandler(database, slog.New(slog.NewTextHandler(&logs, nil)))
	handler.currentUser = func(context.Context) (auth.User, bool) { return auth.User{ID: userID}, true }
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/?page=2&pageSize=2&unreadOnly=true", nil))
	body := response.Body.String()
	for _, expected := range []string{`"severity":"critical"`, `"resourceType":"container"`, `"status":"recovered"`, `"unreadCount":7`, `"hasMore":true`} {
		if !strings.Contains(body, expected) {
			t.Fatalf("response missing %s: %s logs=%s", expected, body, logs.String())
		}
	}
	if strings.Contains(body, "High memory") {
		t.Fatalf("look-ahead alert leaked into page: %s", body)
	}
	if err = database.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAlertReadStateCanBeSetAndCleared(t *testing.T) {
	database, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	userID, alertID := uuid.New(), uuid.New()
	database.ExpectExec(`INSERT INTO "AlertReads"`).WithArgs(alertID, userID, pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	database.ExpectExec(`DELETE FROM "AlertReads"`).WithArgs(alertID, userID).
		WillReturnResult(pgxmock.NewResult("DELETE", 1))
	handler := NewHistoryHandler(database, alertTestLogger())
	handler.currentUser = func(context.Context) (auth.User, bool) { return auth.User{ID: userID}, true }

	readResponse := httptest.NewRecorder()
	handler.ServeHTTP(readResponse, httptest.NewRequest(http.MethodPut, "/"+alertID.String()+"/read", nil))
	if readResponse.Code != http.StatusNoContent {
		t.Fatalf("mark read response = %d %s", readResponse.Code, readResponse.Body.String())
	}
	unreadResponse := httptest.NewRecorder()
	handler.ServeHTTP(unreadResponse, httptest.NewRequest(http.MethodDelete, "/"+alertID.String()+"/read", nil))
	if unreadResponse.Code != http.StatusNoContent {
		t.Fatalf("mark unread response = %d %s", unreadResponse.Code, unreadResponse.Body.String())
	}
	if err = database.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAlertHistoryRejectsInvalidInputAndSanitizesFailures(t *testing.T) {
	database, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	userID := uuid.New()
	handler := NewHistoryHandler(database, alertTestLogger())
	handler.currentUser = func(context.Context) (auth.User, bool) { return auth.User{ID: userID}, true }

	invalid := httptest.NewRecorder()
	handler.ServeHTTP(invalid, httptest.NewRequest(http.MethodPut, "/not-a-uuid/read", nil))
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("invalid response = %d %s", invalid.Code, invalid.Body.String())
	}
	database.ExpectQuery(regexp.QuoteMeta(listAlertsQuery)).WithArgs(userID, false, 31, int64(0)).
		WillReturnError(errors.New("password=private host=db.internal"))
	failure := httptest.NewRecorder()
	handler.ServeHTTP(failure, httptest.NewRequest(http.MethodGet, "/", nil))
	if failure.Code != http.StatusInternalServerError || strings.Contains(failure.Body.String(), "private") || strings.Contains(failure.Body.String(), "db.internal") {
		t.Fatalf("failure response = %d %s", failure.Code, failure.Body.String())
	}
	if err = database.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAlertHistoryRequiresAuthenticatedContext(t *testing.T) {
	database, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	response := httptest.NewRecorder()
	NewHistoryHandler(database, alertTestLogger()).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
}

func TestNotificationPreferencesDefaultAndAdministratorUpdate(t *testing.T) {
	database, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	userID := uuid.New()
	database.ExpectQuery(regexp.QuoteMeta(preferenceQuery)).WillReturnRows(pgxmock.NewRows([]string{
		"Enabled", "MinimumSeverity", "RecoveryEnabled", "DockerEnabled", "CoolifyEnabled", "CloudflareEnabled", "HostEnabled",
	}))
	handler := NewHistoryHandler(database, alertTestLogger())
	handler.currentUser = func(context.Context) (auth.User, bool) { return auth.User{ID: userID}, true }
	handler.hasRole = func(context.Context, string) bool { return true }
	handler.now = func() time.Time { return time.Date(2026, time.September, 6, 13, 0, 0, 0, time.UTC) }

	getResponse := httptest.NewRecorder()
	handler.ServeHTTP(getResponse, httptest.NewRequest(http.MethodGet, "/preferences", nil))
	if getResponse.Code != http.StatusOK || !strings.Contains(getResponse.Body.String(), `"minimumSeverity":"warning"`) || !strings.Contains(getResponse.Body.String(), `"dockerEnabled":true`) {
		t.Fatalf("default preferences = %d %s", getResponse.Code, getResponse.Body.String())
	}
	database.ExpectExec(`INSERT INTO "NotificationPreferences"`).WithArgs(
		true, "critical", false, true, false, false, true, handler.now(), userID,
	).WillReturnResult(pgxmock.NewResult("INSERT", 1))
	putResponse := httptest.NewRecorder()
	body := `{"enabled":true,"minimumSeverity":" CRITICAL ","recoveryEnabled":false,"dockerEnabled":true,"coolifyEnabled":false,"cloudflareEnabled":false,"hostEnabled":true}`
	handler.ServeHTTP(putResponse, httptest.NewRequest(http.MethodPut, "/preferences", strings.NewReader(body)))
	if putResponse.Code != http.StatusOK || !strings.Contains(putResponse.Body.String(), `"minimumSeverity":"critical"`) {
		t.Fatalf("updated preferences = %d %s", putResponse.Code, putResponse.Body.String())
	}
	if err = database.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestNotificationPreferencesRequireAdministrator(t *testing.T) {
	database, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	handler := NewHistoryHandler(database, alertTestLogger())
	handler.currentUser = func(context.Context) (auth.User, bool) { return auth.User{ID: uuid.New()}, true }
	handler.hasRole = func(context.Context, string) bool { return false }
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPut, "/preferences", strings.NewReader(`{}`)))
	if response.Code != http.StatusForbidden {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
}

func alertTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
