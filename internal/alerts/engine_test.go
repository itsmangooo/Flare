package alerts

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
)

type recordingNotificationSink struct {
	notifications []Notification
}

type deliveryPolicyFunc func(context.Context, Signal) (bool, error)

func (policy deliveryPolicyFunc) Allow(ctx context.Context, signal Signal) (bool, error) {
	return policy(ctx, signal)
}

func (sink *recordingNotificationSink) Notify(_ context.Context, notification Notification) error {
	sink.notifications = append(sink.notifications, notification)
	return nil
}

func TestAlertEnginePersistsNotifiesAndResetsReadState(t *testing.T) {
	database, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	sink := &recordingNotificationSink{}
	engine := NewEngine(database, sink, alertTestLogger())
	now := time.Date(2026, time.September, 6, 12, 0, 0, 0, time.UTC)
	engine.now = func() time.Time { return now }
	engine.cooldown = 5 * time.Minute
	alertID := uuid.New()
	signal := Signal{
		Fingerprint: "container.health:abc", Kind: "container.unhealthy", Severity: "warning",
		Title: "Container became unhealthy", Message: "api — Health check failed.", Source: "docker",
		ResourceType: "container", ResourceID: "abc", OccurredAt: now.Add(-time.Second),
	}
	database.ExpectBegin()
	database.ExpectQuery(`INSERT INTO "Alerts"`).WithArgs(
		pgxmock.AnyArg(), signal.Fingerprint, signal.Kind, signal.Severity, signal.Title, signal.Message,
		signal.Source, signal.ResourceType, signal.ResourceID, signal.OccurredAt, now, now.Add(-5*time.Minute), true,
	).WillReturnRows(pgxmock.NewRows([]string{"Id", "shouldNotify"}).AddRow(alertID, true))
	database.ExpectExec(regexp.QuoteMeta(`DELETE FROM "AlertReads" WHERE "AlertId"=$1`)).WithArgs(alertID).
		WillReturnResult(pgxmock.NewResult("DELETE", 2))
	database.ExpectCommit()

	if err = engine.Record(context.Background(), signal); err != nil {
		t.Fatal(err)
	}
	if len(sink.notifications) != 1 || sink.notifications[0].Priority != 4 || sink.notifications[0].Title != signal.Title {
		t.Fatalf("notifications = %#v", sink.notifications)
	}
	if err = database.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAlertEngineDeduplicatesDeliveryInsideCooldown(t *testing.T) {
	database, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	sink := &recordingNotificationSink{}
	engine := NewEngine(database, sink, alertTestLogger())
	now := time.Date(2026, time.September, 6, 12, 0, 0, 0, time.UTC)
	engine.now = func() time.Time { return now }
	alertID := uuid.New()
	signal := Signal{Fingerprint: "docker.availability", Kind: "docker.unavailable", Severity: "critical", Title: "Docker unavailable", Message: "Docker host", Source: "docker", OccurredAt: now}
	database.ExpectBegin()
	database.ExpectQuery(`INSERT INTO "Alerts"`).WithArgs(
		pgxmock.AnyArg(), signal.Fingerprint, signal.Kind, signal.Severity, signal.Title, signal.Message,
		signal.Source, nil, nil, now, now, now.Add(-defaultCooldown), true,
	).WillReturnRows(pgxmock.NewRows([]string{"Id", "shouldNotify"}).AddRow(alertID, false))
	database.ExpectExec(`DELETE FROM "AlertReads"`).WithArgs(alertID).WillReturnResult(pgxmock.NewResult("DELETE", 0))
	database.ExpectCommit()

	if err = engine.Record(context.Background(), signal); err != nil {
		t.Fatal(err)
	}
	if len(sink.notifications) != 0 {
		t.Fatalf("duplicate notification escaped cooldown: %#v", sink.notifications)
	}
	if err = database.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAlertEnginePersistsHistoryWhenPreferencesSuppressDelivery(t *testing.T) {
	database, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	sink := &recordingNotificationSink{}
	policy := deliveryPolicyFunc(func(context.Context, Signal) (bool, error) { return false, nil })
	engine := NewEngine(database, sink, alertTestLogger(), policy)
	now := time.Date(2026, time.September, 6, 12, 0, 0, 0, time.UTC)
	engine.now = func() time.Time { return now }
	alertID := uuid.New()
	signal := Signal{Fingerprint: "host.memory", Kind: "host.threshold", Severity: "warning", Title: "High memory", Message: "Memory exceeded threshold.", Source: "host", OccurredAt: now}
	database.ExpectBegin()
	database.ExpectQuery(`INSERT INTO "Alerts"`).WithArgs(
		pgxmock.AnyArg(), signal.Fingerprint, signal.Kind, signal.Severity, signal.Title, signal.Message,
		signal.Source, nil, nil, now, nil, now.Add(-defaultCooldown), false,
	).WillReturnRows(pgxmock.NewRows([]string{"Id", "shouldNotify"}).AddRow(alertID, false))
	database.ExpectExec(`DELETE FROM "AlertReads"`).WithArgs(alertID).WillReturnResult(pgxmock.NewResult("DELETE", 0))
	database.ExpectCommit()

	if err = engine.Record(context.Background(), signal); err != nil {
		t.Fatal(err)
	}
	if len(sink.notifications) != 0 {
		t.Fatalf("suppressed notification was published: %#v", sink.notifications)
	}
	if err = database.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAlertEngineRecoversOnlyAnActiveFingerprint(t *testing.T) {
	database, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	sink := &recordingNotificationSink{}
	engine := NewEngine(database, sink, alertTestLogger())
	now := time.Date(2026, time.September, 6, 12, 0, 0, 0, time.UTC)
	engine.now = func() time.Time { return now }
	signal := Signal{Fingerprint: "container.health:abc", Kind: "container.unhealthy", Severity: "info", Title: "Container recovered", Message: "api", Source: "docker", Recovery: true, OccurredAt: now}
	database.ExpectBegin()
	database.ExpectQuery(`UPDATE "Alerts" SET`).WithArgs(signal.Fingerprint, signal.Severity, signal.Title, signal.Message, now, now, true).
		WillReturnError(pgx.ErrNoRows)
	database.ExpectRollback()

	if err = engine.Record(context.Background(), signal); err != nil {
		t.Fatal(err)
	}
	if len(sink.notifications) != 0 {
		t.Fatalf("orphan recovery was published: %#v", sink.notifications)
	}
	if err = database.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAlertEnginePersistsAndPublishesRecovery(t *testing.T) {
	database, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	sink := &recordingNotificationSink{}
	engine := NewEngine(database, sink, alertTestLogger())
	now := time.Date(2026, time.September, 6, 12, 0, 0, 0, time.UTC)
	engine.now = func() time.Time { return now }
	alertID := uuid.New()
	signal := Signal{Fingerprint: "docker.availability", Kind: "docker.unavailable", Severity: "info", Title: "Docker reconnected", Message: "Docker host", Source: "docker", Recovery: true, OccurredAt: now}
	database.ExpectBegin()
	database.ExpectQuery(`UPDATE "Alerts" SET`).WithArgs(signal.Fingerprint, signal.Severity, signal.Title, signal.Message, now, now, true).
		WillReturnRows(pgxmock.NewRows([]string{"Id", "shouldNotify"}).AddRow(alertID, true))
	database.ExpectExec(`DELETE FROM "AlertReads"`).WithArgs(alertID).WillReturnResult(pgxmock.NewResult("DELETE", 1))
	database.ExpectCommit()

	if err = engine.Record(context.Background(), signal); err != nil {
		t.Fatal(err)
	}
	if len(sink.notifications) != 1 || sink.notifications[0].Priority != 2 || sink.notifications[0].Title != signal.Title {
		t.Fatalf("recovery notifications = %#v", sink.notifications)
	}
	if err = database.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAlertEngineRejectsInvalidSignalsBeforeDatabaseAccess(t *testing.T) {
	database, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	engine := NewEngine(database, nil, alertTestLogger())
	if err = engine.Record(context.Background(), Signal{Fingerprint: "bad", Severity: "loud"}); err == nil {
		t.Fatal("invalid signal was accepted")
	}
}
