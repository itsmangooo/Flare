//go:build linux

package telemetry

import (
	"errors"
	"math"

	"golang.org/x/sys/unix"
)

func diskUsage(path string) (used, total uint64, err error) {
	var stat unix.Statfs_t
	if err := unix.Statfs(path, &stat); err != nil {
		return 0, 0, err
	}
	if stat.Bsize <= 0 {
		return 0, 0, errors.New("host filesystem reported an invalid block size")
	}
	blockSize := uint64(stat.Bsize)
	if stat.Blocks > math.MaxUint64/blockSize || stat.Bavail > math.MaxUint64/blockSize {
		return 0, 0, errors.New("host filesystem size exceeds supported range")
	}
	total = stat.Blocks * blockSize
	available := stat.Bavail * blockSize
	if available > total {
		return 0, total, nil
	}
	return total - available, total, nil
}
