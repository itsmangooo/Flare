package activity

import (
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
	"github.com/pashagolub/pgxmock/v4"
)

func TestActivityFeedCombinesTransformsAndPaginatesEvents(t *testing.T) {
	database, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	now := time.Date(2026, time.September, 5, 11, 0, 0, 0, time.UTC)
	actor := "admin@example.com"
	database.ExpectQuery(regexp.QuoteMeta(activityQuery)).WithArgs(3, int64(2)).WillReturnRows(
		pgxmock.NewRows([]string{"Id", "Kind", "Action", "Target", "Timestamp", "Result", "Actor"}).
			AddRow(uuid.New(), "Infrastructure", "container.restart", "postgres", now, 0, actor).
			AddRow(uuid.New(), "Security", "auth.login", "admin@example.com", now.Add(-time.Minute), 1, actor).
			AddRow(uuid.New(), "Infrastructure", "deployment.failed", "api", now.Add(-2*time.Minute), 1, nil))

	response := httptest.NewRecorder()
	NewHandler(database, testLogger()).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/?page=2&pageSize=2", nil))
	body := response.Body.String()
	for _, expected := range []string{`"action":"Container restarted"`, `"kind":"Security"`, `"result":"Failed"`, `"page":2`, `"pageSize":2`, `"hasMore":true`} {
		if !strings.Contains(body, expected) {
			t.Fatalf("response missing %s: %s", expected, body)
		}
	}
	if strings.Contains(body, "Deployment failed") {
		t.Fatalf("look-ahead event leaked into page: %s", body)
	}
	if err := database.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestActivityFeedReturnsEmptyArray(t *testing.T) {
	database, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	database.ExpectQuery(regexp.QuoteMeta(activityQuery)).WithArgs(31, int64(0)).WillReturnRows(
		pgxmock.NewRows([]string{"Id", "Kind", "Action", "Target", "Timestamp", "Result", "Actor"}))
	response := httptest.NewRecorder()
	NewHandler(database, testLogger()).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"items":[]`) {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
}

func TestActivityFeedRejectsInvalidPaginationWithoutQuerying(t *testing.T) {
	database, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	response := httptest.NewRecorder()
	NewHandler(database, testLogger()).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/?page=never", nil))
	if response.Code != http.StatusBadRequest || !strings.HasPrefix(response.Header().Get("Content-Type"), "application/problem+json") {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
}

func TestActivityFeedSanitizesDatabaseFailures(t *testing.T) {
	database, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	database.ExpectQuery(regexp.QuoteMeta(activityQuery)).WithArgs(31, int64(0)).WillReturnError(errors.New("password=private host=db.internal"))
	response := httptest.NewRecorder()
	NewHandler(database, testLogger()).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if response.Code != http.StatusInternalServerError || strings.Contains(response.Body.String(), "private") || strings.Contains(response.Body.String(), "db.internal") {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
}

func TestPaginationClampsCompatibleValues(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/?page=-2&pageSize=500", nil)
	page, pageSize, err := pagination(request)
	if err != nil || page != 1 || pageSize != 100 {
		t.Fatalf("pagination = %d %d %v", page, pageSize, err)
	}
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
