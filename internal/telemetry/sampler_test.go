package telemetry

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"math"
	"regexp"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
)

type fixedCollector struct {
	metrics Metrics
	calls   int
}

func (collector *fixedCollector) Collect(context.Context) Metrics {
	collector.calls++
	return collector.metrics
}

func TestSamplerPersistsRealMetricsAtBoundedCadenceAndPrunes(t *testing.T) {
	database, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	now := time.Date(2026, time.September, 5, 13, 0, 0, 0, time.UTC)
	cpu, used, total := 42.5, int64(10_800), int64(16_000)
	collector := &fixedCollector{metrics: Metrics{ObservedAt: now, CPUPercent: &cpu, MemoryUsedBytes: &used, MemoryTotalBytes: &total}}
	sampler := NewSampler(collector, database, discardSamplerLogger())
	sampler.now = func() time.Time { return now }
	database.ExpectExec(regexp.QuoteMeta(`INSERT INTO "MetricSamples" ("Timestamp","CpuPercent","MemoryPercent") VALUES ($1,$2,$3)`)).
		WithArgs(now, 42.5, 67.5).WillReturnResult(pgxmock.NewResult("INSERT", 1))
	database.ExpectExec(regexp.QuoteMeta(`DELETE FROM "MetricSamples" WHERE "Timestamp" < $1`)).
		WithArgs(now.Add(-24 * time.Hour)).WillReturnResult(pgxmock.NewResult("DELETE", 2))
	if err := sampler.sampleOnce(context.Background()); err != nil {
		t.Fatal(err)
	}

	now = now.Add(3 * time.Second)
	collector.metrics.ObservedAt = now
	if err := sampler.sampleOnce(context.Background()); err != nil {
		t.Fatal(err)
	}

	now = now.Add(12 * time.Second)
	collector.metrics.ObservedAt = now
	database.ExpectExec(regexp.QuoteMeta(`INSERT INTO "MetricSamples" ("Timestamp","CpuPercent","MemoryPercent") VALUES ($1,$2,$3)`)).
		WithArgs(now, 42.5, 67.5).WillReturnResult(pgxmock.NewResult("INSERT", 1))
	if err := sampler.sampleOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if collector.calls != 3 {
		t.Fatalf("collector calls = %d", collector.calls)
	}
	if err := database.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSamplerDoesNotPersistUnavailableMetrics(t *testing.T) {
	database, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	now := time.Date(2026, time.September, 5, 13, 0, 0, 0, time.UTC)
	collector := &fixedCollector{metrics: Metrics{ObservedAt: now}}
	sampler := NewSampler(collector, database, discardSamplerLogger())
	sampler.now = func() time.Time { return now }
	database.ExpectExec(regexp.QuoteMeta(`DELETE FROM "MetricSamples" WHERE "Timestamp" < $1`)).
		WithArgs(now.Add(-24 * time.Hour)).WillReturnResult(pgxmock.NewResult("DELETE", 0))
	if err := sampler.sampleOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := database.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSamplerRetriesFailedPersistenceWithoutAdvancingCadence(t *testing.T) {
	database, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	now := time.Date(2026, time.September, 5, 13, 0, 0, 0, time.UTC)
	cpu := 10.0
	collector := &fixedCollector{metrics: Metrics{ObservedAt: now, CPUPercent: &cpu}}
	sampler := NewSampler(collector, database, discardSamplerLogger())
	sampler.now = func() time.Time { return now }
	database.ExpectExec(regexp.QuoteMeta(`INSERT INTO "MetricSamples" ("Timestamp","CpuPercent","MemoryPercent") VALUES ($1,$2,$3)`)).
		WithArgs(now, 10.0, nil).WillReturnError(errors.New("database offline"))
	if err := sampler.sampleOnce(context.Background()); err == nil {
		t.Fatal("sampleOnce should return the database failure")
	}
	database.ExpectExec(regexp.QuoteMeta(`INSERT INTO "MetricSamples" ("Timestamp","CpuPercent","MemoryPercent") VALUES ($1,$2,$3)`)).
		WithArgs(now, 10.0, nil).WillReturnResult(pgxmock.NewResult("INSERT", 1))
	database.ExpectExec(regexp.QuoteMeta(`DELETE FROM "MetricSamples" WHERE "Timestamp" < $1`)).
		WithArgs(now.Add(-24 * time.Hour)).WillReturnResult(pgxmock.NewResult("DELETE", 0))
	if err := sampler.sampleOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := database.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPercentageValidation(t *testing.T) {
	used, total := int64(5), int64(0)
	if memoryPercentage(&used, &total) != nil {
		t.Fatal("zero memory total must be unavailable")
	}
	notANumber := math.NaN()
	if boundedPercentage(&notANumber) != nil {
		t.Fatal("NaN percentage must be unavailable")
	}
}

func discardSamplerLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
