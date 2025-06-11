package signal

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maxkimambo/trasher/internal/progress"
	"github.com/maxkimambo/trasher/internal/writer"
)

func TestNewShutdownHandler(t *testing.T) {
	var buf bytes.Buffer
	ctx := context.Background()
	
	handler := NewShutdownHandler(ctx, &buf)
	
	if handler == nil {
		t.Fatal("handler should not be nil")
	}
	if handler.ctx == nil {
		t.Error("context should not be nil")
	}
	if handler.cancel == nil {
		t.Error("cancel function should not be nil")
	}
	if handler.sigChan == nil {
		t.Error("signal channel should not be nil")
	}
	if handler.output != &buf {
		t.Error("output writer should be set correctly")
	}
	if handler.IsShutdown() {
		t.Error("handler should not be shutdown initially")
	}
}

func TestNewShutdownHandlerNilOutput(t *testing.T) {
	ctx := context.Background()
	handler := NewShutdownHandler(ctx, nil)
	
	if handler.output != os.Stdout {
		t.Error("output should default to os.Stdout when nil")
	}
}

func TestSetWriter(t *testing.T) {
	var buf bytes.Buffer
	ctx := context.Background()
	handler := NewShutdownHandler(ctx, &buf)
	
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.bin")
	
	fileWriter, err := writer.NewFileWriter(testFile, 1024, false)
	if err != nil {
		t.Fatalf("failed to create file writer: %v", err)
	}
	defer fileWriter.Close()
	
	handler.SetWriter(fileWriter)
	
	// Access through private field for testing (normally not accessible)
	handler.mu.Lock()
	defer handler.mu.Unlock()
	if handler.writer != fileWriter {
		t.Error("writer should be set correctly")
	}
}

func TestSetProgressReporter(t *testing.T) {
	var buf bytes.Buffer
	ctx := context.Background()
	handler := NewShutdownHandler(ctx, &buf)
	
	progressReporter := progress.NewProgressReporter(1024*1024*1024, false, &buf)
	handler.SetProgressReporter(progressReporter)
	
	// Access through private field for testing
	handler.mu.Lock()
	defer handler.mu.Unlock()
	if handler.progress != progressReporter {
		t.Error("progress reporter should be set correctly")
	}
}

func TestRegisterCleanupFunc(t *testing.T) {
	var buf bytes.Buffer
	ctx := context.Background()
	handler := NewShutdownHandler(ctx, &buf)
	
	var callOrder []int
	var mu sync.Mutex
	
	// Register multiple cleanup functions
	for i := 1; i <= 3; i++ {
		id := i
		handler.RegisterCleanupFunc(func() error {
			mu.Lock()
			defer mu.Unlock()
			callOrder = append(callOrder, id)
			return nil
		})
	}
	
	// Trigger shutdown to test cleanup
	handler.Stop()
	
	// Wait a bit for cleanup to complete
	time.Sleep(50 * time.Millisecond)
	
	mu.Lock()
	defer mu.Unlock()
	
	// Cleanup functions should be called in reverse order (LIFO)
	expected := []int{3, 2, 1}
	if len(callOrder) != len(expected) {
		t.Errorf("expected %d cleanup calls, got %d", len(expected), len(callOrder))
	}
	
	for i, id := range callOrder {
		if id != expected[i] {
			t.Errorf("cleanup function %d: expected id %d, got %d", i, expected[i], id)
		}
	}
}

func TestRegisterCleanupFuncWithError(t *testing.T) {
	var buf bytes.Buffer
	ctx := context.Background()
	handler := NewShutdownHandler(ctx, &buf)
	
	// Register a cleanup function that returns an error
	handler.RegisterCleanupFunc(func() error {
		return &testError{message: "cleanup failed"}
	})
	
	handler.Stop()
	time.Sleep(50 * time.Millisecond)
	
	output := buf.String()
	if !strings.Contains(output, "cleanup error") {
		t.Error("output should contain cleanup error message")
	}
	if !strings.Contains(output, "cleanup failed") {
		t.Error("output should contain the specific error message")
	}
}

func TestManualShutdown(t *testing.T) {
	var buf bytes.Buffer
	ctx := context.Background()
	handler := NewShutdownHandler(ctx, &buf)
	
	if handler.IsShutdown() {
		t.Error("handler should not be shutdown initially")
	}
	
	// Test manual shutdown
	handler.Stop()
	
	// Wait a bit for shutdown to complete
	time.Sleep(50 * time.Millisecond)
	
	if !handler.IsShutdown() {
		t.Error("handler should be shutdown after Stop()")
	}
	
	// Context should be cancelled
	select {
	case <-handler.Context().Done():
		// Expected
	default:
		t.Error("context should be cancelled after shutdown")
	}
}

func TestPartialProgressReporting(t *testing.T) {
	var buf bytes.Buffer
	ctx := context.Background()
	handler := NewShutdownHandler(ctx, &buf)
	
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.bin")
	
	// Create a file writer
	fileWriter, err := writer.NewFileWriter(testFile, 1000, false)
	if err != nil {
		t.Fatalf("failed to create file writer: %v", err)
	}
	defer fileWriter.Close()
	
	// Write some data
	data := []byte("test data")
	err = fileWriter.WriteAt(data, 0)
	if err != nil {
		t.Fatalf("failed to write data: %v", err)
	}
	
	handler.SetWriter(fileWriter)
	
	// Trigger shutdown
	handler.Stop()
	time.Sleep(50 * time.Millisecond)
	
	output := buf.String()
	if !strings.Contains(output, "Operation interrupted") {
		t.Error("output should contain interruption message")
	}
	if !strings.Contains(output, "%") {
		t.Error("output should contain percentage")
	}
	if !strings.Contains(output, "Written:") {
		t.Error("output should contain written bytes information")
	}
}

func TestProgressReporterShutdown(t *testing.T) {
	var buf bytes.Buffer
	ctx := context.Background()
	handler := NewShutdownHandler(ctx, &buf)
	
	// Create a mock progress reporter
	progressReporter := progress.NewProgressReporter(1024, true, &buf)
	handler.SetProgressReporter(progressReporter)
	
	// Start progress reporter
	written := int64(0)
	getWritten := func() int64 {
		return atomic.LoadInt64(&written)
	}
	progressReporter.Start(getWritten)
	
	// Verify it's running
	if !progressReporter.IsRunning() {
		t.Error("progress reporter should be running")
	}
	
	// Trigger shutdown
	handler.Stop()
	time.Sleep(100 * time.Millisecond)
	
	// Progress reporter should be stopped
	if progressReporter.IsRunning() {
		t.Error("progress reporter should be stopped after shutdown")
	}
}

func TestSignalHandling(t *testing.T) {
	// This test is more complex as it involves actual signal handling
	// We'll test the signal setup rather than sending actual signals
	var buf bytes.Buffer
	ctx := context.Background()
	handler := NewShutdownHandler(ctx, &buf)
	
	// Start signal handling
	handler.Start()
	
	// Verify that the signal channel is set up
	if handler.sigChan == nil {
		t.Error("signal channel should be initialized")
	}
	
	// We can't easily test actual signal reception in a unit test,
	// but we can verify the infrastructure is in place
}

func TestContextCancellation(t *testing.T) {
	var buf bytes.Buffer
	parentCtx, parentCancel := context.WithCancel(context.Background())
	handler := NewShutdownHandler(parentCtx, &buf)
	
	var cleanupCalled bool
	handler.RegisterCleanupFunc(func() error {
		cleanupCalled = true
		return nil
	})
	
	handler.Start()
	
	// Cancel the parent context
	parentCancel()
	
	// Wait for shutdown to complete
	time.Sleep(100 * time.Millisecond)
	
	if !handler.IsShutdown() {
		t.Error("handler should be shutdown when parent context is cancelled")
	}
	
	if !cleanupCalled {
		t.Error("cleanup function should be called when context is cancelled")
	}
}

func TestConcurrentShutdown(t *testing.T) {
	var buf bytes.Buffer
	ctx := context.Background()
	handler := NewShutdownHandler(ctx, &buf)
	
	var cleanupCount int32
	handler.RegisterCleanupFunc(func() error {
		atomic.AddInt32(&cleanupCount, 1)
		time.Sleep(10 * time.Millisecond) // Simulate some cleanup work
		return nil
	})
	
	// Start multiple goroutines that try to trigger shutdown
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			handler.Stop()
		}()
	}
	
	wg.Wait()
	time.Sleep(100 * time.Millisecond)
	
	// Cleanup should only be called once despite multiple Stop() calls
	if atomic.LoadInt32(&cleanupCount) != 1 {
		t.Errorf("cleanup should be called exactly once, got %d", cleanupCount)
	}
}

func TestWithShutdownHandler(t *testing.T) {
	var buf bytes.Buffer
	ctx, handler := WithShutdownHandler(&buf)
	
	if ctx == nil {
		t.Error("context should not be nil")
	}
	if handler == nil {
		t.Error("handler should not be nil")
	}
	
	// Context should not be cancelled initially
	select {
	case <-ctx.Done():
		t.Error("context should not be cancelled initially")
	default:
		// Expected
	}
	
	// Trigger shutdown
	handler.Stop()
	time.Sleep(50 * time.Millisecond)
	
	// Context should be cancelled after shutdown
	select {
	case <-ctx.Done():
		// Expected
	default:
		t.Error("context should be cancelled after shutdown")
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.00 KB"},
		{1024 * 1024, "1.00 MB"},
		{1024 * 1024 * 1024, "1.00 GB"},
		{int64(1024) * 1024 * 1024 * 1024, "1.00 TB"},
		{1536, "1.50 KB"},
		{int64(2.5 * 1024 * 1024 * 1024), "2.50 GB"},
	}

	for _, test := range tests {
		result := formatBytes(test.bytes)
		if result != test.expected {
			t.Errorf("formatBytes(%d) = %s, expected %s", 
				test.bytes, result, test.expected)
		}
	}
}

func TestWaitForShutdown(t *testing.T) {
	// Test with nil handler (should not panic)
	done := make(chan bool)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("WaitForShutdown with nil handler should not panic: %v", r)
			}
			done <- true
		}()
		
		// This would normally block, but we'll use a timeout
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()
		
		go func() {
			WaitForShutdown(nil)
		}()
		
		<-ctx.Done()
	}()
	
	select {
	case <-done:
		// Test completed
	case <-time.After(100 * time.Millisecond):
		t.Error("test timed out")
	}
}

// Mock error type for testing
type testError struct {
	message string
}

func (e *testError) Error() string {
	return e.message
}

// Test helper functions
func TestZeroWrittenBytes(t *testing.T) {
	var buf bytes.Buffer
	ctx := context.Background()
	handler := NewShutdownHandler(ctx, &buf)
	
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "empty.bin")
	
	// Create a file writer but don't write any data
	fileWriter, err := writer.NewFileWriter(testFile, 1000, false)
	if err != nil {
		t.Fatalf("failed to create file writer: %v", err)
	}
	defer fileWriter.Close()
	
	handler.SetWriter(fileWriter)
	handler.Stop()
	time.Sleep(50 * time.Millisecond)
	
	output := buf.String()
	if !strings.Contains(output, "0.00%") && !strings.Contains(output, "Written: 0 B") {
		t.Errorf("output should indicate no data was written, got: %s", output)
	}
}

// Benchmark tests
func BenchmarkShutdownHandler(b *testing.B) {
	var buf bytes.Buffer
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := context.Background()
		handler := NewShutdownHandler(ctx, &buf)
		
		// Register a simple cleanup function
		handler.RegisterCleanupFunc(func() error {
			return nil
		})
		
		handler.Stop()
	}
}

func BenchmarkCleanupFunctions(b *testing.B) {
	var buf bytes.Buffer
	ctx := context.Background()
	handler := NewShutdownHandler(ctx, &buf)
	
	// Register multiple cleanup functions
	for i := 0; i < 100; i++ {
		handler.RegisterCleanupFunc(func() error {
			return nil
		})
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Reset the handler for each iteration
		handler.shutdownOnce = sync.Once{}
		handler.isShutdown = false
		handler.Stop()
	}
}