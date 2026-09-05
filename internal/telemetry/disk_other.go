//go:build !linux

package telemetry

import "errors"

func diskUsage(string) (used, total uint64, err error) {
	return 0, 0, errors.New("host filesystem telemetry is supported on Linux")
}
