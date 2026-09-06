package monitoring

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/itsmangooo/flare/internal/alerts"
	"github.com/itsmangooo/flare/internal/telemetry"
)

type thresholdSink struct {
	signals []alerts.Signal
	err     error
}

func (sink *thresholdSink) Record(_ context.Context, signal alerts.Signal) error {
	if sink.err != nil {
		err := sink.err
		sink.err = nil
		return err
	}
	sink.signals = append(sink.signals, signal)
	return nil
}

func TestHostThresholdRequiresSustainedBreachAndRecovery(t *testing.T) {
	sink := &thresholdSink{}
	monitor := NewHostThresholdMonitor(sink, 90, 90, 3)
	now := time.Date(2026, time.September, 6, 14, 0, 0, 0, time.UTC)
	high, hysteresis, recovered := 92.0, 87.0, 84.0
	for range 3 {
		if err := monitor.Observe(context.Background(), telemetry.Metrics{HostName: "lab", ObservedAt: now, CPUPercent: &high}); err != nil {
			t.Fatal(err)
		}
	}
	if len(sink.signals) != 1 || sink.signals[0].Fingerprint != "host.cpu:lab" || sink.signals[0].Severity != "warning" || sink.signals[0].Recovery {
		t.Fatalf("breach signals = %#v", sink.signals)
	}
	_ = monitor.Observe(context.Background(), telemetry.Metrics{HostName: "lab", ObservedAt: now, CPUPercent: &high})
	_ = monitor.Observe(context.Background(), telemetry.Metrics{HostName: "lab", ObservedAt: now, CPUPercent: &hysteresis})
	for range 3 {
		if err := monitor.Observe(context.Background(), telemetry.Metrics{HostName: "lab", ObservedAt: now, CPUPercent: &recovered}); err != nil {
			t.Fatal(err)
		}
	}
	if len(sink.signals) != 2 || !sink.signals[1].Recovery || sink.signals[1].Severity != "info" {
		t.Fatalf("recovery signals = %#v", sink.signals)
	}
}

func TestHostThresholdCalculatesMemoryAndIgnoresUnavailableMetrics(t *testing.T) {
	sink := &thresholdSink{}
	monitor := NewHostThresholdMonitor(sink, 90, 80, 2)
	used, total := int64(9), int64(10)
	if err := monitor.Observe(context.Background(), telemetry.Metrics{HostName: "lab"}); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := monitor.Observe(context.Background(), telemetry.Metrics{HostName: "lab", MemoryUsedBytes: &used, MemoryTotalBytes: &total}); err != nil {
			t.Fatal(err)
		}
	}
	if len(sink.signals) != 1 || sink.signals[0].Fingerprint != "host.memory:lab" || sink.signals[0].ResourceType != "host" {
		t.Fatalf("signals = %#v", sink.signals)
	}
}

func TestHostThresholdRetriesFailedSignal(t *testing.T) {
	sink := &thresholdSink{err: errors.New("database unavailable")}
	monitor := NewHostThresholdMonitor(sink, 90, 90, 2)
	high := 95.0
	_ = monitor.Observe(context.Background(), telemetry.Metrics{CPUPercent: &high})
	if err := monitor.Observe(context.Background(), telemetry.Metrics{CPUPercent: &high}); err == nil {
		t.Fatal("alert persistence failure was not returned")
	}
	if err := monitor.Observe(context.Background(), telemetry.Metrics{CPUPercent: &high}); err != nil {
		t.Fatal(err)
	}
	if len(sink.signals) != 1 {
		t.Fatalf("signals = %#v", sink.signals)
	}
}
