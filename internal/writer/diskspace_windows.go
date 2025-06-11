//go:build windows

package writer

import (
	"fmt"
	"path/filepath"
	"syscall"
	"unsafe"
)

// checkDiskSpace verifies that there's enough disk space available on Windows.
func checkDiskSpace(dir string, requiredBytes int64) error {
	// Get the directory path
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %v", err)
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
		return fmt.Errorf("failed to convert volume path: %v", err)
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
		return fmt.Errorf("failed to check disk space: %v", err)
	}

	// Check if we have enough space
	available := int64(freeBytesAvailable)
	if requiredBytes > available {
		return fmt.Errorf("insufficient disk space: need %d bytes, have %d bytes", 
			requiredBytes, available)
	}

	return nil
}