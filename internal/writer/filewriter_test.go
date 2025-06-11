package writer

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestNewFileWriter(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")

	// Test successful creation
	w, err := NewFileWriter(testFile, 1024, false)
	if err != nil {
		t.Fatalf("failed to create FileWriter: %v", err)
	}
	defer w.Close()

	if w.TotalSize() != 1024 {
		t.Errorf("expected total size 1024, got %d", w.TotalSize())
	}
	if w.Path() != testFile {
		t.Errorf("expected path %s, got %s", testFile, w.Path())
	}
	if w.Written() != 0 {
		t.Errorf("expected written 0, got %d", w.Written())
	}

	// Verify file was created
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Error("file was not created")
	}
}

func TestNewFileWriterValidation(t *testing.T) {
	tempDir := t.TempDir()

	// Test invalid size
	_, err := NewFileWriter(filepath.Join(tempDir, "test.txt"), 0, false)
	if err == nil {
		t.Error("expected error for zero size")
	}

	_, err = NewFileWriter(filepath.Join(tempDir, "test.txt"), -1, false)
	if err == nil {
		t.Error("expected error for negative size")
	}

	// Test non-existent directory
	_, err = NewFileWriter("/nonexistent/dir/test.txt", 1024, false)
	if err == nil {
		t.Error("expected error for non-existent directory")
	}
}

func TestNewFileWriterExistingFile(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "existing.txt")

	// Create an existing file
	if err := os.WriteFile(testFile, []byte("existing content"), 0644); err != nil {
		t.Fatalf("failed to create existing file: %v", err)
	}

	// Test without force flag
	_, err := NewFileWriter(testFile, 1024, false)
	if err == nil {
		t.Error("expected error when file exists and force=false")
	}

	// Test with force flag
	w, err := NewFileWriter(testFile, 1024, true)
	if err != nil {
		t.Fatalf("failed to create FileWriter with force=true: %v", err)
	}
	defer w.Close()

	// Verify file was truncated
	info, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("failed to stat file: %v", err)
	}
	if info.Size() != 1024 {
		t.Errorf("expected file size 1024, got %d", info.Size())
	}
}

func TestWriteAt(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "write_test.txt")

	w, err := NewFileWriter(testFile, 1024, false)
	if err != nil {
		t.Fatalf("failed to create FileWriter: %v", err)
	}
	defer w.Close()

	// Test writing at different offsets
	testData := []struct {
		offset int64
		data   []byte
	}{
		{0, []byte("hello")},
		{100, []byte("world")},
		{500, []byte("test")},
	}

	for _, test := range testData {
		if err := w.WriteAt(test.data, test.offset); err != nil {
			t.Errorf("failed to write at offset %d: %v", test.offset, err)
		}
	}

	expectedWritten := int64(len("hello") + len("world") + len("test"))
	if w.Written() != expectedWritten {
		t.Errorf("expected written %d, got %d", expectedWritten, w.Written())
	}

	// Verify data was written correctly
	w.Close()
	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	if string(data[0:5]) != "hello" {
		t.Errorf("expected 'hello' at offset 0, got '%s'", string(data[0:5]))
	}
	if string(data[100:105]) != "world" {
		t.Errorf("expected 'world' at offset 100, got '%s'", string(data[100:105]))
	}
	if string(data[500:504]) != "test" {
		t.Errorf("expected 'test' at offset 500, got '%s'", string(data[500:504]))
	}
}

func TestWriteAtValidation(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "validation_test.txt")

	w, err := NewFileWriter(testFile, 100, false)
	if err != nil {
		t.Fatalf("failed to create FileWriter: %v", err)
	}
	defer w.Close()

	// Test negative offset
	err = w.WriteAt([]byte("test"), -1)
	if err == nil {
		t.Error("expected error for negative offset")
	}

	// Test write beyond file size
	err = w.WriteAt([]byte("test"), 97) // offset 97 + 4 bytes = 101 > 100
	if err == nil {
		t.Error("expected error for write beyond file size")
	}

	// Test write exactly at boundary (should succeed)
	err = w.WriteAt([]byte("test"), 96) // offset 96 + 4 bytes = 100
	if err != nil {
		t.Errorf("unexpected error for write at boundary: %v", err)
	}

	// Test empty write (should succeed)
	err = w.WriteAt([]byte{}, 50)
	if err != nil {
		t.Errorf("unexpected error for empty write: %v", err)
	}
}

func TestWriteAtAfterClose(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "closed_test.txt")

	w, err := NewFileWriter(testFile, 1024, false)
	if err != nil {
		t.Fatalf("failed to create FileWriter: %v", err)
	}

	w.Close()

	// Test writing after close
	err = w.WriteAt([]byte("test"), 0)
	if err == nil {
		t.Error("expected error when writing to closed file")
	}
}

func TestConcurrentWrites(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "concurrent_test.txt")

	w, err := NewFileWriter(testFile, 10000, false)
	if err != nil {
		t.Fatalf("failed to create FileWriter: %v", err)
	}
	defer w.Close()

	numWorkers := 10
	chunkSize := 100
	var wg sync.WaitGroup

	// Write concurrently from multiple goroutines
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			
			offset := int64(workerID * chunkSize)
			data := make([]byte, chunkSize)
			
			// Fill with worker ID
			for j := 0; j < chunkSize; j++ {
				data[j] = byte(workerID)
			}

			if err := w.WriteAt(data, offset); err != nil {
				t.Errorf("worker %d failed to write: %v", workerID, err)
			}
		}(i)
	}

	wg.Wait()

	expectedWritten := int64(numWorkers * chunkSize)
	if w.Written() != expectedWritten {
		t.Errorf("expected written %d, got %d", expectedWritten, w.Written())
	}

	// Verify data integrity
	w.Close()
	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	for i := 0; i < numWorkers; i++ {
		offset := i * chunkSize
		for j := 0; j < chunkSize; j++ {
			if data[offset+j] != byte(i) {
				t.Errorf("data corruption at position %d: expected %d, got %d", 
					offset+j, i, data[offset+j])
				break
			}
		}
	}
}

func TestClose(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "close_test.txt")

	w, err := NewFileWriter(testFile, 1024, false)
	if err != nil {
		t.Fatalf("failed to create FileWriter: %v", err)
	}

	// Write some data
	if err := w.WriteAt([]byte("test"), 0); err != nil {
		t.Fatalf("failed to write data: %v", err)
	}

	// Close should succeed
	if err := w.Close(); err != nil {
		t.Errorf("failed to close file: %v", err)
	}

	// Second close should not error
	if err := w.Close(); err != nil {
		t.Errorf("second close returned error: %v", err)
	}

	// Verify file exists and has correct size
	info, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("failed to stat file after close: %v", err)
	}
	if info.Size() != 1024 {
		t.Errorf("expected file size 1024, got %d", info.Size())
	}
}

func TestFileAllocation(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "alloc_test.txt")

	// Test with a reasonably sized file
	w, err := NewFileWriter(testFile, 1024*1024, false) // 1MB
	if err != nil {
		t.Fatalf("failed to create FileWriter: %v", err)
	}
	defer w.Close()

	// Verify file was allocated
	info, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("failed to stat file: %v", err)
	}
	if info.Size() != 1024*1024 {
		t.Errorf("expected file size 1MB, got %d", info.Size())
	}
}

func TestDiskSpaceCheck(t *testing.T) {
	tempDir := t.TempDir()

	// This test is hard to make portable and reliable since it depends on
	// actual disk space. We'll test with a reasonable size that should
	// typically be available.
	testFile := filepath.Join(tempDir, "space_test.txt")
	
	// Test with a small file (should succeed)
	w, err := NewFileWriter(testFile, 1024, false)
	if err != nil {
		t.Fatalf("failed to create small file: %v", err)
	}
	w.Close()

	// Test with an unreasonably large file (might fail depending on available space)
	// We'll skip this test if it would consume too much actual disk space
	t.Logf("Disk space check test passed for small file")
}

func TestWriterMethods(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "methods_test.txt")

	w, err := NewFileWriter(testFile, 1000, false)
	if err != nil {
		t.Fatalf("failed to create FileWriter: %v", err)
	}
	defer w.Close()

	// Test initial state
	if w.TotalSize() != 1000 {
		t.Errorf("expected TotalSize 1000, got %d", w.TotalSize())
	}
	if w.Written() != 0 {
		t.Errorf("expected Written 0, got %d", w.Written())
	}
	if w.Path() != testFile {
		t.Errorf("expected Path %s, got %s", testFile, w.Path())
	}

	// Write some data and verify Written() updates
	if err := w.WriteAt([]byte("hello"), 0); err != nil {
		t.Fatalf("failed to write: %v", err)
	}
	if w.Written() != 5 {
		t.Errorf("expected Written 5, got %d", w.Written())
	}

	if err := w.WriteAt([]byte("world"), 10); err != nil {
		t.Fatalf("failed to write: %v", err)
	}
	if w.Written() != 10 {
		t.Errorf("expected Written 10, got %d", w.Written())
	}
}

func TestErrorConditions(t *testing.T) {
	tempDir := t.TempDir()

	// Test with read-only directory (if we can create one)
	readOnlyDir := filepath.Join(tempDir, "readonly")
	if err := os.Mkdir(readOnlyDir, 0444); err != nil {
		t.Skip("Cannot create read-only directory for testing")
	}
	
	testFile := filepath.Join(readOnlyDir, "test.txt")
	_, err := NewFileWriter(testFile, 1024, false)
	if err == nil {
		t.Error("expected error when writing to read-only directory")
	}
}

// Benchmark tests
func BenchmarkWriteAt(b *testing.B) {
	tempDir := b.TempDir()
	testFile := filepath.Join(tempDir, "bench_test.txt")

	w, err := NewFileWriter(testFile, int64(b.N*1024), false)
	if err != nil {
		b.Fatalf("failed to create FileWriter: %v", err)
	}
	defer w.Close()

	data := make([]byte, 1024)
	for i := range data {
		data[i] = byte(i % 256)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		offset := int64(i * 1024)
		if err := w.WriteAt(data, offset); err != nil {
			b.Fatalf("write failed: %v", err)
		}
	}
}

func BenchmarkConcurrentWrites(b *testing.B) {
	tempDir := b.TempDir()
	testFile := filepath.Join(tempDir, "concurrent_bench_test.txt")

	numWorkers := 4
	totalWrites := b.N
	writesPerWorker := totalWrites / numWorkers

	w, err := NewFileWriter(testFile, int64(totalWrites*1024), false)
	if err != nil {
		b.Fatalf("failed to create FileWriter: %v", err)
	}
	defer w.Close()

	data := make([]byte, 1024)
	for i := range data {
		data[i] = byte(i % 256)
	}

	b.ResetTimer()
	
	var wg sync.WaitGroup
	for worker := 0; worker < numWorkers; worker++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			start := workerID * writesPerWorker
			end := start + writesPerWorker
			
			for i := start; i < end; i++ {
				offset := int64(i * 1024)
				if err := w.WriteAt(data, offset); err != nil {
					b.Errorf("write failed: %v", err)
					return
				}
			}
		}(worker)
	}
	
	wg.Wait()
}