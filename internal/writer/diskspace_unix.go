//go:build unix || linux || darwin

package writer

import (
	"fmt"
	"syscall"
)

// checkDiskSpace verifies that there's enough disk space available on Unix systems.
func checkDiskSpace(dir string, requiredBytes int64) error {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(dir, &stat); err != nil {
		return fmt.Errorf("failed to check disk space: %v", err)
	}

	// Calculate available bytes
	available := int64(stat.Bavail) * int64(stat.Bsize)
	if requiredBytes > available {
		return fmt.Errorf("insufficient disk space: need %d bytes, have %d bytes", 
			requiredBytes, available)
	}

	return nil
}