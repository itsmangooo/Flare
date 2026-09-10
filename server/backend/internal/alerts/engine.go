package alerts

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const defaultCooldown = 15 * time.Minute

type Signal struct {
	Fingerprint  string
	Kind         string
	Severity     string
	Title        string
	Message      string
	Source       string
	ResourceType string
	ResourceID   string
	OccurredAt   time.Time
	Recovery     bool
}

type notificationSink interface {
	Notify(context.Context, Notification) error
}

type DeliveryPolicy interface {
	Allow(context.Context, Signal) (bool, error)
}

type transactionDatabase interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

type Engine struct {
	database transactionDatabase
	sink     notificationSink
	logger   *slog.Logger
	policy   DeliveryPolicy
	now      func() time.Time
	cooldown time.Duration
}

func NewEngine(database transactionDatabase, sink notificationSink, logger *slog.Logger, policies ...DeliveryPolicy) *Engine {
	if logger == nil {
		logger = slog.Default()
	}
	engine := &Engine{database: database, sink: sink, logger: logger, now: time.Now, cooldown: defaultCooldown}
	if len(policies) > 0 {
		engine.policy = policies[0]
	}
	return engine
}

func (engine *Engine) Record(ctx context.Context, signal Signal) error {
	if engine == nil || engine.database == nil {
		return errors.New("alert engine is not configured")
	}
	if err := validateSignal(&signal); err != nil {
		return err
	}
	deliveryAllowed := true
	if engine.policy != nil {
		var err error
		deliveryAllowed, err = engine.policy.Allow(ctx, signal)
		if err != nil {
			return fmt.Errorf("read notification preferences: %w", err)
		}
	}
	observedAt := signal.OccurredAt.UTC()
	if observedAt.IsZero() {
		observedAt = engine.now().UTC()
	}
	notificationAt := engine.now().UTC()
	var notificationValue any
	if deliveryAllowed {
		notificationValue = notificationAt
	}
	tx, err := engine.database.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin alert transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var alertID uuid.UUID
	shouldNotify := false
	if signal.Recovery {
		err = tx.QueryRow(ctx, `UPDATE "Alerts" SET
    "Severity"=$2,"Title"=$3,"Message"=$4,"LastSeenAt"=GREATEST("LastSeenAt",$5),
    "RecoveredAt"=$5,"Status"='recovered',"OccurrenceCount"="OccurrenceCount"+1,
    "LastNotifiedAt"=CASE WHEN $7 THEN $6 ELSE "LastNotifiedAt" END
WHERE "Fingerprint"=$1 AND "Status"='active' AND "LastSeenAt" <= $5
RETURNING "Id",$7`, signal.Fingerprint, signal.Severity, signal.Title, signal.Message, observedAt, notificationAt, deliveryAllowed).Scan(&alertID, &shouldNotify)
	} else {
		err = tx.QueryRow(ctx, `INSERT INTO "Alerts"
    ("Id","Fingerprint","Kind","Severity","Title","Message","Source","ResourceType","ResourceId",
     "Status","FirstSeenAt","LastSeenAt","RecoveredAt","OccurrenceCount","LastNotifiedAt")
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,'active',$10,$10,NULL,1,$11)
ON CONFLICT ("Fingerprint") WHERE "Status"='active' DO UPDATE SET
    "Kind"=EXCLUDED."Kind","Severity"=EXCLUDED."Severity","Title"=EXCLUDED."Title",
    "Message"=EXCLUDED."Message","Source"=EXCLUDED."Source","ResourceType"=EXCLUDED."ResourceType",
    "ResourceId"=EXCLUDED."ResourceId","LastSeenAt"=GREATEST("Alerts"."LastSeenAt",EXCLUDED."LastSeenAt"),
    "OccurrenceCount"="Alerts"."OccurrenceCount"+1,
    "LastNotifiedAt"=CASE WHEN $13 AND ("Alerts"."LastNotifiedAt" IS NULL OR "Alerts"."LastNotifiedAt" <= $12)
        THEN $11 ELSE "Alerts"."LastNotifiedAt" END
RETURNING "Id",COALESCE("LastNotifiedAt"=$11,FALSE)`, uuid.New(), signal.Fingerprint, signal.Kind, signal.Severity,
			signal.Title, signal.Message, signal.Source, nullableSignal(signal.ResourceType), nullableSignal(signal.ResourceID),
			observedAt, notificationValue, notificationAt.Add(-engine.cooldown), deliveryAllowed).Scan(&alertID, &shouldNotify)
	}
	if errors.Is(err, pgx.ErrNoRows) && signal.Recovery {
		return tx.Rollback(ctx)
	}
	if err != nil {
		return fmt.Errorf("persist alert: %w", err)
	}
	if _, err = tx.Exec(ctx, `DELETE FROM "AlertReads" WHERE "AlertId"=$1`, alertID); err != nil {
		return fmt.Errorf("reset alert read state: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit alert: %w", err)
	}
	if shouldNotify && engine.sink != nil {
		if err = engine.sink.Notify(ctx, signalNotification(alertID, notificationAt, signal)); err != nil {
			engine.logger.Error("Alert notification could not be queued", "alert_id", alertID, "kind", signal.Kind)
			return err
		}
	}
	return nil
}

func validateSignal(signal *Signal) error {
	signal.Fingerprint = truncate(signal.Fingerprint, 256)
	signal.Kind = truncate(signal.Kind, 100)
	signal.Severity = strings.ToLower(strings.TrimSpace(signal.Severity))
	signal.Title = truncate(signal.Title, maximumTitleBytes)
	signal.Message = truncate(signal.Message, maximumMessageBytes)
	signal.Source = truncate(signal.Source, 100)
	signal.ResourceType = truncate(signal.ResourceType, 100)
	signal.ResourceID = truncate(signal.ResourceID, 256)
	if signal.Fingerprint == "" || signal.Kind == "" || signal.Title == "" || signal.Message == "" || signal.Source == "" {
		return errors.New("alert signal requires fingerprint, kind, title, message, and source")
	}
	if signal.Severity != "info" && signal.Severity != "warning" && signal.Severity != "critical" {
		return errors.New("alert severity must be info, warning, or critical")
	}
	return nil
}

func signalNotification(id uuid.UUID, at time.Time, signal Signal) Notification {
	priority := 3
	if signal.Severity == "critical" {
		priority = 5
	} else if signal.Severity == "warning" {
		priority = 4
	} else if signal.Recovery {
		priority = 2
	}
	tags := []string{signal.Severity, signal.Source}
	if signal.Recovery {
		tags[0] = "white_check_mark"
	}
	return Notification{
		ID: fmt.Sprintf("%s-%d", id, at.UnixNano()), Title: signal.Title,
		Message: signal.Message, Priority: priority, Tags: tags,
	}
}

func nullableSignal(value string) any {
	if value == "" {
		return nil
	}
	return value
}
