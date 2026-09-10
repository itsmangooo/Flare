package telemetry

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const maximumProcFileBytes = 1 << 20

// Metrics matches the existing Flutter/API host telemetry contract. Nullable
// values are intentional: Flare never substitutes container telemetry for
// unavailable host data.
type Metrics struct {
	HostName                      string    `json:"hostName"`
	ObservedAt                    time.Time `json:"observedAt"`
	CPUPercent                    *float64  `json:"cpuPercent"`
	LoadAverage                   *float64  `json:"loadAverage"`
	MemoryUsedBytes               *int64    `json:"memoryUsedBytes"`
	MemoryTotalBytes              *int64    `json:"memoryTotalBytes"`
	DiskUsedBytes                 *int64    `json:"diskUsedBytes"`
	DiskTotalBytes                *int64    `json:"diskTotalBytes"`
	NetworkReceiveBytesPerSecond  *float64  `json:"networkReceiveBytesPerSecond"`
	NetworkTransmitBytesPerSecond *float64  `json:"networkTransmitBytesPerSecond"`
	Uptime                        *string   `json:"uptime"`
}

type cpuSample struct {
	total uint64
	idle  uint64
}

type networkSample struct {
	received    uint64
	transmitted uint64
	timestamp   time.Time
}

type diskUsageFunc func(string) (used, total uint64, err error)

// Collector reads explicitly configured host mounts. It is safe for concurrent
// callers, though the normal runtime uses a single periodic collector.
type Collector struct {
	hostName    string
	procPath    string
	rootFSPath  string
	logger      *slog.Logger
	now         func() time.Time
	diskUsage   diskUsageFunc
	mu          sync.Mutex
	previousCPU *cpuSample
	previousNet *networkSample
}

func NewCollector(hostName, procPath, rootFSPath string, logger *slog.Logger) *Collector {
	return newCollector(hostName, procPath, rootFSPath, logger, time.Now, diskUsage)
}

func newCollector(hostName, procPath, rootFSPath string, logger *slog.Logger, now func() time.Time, disk diskUsageFunc) *Collector {
	if logger == nil {
		logger = slog.Default()
	}
	return &Collector{
		hostName:   hostName,
		procPath:   procPath,
		rootFSPath: rootFSPath,
		logger:     logger,
		now:        now,
		diskUsage:  disk,
	}
}

func (collector *Collector) Collect(ctx context.Context) Metrics {
	now := collector.now().UTC()
	metrics := Metrics{HostName: collector.hostName, ObservedAt: now}

	if contents, err := collector.readProc(ctx, "stat"); err == nil {
		if current, ok := parseCPU(contents); ok {
			metrics.CPUPercent = collector.cpuPercent(current)
		}
	} else {
		collector.unavailable("cpu", err)
	}
	if contents, err := collector.readProc(ctx, "loadavg"); err == nil {
		metrics.LoadAverage = parseLoadAverage(contents)
	} else {
		collector.unavailable("load", err)
	}
	if contents, err := collector.readProc(ctx, "meminfo"); err == nil {
		metrics.MemoryUsedBytes, metrics.MemoryTotalBytes = parseMemory(contents)
	} else {
		collector.unavailable("memory", err)
	}
	if contents, err := collector.readProc(ctx, "uptime"); err == nil {
		metrics.Uptime = parseUptime(contents)
	} else {
		collector.unavailable("uptime", err)
	}
	if contents, err := collector.readProc(ctx, "net", "dev"); err == nil {
		if current, ok := parseNetwork(contents, now); ok {
			metrics.NetworkReceiveBytesPerSecond, metrics.NetworkTransmitBytesPerSecond = collector.networkRates(current)
		}
	} else {
		collector.unavailable("network", err)
	}
	if err := ctx.Err(); err == nil {
		if used, total, diskErr := collector.diskUsage(collector.rootFSPath); diskErr == nil {
			metrics.DiskUsedBytes = uint64ToInt64(used)
			metrics.DiskTotalBytes = uint64ToInt64(total)
		} else {
			collector.unavailable("disk", diskErr)
		}
	}
	return metrics
}

func (collector *Collector) readProc(ctx context.Context, elements ...string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	path := filepath.Join(append([]string{collector.procPath}, elements...)...)
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	contents, err := io.ReadAll(io.LimitReader(file, maximumProcFileBytes+1))
	if err != nil {
		return "", err
	}
	if len(contents) > maximumProcFileBytes {
		return "", errors.New("host telemetry file exceeds safety limit")
	}
	return string(contents), nil
}

func (collector *Collector) cpuPercent(current cpuSample) *float64 {
	collector.mu.Lock()
	defer collector.mu.Unlock()
	previous := collector.previousCPU
	collector.previousCPU = &current
	if previous == nil || current.total <= previous.total || current.idle < previous.idle {
		return nil
	}
	totalDelta := current.total - previous.total
	idleDelta := current.idle - previous.idle
	if idleDelta > totalDelta {
		return nil
	}
	value := math.Max(0, math.Min(100, float64(totalDelta-idleDelta)*100/float64(totalDelta)))
	return &value
}

func (collector *Collector) networkRates(current networkSample) (*float64, *float64) {
	collector.mu.Lock()
	defer collector.mu.Unlock()
	previous := collector.previousNet
	collector.previousNet = &current
	if previous == nil || !current.timestamp.After(previous.timestamp) || current.received < previous.received || current.transmitted < previous.transmitted {
		return nil, nil
	}
	seconds := current.timestamp.Sub(previous.timestamp).Seconds()
	received := float64(current.received-previous.received) / seconds
	transmitted := float64(current.transmitted-previous.transmitted) / seconds
	return &received, &transmitted
}

func parseCPU(contents string) (cpuSample, bool) {
	line, _, _ := strings.Cut(contents, "\n")
	fields := strings.Fields(line)
	if len(fields) < 6 || fields[0] != "cpu" {
		return cpuSample{}, false
	}
	values := make([]uint64, 0, len(fields)-1)
	var total uint64
	for _, field := range fields[1:] {
		value, err := strconv.ParseUint(field, 10, 64)
		if err != nil || math.MaxUint64-total < value {
			return cpuSample{}, false
		}
		total += value
		values = append(values, value)
	}
	if math.MaxUint64-values[3] < values[4] {
		return cpuSample{}, false
	}
	return cpuSample{total: total, idle: values[3] + values[4]}, true
}

func parseLoadAverage(contents string) *float64 {
	fields := strings.Fields(contents)
	if len(fields) == 0 {
		return nil
	}
	value, err := strconv.ParseFloat(fields[0], 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
		return nil
	}
	return &value
}

func parseMemory(contents string) (used, total *int64) {
	var totalBytes, availableBytes *int64
	for _, line := range strings.Split(contents, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		value, err := strconv.ParseInt(fields[1], 10, 64)
		if err != nil || value < 0 || value > math.MaxInt64/1024 {
			continue
		}
		bytes := value * 1024
		switch strings.TrimSuffix(fields[0], ":") {
		case "MemTotal":
			totalBytes = &bytes
		case "MemAvailable":
			availableBytes = &bytes
		}
	}
	if totalBytes == nil {
		return nil, nil
	}
	if availableBytes == nil || *availableBytes > *totalBytes {
		return nil, totalBytes
	}
	usedBytes := *totalBytes - *availableBytes
	return &usedBytes, totalBytes
}

func parseUptime(contents string) *string {
	fields := strings.Fields(contents)
	if len(fields) == 0 {
		return nil
	}
	seconds, err := strconv.ParseFloat(fields[0], 64)
	if err != nil || math.IsNaN(seconds) || math.IsInf(seconds, 0) || seconds < 0 || seconds > float64(math.MaxInt64/int64(time.Second)) {
		return nil
	}
	value := formatDuration(time.Duration(seconds * float64(time.Second)))
	return &value
}

func parseNetwork(contents string, timestamp time.Time) (networkSample, bool) {
	var received, transmitted uint64
	found := false
	for _, line := range strings.Split(contents, "\n") {
		interfaceName, values, ok := strings.Cut(line, ":")
		if !ok || strings.TrimSpace(interfaceName) == "lo" {
			continue
		}
		fields := strings.Fields(values)
		if len(fields) < 9 {
			continue
		}
		rx, rxErr := strconv.ParseUint(fields[0], 10, 64)
		tx, txErr := strconv.ParseUint(fields[8], 10, 64)
		if rxErr != nil || txErr != nil || math.MaxUint64-received < rx || math.MaxUint64-transmitted < tx {
			continue
		}
		received += rx
		transmitted += tx
		found = true
	}
	return networkSample{received: received, transmitted: transmitted, timestamp: timestamp}, found
}

func formatDuration(duration time.Duration) string {
	totalSeconds := int64(duration / time.Second)
	days := totalSeconds / 86400
	hours := (totalSeconds % 86400) / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60
	if days > 0 {
		return fmt.Sprintf("%d.%02d:%02d:%02d", days, hours, minutes, seconds)
	}
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}

func uint64ToInt64(value uint64) *int64 {
	if value > math.MaxInt64 {
		return nil
	}
	result := int64(value)
	return &result
}

func (collector *Collector) unavailable(metric string, err error) {
	collector.logger.Debug("Host telemetry unavailable", "metric", metric, "error", err)
}
