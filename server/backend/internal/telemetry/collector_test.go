package telemetry

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCollectorReadsHostMountAndCalculatesRates(t *testing.T) {
	procPath := t.TempDir()
	writeProcFixture(t, procPath,
		"cpu  100 0 50 850 0 0 0 0 0 0\n",
		"eth0: 1000 1 0 0 0 0 0 0 2000 1 0 0 0 0 0 0\n")
	now := time.Date(2026, time.September, 5, 10, 0, 0, 0, time.FixedZone("test", 2*60*60))
	collector := newCollector("lab-01", procPath, "/host/rootfs", discardLogger(), func() time.Time { return now },
		func(path string) (uint64, uint64, error) {
			if path != "/host/rootfs" {
				t.Fatalf("disk path = %q", path)
			}
			return 226 << 30, 512 << 30, nil
		})

	first := collector.Collect(context.Background())
	if first.HostName != "lab-01" || !first.ObservedAt.Equal(now.UTC()) {
		t.Fatalf("identity = %#v", first)
	}
	assertFloat(t, first.LoadAverage, 2.14)
	assertInt(t, first.MemoryTotalBytes, 16_777_216)
	assertInt(t, first.MemoryUsedBytes, 12_582_912)
	assertInt(t, first.DiskUsedBytes, 226<<30)
	assertInt(t, first.DiskTotalBytes, 512<<30)
	if first.CPUPercent != nil || first.NetworkReceiveBytesPerSecond != nil || first.NetworkTransmitBytesPerSecond != nil {
		t.Fatalf("first sample invented a rate: %#v", first)
	}
	if first.Uptime == nil || *first.Uptime != "01:00:00" {
		t.Fatalf("uptime = %v", first.Uptime)
	}

	now = now.Add(2 * time.Second)
	writeFile(t, filepath.Join(procPath, "stat"), "cpu  200 0 100 900 0 0 0 0 0 0\n")
	writeFile(t, filepath.Join(procPath, "net", "dev"), networkHeader+"eth0: 1600 1 0 0 0 0 0 0 2300 1 0 0 0 0 0 0\n")
	second := collector.Collect(context.Background())
	assertFloat(t, second.CPUPercent, 75)
	assertFloat(t, second.NetworkReceiveBytesPerSecond, 300)
	assertFloat(t, second.NetworkTransmitBytesPerSecond, 150)
}

func TestCollectorReturnsUnavailableInsteadOfContainerMetrics(t *testing.T) {
	collector := newCollector("lab-02", t.TempDir(), filepath.Join(t.TempDir(), "missing"), discardLogger(), time.Now,
		func(string) (uint64, uint64, error) { return 0, 0, errors.New("not mounted") })
	metrics := collector.Collect(context.Background())
	if metrics.CPUPercent != nil || metrics.LoadAverage != nil || metrics.MemoryUsedBytes != nil ||
		metrics.MemoryTotalBytes != nil || metrics.DiskUsedBytes != nil || metrics.DiskTotalBytes != nil ||
		metrics.NetworkReceiveBytesPerSecond != nil || metrics.NetworkTransmitBytesPerSecond != nil || metrics.Uptime != nil {
		t.Fatalf("unavailable host data must remain null: %#v", metrics)
	}
}

func TestParsersRejectMalformedAndOverflowingInput(t *testing.T) {
	if _, ok := parseCPU("cpu  1 2 nope 4 5\n"); ok {
		t.Fatal("malformed CPU input was accepted")
	}
	if value := parseLoadAverage("NaN 0 0"); value != nil {
		t.Fatalf("NaN load = %v", *value)
	}
	used, total := parseMemory("MemTotal: 9223372036854775807 kB\nMemAvailable: 1 kB\n")
	if used != nil || total != nil {
		t.Fatalf("overflowing memory input = %v %v", used, total)
	}
	if value := parseUptime("-1 0"); value != nil {
		t.Fatalf("negative uptime = %v", *value)
	}
	if _, ok := parseNetwork("eth0: invalid 0 0 0 0 0 0 0 1\n", time.Now()); ok {
		t.Fatal("malformed network input was accepted")
	}
}

const networkHeader = "Inter-| Receive | Transmit\n face |bytes packets errs drop fifo frame compressed multicast|bytes packets errs drop fifo colls carrier compressed\n"

func writeProcFixture(t *testing.T, procPath, stat, network string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(procPath, "net"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(procPath, "stat"), stat)
	writeFile(t, filepath.Join(procPath, "loadavg"), "2.14 1.50 1.00 1/100 1\n")
	writeFile(t, filepath.Join(procPath, "meminfo"), "MemTotal: 16384 kB\nMemAvailable: 4096 kB\n")
	writeFile(t, filepath.Join(procPath, "uptime"), "3600.00 0.00\n")
	writeFile(t, filepath.Join(procPath, "net", "dev"), networkHeader+network)
}

func writeFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func assertFloat(t *testing.T, value *float64, expected float64) {
	t.Helper()
	if value == nil || math.Abs(*value-expected) > 0.0001 {
		t.Fatalf("value = %v, want %v", value, expected)
	}
}

func assertInt(t *testing.T, value *int64, expected int64) {
	t.Helper()
	if value == nil || *value != expected {
		t.Fatalf("value = %v, want %v", value, expected)
	}
}
