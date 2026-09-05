package telemetry

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

const (
	collectionInterval  = 3 * time.Second
	persistenceInterval = 15 * time.Second
	pruneInterval       = time.Hour
	historyRetention    = 24 * time.Hour
)

type sampleCollector interface {
	Collect(context.Context) Metrics
}

type sampleDatabase interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

type Sampler struct {
	collector     sampleCollector
	database      sampleDatabase
	logger        *slog.Logger
	now           func() time.Time
	lastPersisted time.Time
	lastPruned    time.Time
}

func NewSampler(collector sampleCollector, database sampleDatabase, logger *slog.Logger) *Sampler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Sampler{collector: collector, database: database, logger: logger, now: time.Now}
}

// Run samples at a restrained cadence until the application context is
// canceled. Errors are retried and warning logs are rate-limited.
func (sampler *Sampler) Run(ctx context.Context) {
	ticker := time.NewTicker(collectionInterval)
	defer ticker.Stop()
	var lastWarning time.Time
	for {
		if err := sampler.sampleOnce(ctx); err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
			now := sampler.now().UTC()
			if lastWarning.IsZero() || now.Sub(lastWarning) >= time.Minute {
				sampler.logger.Warn("Host metric persistence failed; sampling will retry", "error", err)
				lastWarning = now
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (sampler *Sampler) sampleOnce(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	now := sampler.now().UTC()
	metrics := sampler.collector.Collect(ctx)
	if sampler.lastPersisted.IsZero() || now.Sub(sampler.lastPersisted) >= persistenceInterval {
		cpu := boundedPercentage(metrics.CPUPercent)
		memory := memoryPercentage(metrics.MemoryUsedBytes, metrics.MemoryTotalBytes)
		if cpu != nil || memory != nil {
			if _, err := sampler.database.Exec(ctx,
				`INSERT INTO "MetricSamples" ("Timestamp","CpuPercent","MemoryPercent") VALUES ($1,$2,$3)`,
				metrics.ObservedAt.UTC(), nullablePercentage(cpu), nullablePercentage(memory)); err != nil {
				return err
			}
			sampler.lastPersisted = now
		}
	}
	if sampler.lastPruned.IsZero() || now.Sub(sampler.lastPruned) >= pruneInterval {
		if _, err := sampler.database.Exec(ctx, `DELETE FROM "MetricSamples" WHERE "Timestamp" < $1`, now.Add(-historyRetention)); err != nil {
			return err
		}
		sampler.lastPruned = now
	}
	return nil
}

func nullablePercentage(value *float64) any {
	if value == nil {
		return nil
	}
	return *value
}

func memoryPercentage(used, total *int64) *float64 {
	if used == nil || total == nil || *used < 0 || *total <= 0 {
		return nil
	}
	value := float64(*used) * 100 / float64(*total)
	return boundedPercentage(&value)
}

func boundedPercentage(value *float64) *float64 {
	if value == nil || math.IsNaN(*value) || math.IsInf(*value, 0) {
		return nil
	}
	bounded := math.Max(0, math.Min(100, *value))
	return &bounded
}
