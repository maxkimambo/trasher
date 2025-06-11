//go:build windows

package validation

import (
	"fmt"
	"path/filepath"
	"syscall"
	"unsafe"
)

// getDiskSpaceInfo returns available and total disk space for the given path on Windows.
func getDiskSpaceInfo(path string) (available, total int64, err error) {
	// Get the directory path
	absDir, err := filepath.Abs(path)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get absolute path: %v", err)
	}

	// Get the volume root (e.g., "C:\")
	volume := filepath.VolumeName(absDir)
	if volume == "" {
		volume = absDir // fallback for UNC paths or current directory
	}

	// Add trailing backslash if needed
	if volume[len(volume)-1] != '\\' {
		volume += "\\"
	}

	// Convert to UTF-16 for Windows API
	volumePtr, err := syscall.UTF16PtrFromString(volume)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to convert volume path: %v", err)
	}

	// Call GetDiskFreeSpaceEx
	var freeBytesAvailable, totalBytes, totalFreeBytes uint64
	
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	getDiskFreeSpaceEx := kernel32.NewProc("GetDiskFreeSpaceExW")
	
	ret, _, err := getDiskFreeSpaceEx.Call(
		uintptr(unsafe.Pointer(volumePtr)),
		uintptr(unsafe.Pointer(&freeBytesAvailable)),
		uintptr(unsafe.Pointer(&totalBytes)),
		uintptr(unsafe.Pointer(&totalFreeBytes)),
	)
	
	if ret == 0 {
		return 0, 0, fmt.Errorf("failed to check disk space: %v", err)
	}

	return int64(freeBytesAvailable), int64(totalBytes), nil
}