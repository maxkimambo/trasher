package writer

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"syscall"
)

// FileWriter provides thread-safe writing to a file at specific offsets.
type FileWriter struct {
	file      *os.File
	mu        sync.Mutex
	written   int64
	totalSize int64
	path      string
}

// NewFileWriter creates a new FileWriter that writes to the specified path.
// If force is false and the file exists, an error is returned.
// The file is pre-allocated to the specified size if possible.
func NewFileWriter(path string, size int64, force bool) (*FileWriter, error) {
	if size <= 0 {
		return nil, fmt.Errorf("file size must be positive, got %d", size)
	}

	// Check if file exists and handle --force flag
	if _, err := os.Stat(path); err == nil && !force {
		return nil, fmt.Errorf("file %s already exists, use --force to overwrite", path)
	}

	// Validate directory exists and is writable
	dir := filepath.Dir(path)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, fmt.Errorf("directory %s does not exist", dir)
	}

	// Check directory is writable by attempting to create a temp file
	tempFile := filepath.Join(dir, ".trasher_write_test")
	if f, err := os.Create(tempFile); err != nil {
		return nil, fmt.Errorf("directory %s is not writable: %v", dir, err)
	} else {
		f.Close()
		os.Remove(tempFile)
	}

	// Check available disk space
	if err := checkDiskSpace(dir, size); err != nil {
		return nil, err
	}

	// Create or truncate the file
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to create output file: %v", err)
	}

	// Pre-allocate file space if possible
	if err := preAllocateFile(file, size); err != nil {
		file.Close()
		return nil, err
	}

	return &FileWriter{
		file:      file,
		totalSize: size,
		path:      path,
	}, nil
}

// WriteAt writes data at the specified offset in the file.
// This method is thread-safe and can be called concurrently.
func (w *FileWriter) WriteAt(data []byte, offset int64) error {
	if len(data) == 0 {
		return nil
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file == nil {
		return fmt.Errorf("file writer is closed")
	}

	// Validate offset and size
	if offset < 0 {
		return fmt.Errorf("offset cannot be negative: %d", offset)
	}
	if offset+int64(len(data)) > w.totalSize {
		return fmt.Errorf("write would exceed file size: offset=%d, len=%d, total=%d", 
			offset, len(data), w.totalSize)
	}

	// Seek to the correct position
	if _, err := w.file.Seek(offset, io.SeekStart); err != nil {
		return fmt.Errorf("failed to seek to position %d: %v", offset, err)
	}

	// Write the data
	n, err := w.file.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write data at offset %d: %v", offset, err)
	}
	if n != len(data) {
		return fmt.Errorf("short write: wrote %d bytes out of %d", n, len(data))
	}

	w.written += int64(n)
	return nil
}

// Close closes the file and syncs any pending writes to disk.
func (w *FileWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file == nil {
		return nil
	}

	// Sync to ensure all data is written to disk
	if err := w.file.Sync(); err != nil {
		w.file.Close()
		w.file = nil
		return fmt.Errorf("failed to sync file: %v", err)
	}

	err := w.file.Close()
	w.file = nil
	return err
}

// Written returns the total number of bytes written so far.
func (w *FileWriter) Written() int64 {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.written
}

// TotalSize returns the total expected size of the file.
func (w *FileWriter) TotalSize() int64 {
	return w.totalSize
}

// Path returns the file path.
func (w *FileWriter) Path() string {
	return w.path
}

// checkDiskSpace verifies that there's enough disk space available.
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

// preAllocateFile attempts to pre-allocate file space for better performance.
func preAllocateFile(file *os.File, size int64) error {
	// Try platform-specific allocation first
	if err := tryFallocate(file, size); err == nil {
		return nil
	}

	// Fallback: seek to end and write a single byte
	if _, err := file.Seek(size-1, io.SeekStart); err != nil {
		return fmt.Errorf("failed to seek to end of file: %v", err)
	}
	if _, err := file.Write([]byte{0}); err != nil {
		return fmt.Errorf("failed to write last byte: %v", err)
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("failed to seek back to start: %v", err)
	}

	return nil
}

// tryFallocate attempts to use platform-specific file allocation.
func tryFallocate(file *os.File, size int64) error {
	// For now, we'll use the portable fallback approach across all platforms
	// In a production implementation, we could add platform-specific optimizations
	return fmt.Errorf("using portable allocation method")
}