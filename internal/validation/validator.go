package validation

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"syscall"

	"github.com/maxkimambo/trasher/pkg/generator"
	"github.com/maxkimambo/trasher/pkg/sizeparser"
)

// Validator provides comprehensive validation for trasher inputs and system conditions.
type Validator struct {
	// No internal state needed - all methods are stateless
}

// ValidationConfig holds all the parameters that need to be validated.
type ValidationConfig struct {
	Size       string
	Pattern    string
	OutputPath string
	Workers    int
	ChunkSize  string
	Force      bool
}

// ValidationError represents a validation error with a user-friendly message.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("%s: %s", e.Field, e.Message)
	}
	return e.Message
}

// NewValidator creates a new validator instance.
func NewValidator() *Validator {
	return &Validator{}
}

// ValidateAll performs comprehensive validation of all input parameters and system conditions.
func (v *Validator) ValidateAll(config ValidationConfig) error {
	// Validate size first as it's needed for other validations
	sizeBytes, err := v.ValidateSize(config.Size)
	if err != nil {
		return err
	}

	// Validate pattern
	if err := v.ValidatePattern(config.Pattern); err != nil {
		return err
	}

	// Validate output path
	if err := v.ValidateOutputPath(config.OutputPath, config.Force); err != nil {
		return err
	}

	// Validate disk space
	if err := v.ValidateDiskSpace(config.OutputPath, sizeBytes); err != nil {
		return err
	}

	// Validate file system capabilities
	if err := v.ValidateFileSystemCapabilities(config.OutputPath, sizeBytes); err != nil {
		return err
	}

	// Validate worker count
	if err := v.ValidateWorkers(config.Workers); err != nil {
		return err
	}

	// Validate chunk size
	if err := v.ValidateChunkSize(config.ChunkSize); err != nil {
		return err
	}

	return nil
}

// ValidateSize validates the size specification and returns the size in bytes.
func (v *Validator) ValidateSize(size string) (int64, error) {
	if size == "" {
		return 0, &ValidationError{
			Field:   "size",
			Message: "size cannot be empty",
		}
	}

	sizeBytes, err := sizeparser.Parse(size)
	if err != nil {
		return 0, &ValidationError{
			Field:   "size",
			Message: fmt.Sprintf("invalid size format: %v", err),
		}
	}

	// Additional validations beyond what sizeparser already does
	if sizeBytes < 1 {
		return 0, &ValidationError{
			Field:   "size",
			Message: "size must be at least 1 byte",
		}
	}

	// Check for reasonable maximum (sizeparser already enforces 10PB)
	maxSize := int64(10) * (1024 * 1024 * 1024 * 1024 * 1024) // 10PB
	if sizeBytes > maxSize {
		return 0, &ValidationError{
			Field:   "size",
			Message: "size must be at most 10PB",
		}
	}

	return sizeBytes, nil
}

// ValidatePattern validates the data generation pattern.
func (v *Validator) ValidatePattern(pattern string) error {
	if pattern == "" {
		return &ValidationError{
			Field:   "pattern",
			Message: "pattern cannot be empty",
		}
	}

	// Use the generator package to validate available patterns
	availablePatterns := generator.AvailablePatterns()
	for _, valid := range availablePatterns {
		if pattern == valid {
			return nil
		}
	}

	return &ValidationError{
		Field:   "pattern",
		Message: fmt.Sprintf("invalid pattern '%s' (available: %v)", pattern, availablePatterns),
	}
}

// ValidateOutputPath validates the output file path and directory permissions.
func (v *Validator) ValidateOutputPath(path string, force bool) error {
	if path == "" {
		return &ValidationError{
			Field:   "output",
			Message: "output path cannot be empty",
		}
	}

	// Check if file already exists and force flag
	if _, err := os.Stat(path); err == nil && !force {
		return &ValidationError{
			Field:   "output",
			Message: fmt.Sprintf("file '%s' already exists (use --force to overwrite)", path),
		}
	}

	// Validate directory exists and is writable
	dir := filepath.Dir(path)
	if err := v.validateDirectory(dir); err != nil {
		return &ValidationError{
			Field:   "output",
			Message: err.Error(),
		}
	}

	return nil
}

// validateDirectory checks if a directory exists and is writable.
func (v *Validator) validateDirectory(dir string) error {
	info, err := os.Stat(dir)
	if os.IsNotExist(err) {
		return fmt.Errorf("directory '%s' does not exist", dir)
	}
	if err != nil {
		return fmt.Errorf("cannot access directory '%s': %v", dir, err)
	}

	if !info.IsDir() {
		return fmt.Errorf("'%s' is not a directory", dir)
	}

	// Test write permissions by creating a temporary file
	tempFile, err := os.CreateTemp(dir, ".trasher-write-test-")
	if err != nil {
		return fmt.Errorf("directory '%s' is not writable: %v", dir, err)
	}
	tempFile.Close()
	os.Remove(tempFile.Name())

	return nil
}

// ValidateDiskSpace checks if there's sufficient disk space for the file.
func (v *Validator) ValidateDiskSpace(path string, size int64) error {
	dir := filepath.Dir(path)

	var stat syscall.Statfs_t
	if err := syscall.Statfs(dir, &stat); err != nil {
		return &ValidationError{
			Field:   "disk_space",
			Message: fmt.Sprintf("failed to check disk space: %v", err),
		}
	}

	available := int64(stat.Bavail) * int64(stat.Bsize)
	if size > available {
		return &ValidationError{
			Field:   "disk_space",
			Message: fmt.Sprintf("insufficient disk space: need %s, have %s", 
				formatSize(size), formatSize(available)),
		}
	}

	// Warn if less than 10% free space will remain
	total := int64(stat.Blocks) * int64(stat.Bsize)
	remaining := available - size
	if remaining < total/10 {
		// This is a warning, not an error, so we don't return it
		// In a real implementation, we might want a separate warning system
	}

	return nil
}

// ValidateFileSystemCapabilities checks file system limitations.
func (v *Validator) ValidateFileSystemCapabilities(path string, size int64) error {
	dir := filepath.Dir(path)

	var stat syscall.Statfs_t
	if err := syscall.Statfs(dir, &stat); err != nil {
		return &ValidationError{
			Field:   "filesystem",
			Message: fmt.Sprintf("failed to check file system: %v", err),
		}
	}

	// Check for known file system limitations
	// Note: The exact magic numbers may vary by platform
	const (
		// Common file system magic numbers (these are Linux-specific)
		EXT2_SUPER_MAGIC  = 0xEF53
		EXT3_SUPER_MAGIC  = 0xEF53
		EXT4_SUPER_MAGIC  = 0xEF53
		XFS_SUPER_MAGIC   = 0x58465342
		BTRFS_SUPER_MAGIC = 0x9123683E
	)

	// For portability, we'll focus on general size limits rather than
	// trying to detect specific file system types
	
	// Most modern file systems support very large files, but let's check
	// for some reasonable limits
	const maxSingleFileSize = int64(8) * 1024 * 1024 * 1024 * 1024 * 1024 // 8EB (exabytes)
	
	if size > maxSingleFileSize {
		return &ValidationError{
			Field:   "filesystem",
			Message: fmt.Sprintf("file size %s exceeds maximum file size limit", formatSize(size)),
		}
	}

	return nil
}

// ValidateWorkers validates the worker count.
func (v *Validator) ValidateWorkers(workers int) error {
	if workers < 1 {
		return &ValidationError{
			Field:   "workers",
			Message: "worker count must be at least 1",
		}
	}

	// Set a reasonable maximum based on CPU count
	maxWorkers := runtime.NumCPU() * 4
	if workers > maxWorkers {
		return &ValidationError{
			Field:   "workers",
			Message: fmt.Sprintf("worker count %d exceeds recommended maximum of %d (4x CPU count)", 
				workers, maxWorkers),
		}
	}

	return nil
}

// ValidateChunkSize validates the chunk size specification.
func (v *Validator) ValidateChunkSize(chunkSize string) error {
	if chunkSize == "" {
		return &ValidationError{
			Field:   "chunk_size",
			Message: "chunk size cannot be empty",
		}
	}

	size, err := sizeparser.Parse(chunkSize)
	if err != nil {
		return &ValidationError{
			Field:   "chunk_size",
			Message: fmt.Sprintf("invalid chunk size format: %v", err),
		}
	}

	const minChunkSize = 1024        // 1KB
	const maxChunkSize = 1024 * 1024 * 1024 // 1GB

	if size < minChunkSize {
		return &ValidationError{
			Field:   "chunk_size",
			Message: fmt.Sprintf("chunk size must be at least %s", formatSize(minChunkSize)),
		}
	}

	if size > maxChunkSize {
		return &ValidationError{
			Field:   "chunk_size",
			Message: fmt.Sprintf("chunk size must be at most %s", formatSize(maxChunkSize)),
		}
	}

	return nil
}

// formatSize formats a byte count into a human-readable string.
func formatSize(bytes int64) string {
	if bytes == 0 {
		return "0 B"
	}

	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// ValidateConfiguration is a convenience function that validates a complete configuration.
func ValidateConfiguration(size, pattern, outputPath string, workers int, chunkSize string, force bool) error {
	validator := NewValidator()
	config := ValidationConfig{
		Size:       size,
		Pattern:    pattern,
		OutputPath: outputPath,
		Workers:    workers,
		ChunkSize:  chunkSize,
		Force:      force,
	}
	return validator.ValidateAll(config)
}

// GetSystemInfo returns information about the system capabilities.
func GetSystemInfo(path string) (*SystemInfo, error) {
	dir := filepath.Dir(path)
	
	var stat syscall.Statfs_t
	if err := syscall.Statfs(dir, &stat); err != nil {
		return nil, fmt.Errorf("failed to get system info: %v", err)
	}

	return &SystemInfo{
		AvailableSpace: int64(stat.Bavail) * int64(stat.Bsize),
		TotalSpace:     int64(stat.Blocks) * int64(stat.Bsize),
		CPUCount:       runtime.NumCPU(),
		MaxWorkers:     runtime.NumCPU() * 4,
	}, nil
}

// SystemInfo holds information about system capabilities.
type SystemInfo struct {
	AvailableSpace int64
	TotalSpace     int64
	CPUCount       int
	MaxWorkers     int
}