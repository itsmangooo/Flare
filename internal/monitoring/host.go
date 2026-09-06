package monitoring

import (
	"context"
	"fmt"
	"math"

	"github.com/itsmangooo/flare/internal/alerts"
	"github.com/itsmangooo/flare/internal/telemetry"
)

const hostRecoveryHysteresis = 5.0

type thresholdState struct {
	breaches   int
	recoveries int
	active     bool
}

type HostThresholdMonitor struct {
	sink            alertSink
	cpuThreshold    float64
	memoryThreshold float64
	sustained       int
	cpu             thresholdState
	memory          thresholdState
}

func NewHostThresholdMonitor(sink alertSink, cpuThreshold, memoryThreshold float64, sustainedSamples int) *HostThresholdMonitor {
	return &HostThresholdMonitor{
		sink: sink, cpuThreshold: cpuThreshold, memoryThreshold: memoryThreshold,
		sustained: max(sustainedSamples, 2),
	}
}

func (monitor *HostThresholdMonitor) Observe(ctx context.Context, metrics telemetry.Metrics) error {
	if monitor == nil || monitor.sink == nil {
		return nil
	}
	if err := monitor.evaluate(ctx, metrics, "cpu", metrics.CPUPercent, monitor.cpuThreshold, &monitor.cpu); err != nil {
		return err
	}
	memory := memoryPercentage(metrics.MemoryUsedBytes, metrics.MemoryTotalBytes)
	return monitor.evaluate(ctx, metrics, "memory", memory, monitor.memoryThreshold, &monitor.memory)
}

func (monitor *HostThresholdMonitor) evaluate(ctx context.Context, metrics telemetry.Metrics, resource string, value *float64, threshold float64, state *thresholdState) error {
	if value == nil || math.IsNaN(*value) || math.IsInf(*value, 0) {
		if state.active {
			state.recoveries = 0
		} else {
			state.breaches = 0
		}
		return nil
	}
	if *value >= threshold {
		state.recoveries = 0
		state.breaches++
		if state.active || state.breaches < monitor.sustained {
			return nil
		}
		signal := hostSignal(metrics, resource, *value, threshold, false)
		if err := monitor.sink.Record(ctx, signal); err != nil {
			return err
		}
		state.active = true
		return nil
	}
	state.breaches = 0
	if !state.active || *value > math.Max(0, threshold-hostRecoveryHysteresis) {
		state.recoveries = 0
		return nil
	}
	state.recoveries++
	if state.recoveries < monitor.sustained {
		return nil
	}
	signal := hostSignal(metrics, resource, *value, threshold, true)
	if err := monitor.sink.Record(ctx, signal); err != nil {
		return err
	}
	*state = thresholdState{}
	return nil
}

func hostSignal(metrics telemetry.Metrics, resource string, value, threshold float64, recovery bool) alerts.Signal {
	title := "Host resource threshold exceeded"
	severity := "warning"
	message := fmt.Sprintf("%s is %.1f%% (threshold %.1f%%).", resource, value, threshold)
	if recovery {
		title = "Host resource recovered"
		severity = "info"
		message = fmt.Sprintf("%s returned to %.1f%%.", resource, value)
	}
	hostName := metrics.HostName
	if hostName == "" {
		hostName = "homelab"
	}
	return alerts.Signal{
		Fingerprint: "host." + resource + ":" + hostName, Kind: "host.resource_threshold",
		Severity: severity, Title: title, Message: message, Source: "host",
		ResourceType: "host", ResourceID: hostName, OccurredAt: metrics.ObservedAt, Recovery: recovery,
	}
}

func memoryPercentage(used, total *int64) *float64 {
	if used == nil || total == nil || *used < 0 || *total <= 0 {
		return nil
	}
	value := math.Max(0, math.Min(100, float64(*used)*100/float64(*total)))
	return &value
}
