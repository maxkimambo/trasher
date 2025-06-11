//go:build unix || linux || darwin

package validation

import (
	"fmt"
	"path/filepath"
	"syscall"
)

// getDiskSpaceInfo returns available and total disk space for the given path on Unix systems.
func getDiskSpaceInfo(path string) (available, total int64, err error) {
	dir := filepath.Dir(path)
	
	var stat syscall.Statfs_t
	if err := syscall.Statfs(dir, &stat); err != nil {
		return 0, 0, fmt.Errorf("failed to check disk space: %v", err)
	}

	available = int64(stat.Bavail) * int64(stat.Bsize)
	total = int64(stat.Blocks) * int64(stat.Bsize)
	return available, total, nil
}