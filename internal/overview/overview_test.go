package overview

import (
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
	"github.com/itsmangooo/flare/internal/activity"
	"github.com/itsmangooo/flare/internal/telemetry"
	containertypes "github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
	"github.com/pashagolub/pgxmock/v4"
)

type fakeDocker struct {
	items []containertypes.Summary
	err   error
}

func (docker fakeDocker) ContainerList(context.Context, client.ContainerListOptions) (client.ContainerListResult, error) {
	return client.ContainerListResult{Items: docker.items}, docker.err
}

type fakeMetrics struct {
	value telemetry.Metrics
}

func (metrics fakeMetrics) Collect(context.Context) telemetry.Metrics { return metrics.value }

func TestOverviewPreservesFlutterContractWithRealSources(t *testing.T) {
	database, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	now := time.Date(2026, time.September, 5, 12, 0, 0, 0, time.UTC)
	database.ExpectQuery(regexp.QuoteMeta(historyQuery)).WithArgs(now.Add(-time.Hour)).WillReturnRows(
		pgxmock.NewRows([]string{"Timestamp", "CpuPercent", "MemoryPercent"}).AddRow(now.Add(-time.Minute), 42.5, 61.0))
	database.ExpectQuery(`SELECT "Id", "Kind", "Action"`).WithArgs(6, int64(0)).WillReturnRows(
		pgxmock.NewRows([]string{"Id", "Kind", "Action", "Target", "Timestamp", "Result", "Actor"}).
			AddRow(uuid.New(), "Infrastructure", "container.restart", "postgres", now.Add(-2*time.Minute), 0, nil))
	memoryTotal := int64(16 << 30)
	uptime := "18.04:00:00"
	handler := newHandler(
		fakeDocker{items: []containertypes.Summary{
			{State: "running", Status: "Up 4 hours (healthy)"},
			{State: "running", Status: "Up 2 minutes (unhealthy)"},
			{State: "exited", Status: "Exited (1) 10 minutes ago"},
		}},
		fakeMetrics{value: telemetry.Metrics{HostName: "lab-01", ObservedAt: now, MemoryTotalBytes: &memoryTotal, Uptime: &uptime}},
		database, activity.NewReader(database), testLogger(), func() time.Time { return now },
	)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	body := response.Body.String()
	for _, expected := range []string{
		`"freshness":"Live"`, `"hostName":"lab-01"`, `"running":2`, `"stopped":1`, `"unhealthy":1`,
		`"cpuPercent":42.5`, `"memoryPercent":61`, `"action":"Container restarted"`,
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("response missing %s: %s", expected, body)
		}
	}
	if err := database.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestOverviewReturnsOfflineAndNullDockerTotalsWhenSourcesUnavailable(t *testing.T) {
	database, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	now := time.Date(2026, time.September, 5, 12, 0, 0, 0, time.UTC)
	database.ExpectQuery(regexp.QuoteMeta(historyQuery)).WithArgs(now.Add(-time.Hour)).WillReturnRows(
		pgxmock.NewRows([]string{"Timestamp", "CpuPercent", "MemoryPercent"}))
	database.ExpectQuery(`SELECT "Id", "Kind", "Action"`).WithArgs(6, int64(0)).WillReturnRows(
		pgxmock.NewRows([]string{"Id", "Kind", "Action", "Target", "Timestamp", "Result", "Actor"}))
	handler := newHandler(fakeDocker{err: errors.New("socket unavailable")}, fakeMetrics{}, database,
		activity.NewReader(database), testLogger(), func() time.Time { return now })
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	body := response.Body.String()
	if response.Code != http.StatusOK || !strings.Contains(body, `"freshness":"Offline"`) ||
		!strings.Contains(body, `"containers":{"running":null,"stopped":null,"unhealthy":null}`) ||
		!strings.Contains(body, `"history":[]`) || !strings.Contains(body, `"recentActivity":[]`) {
		t.Fatalf("response = %d %s", response.Code, body)
	}
}

func TestOverviewSanitizesDatabaseFailure(t *testing.T) {
	database, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	database.ExpectQuery(regexp.QuoteMeta(historyQuery)).WithArgs(pgxmock.AnyArg()).WillReturnError(errors.New("password=secret host=db.internal"))
	handler := newHandler(fakeDocker{}, fakeMetrics{}, database, activity.NewReader(database), testLogger(), time.Now)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if response.Code != http.StatusInternalServerError || strings.Contains(response.Body.String(), "secret") || strings.Contains(response.Body.String(), "db.internal") {
		t.Fatalf("response = %d %s", response.Code, response.Body.String())
	}
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
