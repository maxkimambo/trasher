package validation

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestNewValidator(t *testing.T) {
	validator := NewValidator()
	if validator == nil {
		t.Error("NewValidator should return a non-nil validator")
	}
}

func TestValidateSize(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name        string
		size        string
		expectError bool
		expectedMsg string
	}{
		{"valid size", "1GB", false, ""},
		{"empty size", "", true, "size cannot be empty"},
		{"invalid format", "invalid", true, "invalid size format"},
		{"zero size", "0B", true, "size must be at least 1 byte"},
		{"valid minimum", "1B", false, ""},
		{"valid large", "1TB", false, ""},
		{"negative implied", "-1GB", true, "invalid size format"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := validator.ValidateSize(test.size)
			
			if test.expectError && err == nil {
				t.Errorf("expected error for size '%s'", test.size)
			}
			if !test.expectError && err != nil {
				t.Errorf("unexpected error for size '%s': %v", test.size, err)
			}
			if test.expectError && test.expectedMsg != "" {
				if err == nil || !strings.Contains(err.Error(), test.expectedMsg) {
					t.Errorf("expected error message to contain '%s', got: %v", test.expectedMsg, err)
				}
			}
		})
	}
}

func TestValidatePattern(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name        string
		pattern     string
		expectError bool
	}{
		{"valid random", "random", false},
		{"valid sequential", "sequential", false},
		{"valid zero", "zero", false},
		{"valid mixed", "mixed", false},
		{"invalid pattern", "invalid", true},
		{"empty pattern", "", true},
		{"case sensitive", "Random", true}, // patterns are case-sensitive
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validator.ValidatePattern(test.pattern)
			
			if test.expectError && err == nil {
				t.Errorf("expected error for pattern '%s'", test.pattern)
			}
			if !test.expectError && err != nil {
				t.Errorf("unexpected error for pattern '%s': %v", test.pattern, err)
			}
		})
	}
}

func TestValidateOutputPath(t *testing.T) {
	validator := NewValidator()
	tempDir := t.TempDir()

	tests := []struct {
		name        string
		path        string
		force       bool
		setupFunc   func() string // Returns actual path to test
		expectError bool
		expectedMsg string
	}{
		{
			name:        "empty path",
			path:        "",
			force:       false,
			setupFunc:   func() string { return "" },
			expectError: true,
			expectedMsg: "output path cannot be empty",
		},
		{
			name:  "valid new file",
			path:  "newfile.bin",
			force: false,
			setupFunc: func() string {
				return filepath.Join(tempDir, "newfile.bin")
			},
			expectError: false,
		},
		{
			name:  "existing file without force",
			path:  "existing.bin",
			force: false,
			setupFunc: func() string {
				path := filepath.Join(tempDir, "existing.bin")
				os.WriteFile(path, []byte("test"), 0644)
				return path
			},
			expectError: true,
			expectedMsg: "already exists",
		},
		{
			name:  "existing file with force",
			path:  "existing_force.bin",
			force: true,
			setupFunc: func() string {
				path := filepath.Join(tempDir, "existing_force.bin")
				os.WriteFile(path, []byte("test"), 0644)
				return path
			},
			expectError: false,
		},
		{
			name:        "nonexistent directory",
			path:        "nonexistent.bin",
			force:       false,
			setupFunc:   func() string { return "/nonexistent/dir/file.bin" },
			expectError: true,
			expectedMsg: "does not exist",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actualPath := test.setupFunc()
			err := validator.ValidateOutputPath(actualPath, test.force)
			
			if test.expectError && err == nil {
				t.Errorf("expected error for path '%s'", actualPath)
			}
			if !test.expectError && err != nil {
				t.Errorf("unexpected error for path '%s': %v", actualPath, err)
			}
			if test.expectError && test.expectedMsg != "" {
				if err == nil || !strings.Contains(err.Error(), test.expectedMsg) {
					t.Errorf("expected error message to contain '%s', got: %v", test.expectedMsg, err)
				}
			}
		})
	}
}

func TestValidateDiskSpace(t *testing.T) {
	validator := NewValidator()
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.bin")

	// Test with a reasonable size that should be available
	err := validator.ValidateDiskSpace(testFile, 1024*1024) // 1MB
	if err != nil {
		t.Errorf("unexpected error for reasonable size: %v", err)
	}

	// Test with an unreasonably large size (this may or may not fail depending on system)
	// We'll use a very large size that's likely to exceed available space
	err = validator.ValidateDiskSpace(testFile, 1024*1024*1024*1024*1024) // 1PB
	if err == nil {
		t.Log("Warning: 1PB validation passed - system has very large available space")
	}
}

func TestValidateWorkers(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name        string
		workers     int
		expectError bool
		expectedMsg string
	}{
		{"valid workers", 2, false, ""},
		{"minimum workers", 1, false, ""},
		{"zero workers", 0, true, "must be at least 1"},
		{"negative workers", -1, true, "must be at least 1"},
		{"max workers", runtime.NumCPU(), false, ""},
		{"excessive workers", runtime.NumCPU()*10, true, "exceeds recommended maximum"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validator.ValidateWorkers(test.workers)
			
			if test.expectError && err == nil {
				t.Errorf("expected error for workers %d", test.workers)
			}
			if !test.expectError && err != nil {
				t.Errorf("unexpected error for workers %d: %v", test.workers, err)
			}
			if test.expectError && test.expectedMsg != "" {
				if err == nil || !strings.Contains(err.Error(), test.expectedMsg) {
					t.Errorf("expected error message to contain '%s', got: %v", test.expectedMsg, err)
				}
			}
		})
	}
}

func TestValidateChunkSize(t *testing.T) {
	validator := NewValidator()

	tests := []struct {
		name        string
		chunkSize   string
		expectError bool
		expectedMsg string
	}{
		{"valid chunk size", "64MB", false, ""},
		{"minimum chunk size", "1KB", false, ""},
		{"empty chunk size", "", true, "chunk size cannot be empty"},
		{"invalid format", "invalid", true, "invalid chunk size format"},
		{"too small", "512B", true, "chunk size must be at least"},
		{"too large", "2GB", true, "chunk size must be at most"},
		{"maximum chunk size", "1GB", false, ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validator.ValidateChunkSize(test.chunkSize)
			
			if test.expectError && err == nil {
				t.Errorf("expected error for chunk size '%s'", test.chunkSize)
			}
			if !test.expectError && err != nil {
				t.Errorf("unexpected error for chunk size '%s': %v", test.chunkSize, err)
			}
			if test.expectError && test.expectedMsg != "" {
				if err == nil || !strings.Contains(err.Error(), test.expectedMsg) {
					t.Errorf("expected error message to contain '%s', got: %v", test.expectedMsg, err)
				}
			}
		})
	}
}

func TestValidateAll(t *testing.T) {
	validator := NewValidator()
	tempDir := t.TempDir()

	tests := []struct {
		name        string
		config      ValidationConfig
		expectError bool
		expectedMsg string
	}{
		{
			name: "valid configuration",
			config: ValidationConfig{
				Size:       "1GB",
				Pattern:    "random",
				OutputPath: filepath.Join(tempDir, "valid.bin"),
				Workers:    2,
				ChunkSize:  "64MB",
				Force:      false,
			},
			expectError: false,
		},
		{
			name: "invalid size",
			config: ValidationConfig{
				Size:       "invalid",
				Pattern:    "random",
				OutputPath: filepath.Join(tempDir, "test.bin"),
				Workers:    2,
				ChunkSize:  "64MB",
				Force:      false,
			},
			expectError: true,
			expectedMsg: "size",
		},
		{
			name: "invalid pattern",
			config: ValidationConfig{
				Size:       "1GB",
				Pattern:    "invalid",
				OutputPath: filepath.Join(tempDir, "test.bin"),
				Workers:    2,
				ChunkSize:  "64MB",
				Force:      false,
			},
			expectError: true,
			expectedMsg: "pattern",
		},
		{
			name: "invalid workers",
			config: ValidationConfig{
				Size:       "1GB",
				Pattern:    "random",
				OutputPath: filepath.Join(tempDir, "test.bin"),
				Workers:    0,
				ChunkSize:  "64MB",
				Force:      false,
			},
			expectError: true,
			expectedMsg: "workers",
		},
		{
			name: "invalid chunk size",
			config: ValidationConfig{
				Size:       "1GB",
				Pattern:    "random",
				OutputPath: filepath.Join(tempDir, "test.bin"),
				Workers:    2,
				ChunkSize:  "invalid",
				Force:      false,
			},
			expectError: true,
			expectedMsg: "chunk_size",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validator.ValidateAll(test.config)
			
			if test.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !test.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if test.expectError && test.expectedMsg != "" {
				if err == nil || !strings.Contains(err.Error(), test.expectedMsg) {
					t.Errorf("expected error message to contain '%s', got: %v", test.expectedMsg, err)
				}
			}
		})
	}
}

func TestValidateConfiguration(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "config_test.bin")

	// Test valid configuration
	err := ValidateConfiguration("1GB", "random", testFile, 2, "64MB", false)
	if err != nil {
		t.Errorf("unexpected error for valid configuration: %v", err)
	}

	// Test invalid configuration
	err = ValidateConfiguration("invalid", "random", testFile, 2, "64MB", false)
	if err == nil {
		t.Error("expected error for invalid size")
	}
}

func TestGetSystemInfo(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "system_test.bin")

	info, err := GetSystemInfo(testFile)
	if err != nil {
		t.Fatalf("failed to get system info: %v", err)
	}

	if info.AvailableSpace <= 0 {
		t.Error("available space should be positive")
	}
	if info.TotalSpace <= 0 {
		t.Error("total space should be positive")
	}
	if info.CPUCount <= 0 {
		t.Error("CPU count should be positive")
	}
	if info.MaxWorkers != info.CPUCount*4 {
		t.Errorf("max workers should be 4x CPU count, got %d for %d CPUs", 
			info.MaxWorkers, info.CPUCount)
	}
	if info.AvailableSpace > info.TotalSpace {
		t.Error("available space should not exceed total space")
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1024 * 1024, "1.0 MB"},
		{1024 * 1024 * 1024, "1.0 GB"},
		{int64(1024) * 1024 * 1024 * 1024, "1.0 TB"},
		{int64(1024) * 1024 * 1024 * 1024 * 1024, "1.0 PB"},
		{int64(1024) * 1024 * 1024 * 1024 * 1024 * 1024, "1.0 EB"},
	}

	for _, test := range tests {
		result := formatSize(test.bytes)
		if result != test.expected {
			t.Errorf("formatSize(%d) = %s, expected %s", 
				test.bytes, result, test.expected)
		}
	}
}

func TestValidationError(t *testing.T) {
	// Test ValidationError with field
	err := &ValidationError{
		Field:   "test_field",
		Message: "test message",
	}
	expected := "test_field: test message"
	if err.Error() != expected {
		t.Errorf("expected '%s', got '%s'", expected, err.Error())
	}

	// Test ValidationError without field
	err = &ValidationError{
		Message: "test message",
	}
	expected = "test message"
	if err.Error() != expected {
		t.Errorf("expected '%s', got '%s'", expected, err.Error())
	}
}

func TestValidateFileSystemCapabilities(t *testing.T) {
	validator := NewValidator()
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "fs_test.bin")

	// Test with reasonable size
	err := validator.ValidateFileSystemCapabilities(testFile, 1024*1024*1024) // 1GB
	if err != nil {
		t.Errorf("unexpected error for 1GB file: %v", err)
	}

	// Test with extremely large size
	err = validator.ValidateFileSystemCapabilities(testFile, 9*1024*1024*1024*1024*1024) // 9EB
	if err == nil {
		t.Error("expected error for extremely large file size")
	}
}

func TestValidateDirectory(t *testing.T) {
	validator := NewValidator()
	tempDir := t.TempDir()

	// Test valid directory
	err := validator.validateDirectory(tempDir)
	if err != nil {
		t.Errorf("unexpected error for valid directory: %v", err)
	}

	// Test nonexistent directory
	err = validator.validateDirectory("/nonexistent/directory")
	if err == nil {
		t.Error("expected error for nonexistent directory")
	}

	// Test file instead of directory
	testFile := filepath.Join(tempDir, "notadir")
	os.WriteFile(testFile, []byte("test"), 0644)
	err = validator.validateDirectory(testFile)
	if err == nil {
		t.Error("expected error when path is a file, not a directory")
	}

	// Test read-only directory (if possible)
	readOnlyDir := filepath.Join(tempDir, "readonly")
	if err := os.Mkdir(readOnlyDir, 0444); err == nil {
		defer os.Chmod(readOnlyDir, 0755) // Restore permissions for cleanup
		err = validator.validateDirectory(readOnlyDir)
		if err == nil {
			t.Error("expected error for read-only directory")
		}
	}
}

// Benchmark tests
func BenchmarkValidateAll(b *testing.B) {
	validator := NewValidator()
	tempDir := b.TempDir()
	
	config := ValidationConfig{
		Size:       "1GB",
		Pattern:    "random",
		OutputPath: filepath.Join(tempDir, "bench.bin"),
		Workers:    2,
		ChunkSize:  "64MB",
		Force:      false,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator.ValidateAll(config)
	}
}

func BenchmarkValidateSize(b *testing.B) {
	validator := NewValidator()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator.ValidateSize("1GB")
	}
}

func BenchmarkFormatSize(b *testing.B) {
	size := int64(1024 * 1024 * 1024) // 1GB
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		formatSize(size)
	}
}