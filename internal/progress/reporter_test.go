package progress

import (
	"bytes"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewProgressReporter(t *testing.T) {
	// Test with small file (< 1GB), no verbose
	smallFile := int64(1024 * 1024) // 1MB
	pr := NewProgressReporter(smallFile, false, nil)
	if pr.ShouldShowProgress() {
		t.Error("small file without verbose should not show progress")
	}

	// Test with small file, verbose
	pr = NewProgressReporter(smallFile, true, nil)
	if !pr.ShouldShowProgress() {
		t.Error("small file with verbose should show progress")
	}

	// Test with large file (>= 1GB), no verbose
	largeFile := int64(2 * 1024 * 1024 * 1024) // 2GB
	pr = NewProgressReporter(largeFile, false, nil)
	if !pr.ShouldShowProgress() {
		t.Error("large file should show progress even without verbose")
	}

	// Test with large file, verbose
	pr = NewProgressReporter(largeFile, true, nil)
	if !pr.ShouldShowProgress() {
		t.Error("large file with verbose should show progress")
	}
}

func TestProgressReporterBasic(t *testing.T) {
	var buf bytes.Buffer
	totalSize := int64(2 * 1024 * 1024 * 1024) // 2GB
	pr := NewProgressReporter(totalSize, false, &buf)

	var written int64
	getWritten := func() int64 {
		return atomic.LoadInt64(&written)
	}

	pr.Start(getWritten)
	if !pr.IsRunning() {
		t.Error("progress reporter should be running after Start()")
	}

	// Simulate some progress
	atomic.StoreInt64(&written, totalSize/4) // 25%
	time.Sleep(150 * time.Millisecond)       // Give time for update

	// Stop and check final output
	pr.Stop()
	if pr.IsRunning() {
		t.Error("progress reporter should not be running after Stop()")
	}

	output := buf.String()
	if len(output) == 0 {
		t.Error("expected some output from progress reporter")
	}

	// Should contain progress elements
	if !strings.Contains(output, "%") {
		t.Error("output should contain percentage")
	}
}

func TestProgressReporterVerbose(t *testing.T) {
	var buf bytes.Buffer
	totalSize := int64(1024 * 1024) // 1MB (small file, but verbose)
	pr := NewProgressReporter(totalSize, true, &buf)

	var written int64
	getWritten := func() int64 {
		return atomic.LoadInt64(&written)
	}

	pr.Start(getWritten)

	// Simulate progress
	atomic.StoreInt64(&written, totalSize/2) // 50%
	time.Sleep(150 * time.Millisecond)

	pr.Stop()

	output := buf.String()
	if len(output) == 0 {
		t.Error("verbose mode should produce output even for small files")
	}

	// Verbose mode should show more details
	if !strings.Contains(output, "Written:") || !strings.Contains(output, "Elapsed:") {
		t.Error("verbose mode should show detailed information")
	}
}

func TestProgressReporterNoProgress(t *testing.T) {
	var buf bytes.Buffer
	totalSize := int64(1024 * 1024) // 1MB (small file, no verbose)
	pr := NewProgressReporter(totalSize, false, &buf)

	var written int64
	getWritten := func() int64 {
		return atomic.LoadInt64(&written)
	}

	pr.Start(getWritten)
	atomic.StoreInt64(&written, totalSize/2)
	time.Sleep(150 * time.Millisecond)
	pr.Stop()

	output := buf.String()
	// Should have minimal or no output for small files without verbose
	if strings.Contains(output, "[=") {
		t.Error("small file without verbose should not show progress bar")
	}
}

func TestFormatThroughput(t *testing.T) {
	tests := []struct {
		bytesPerSec float64
		expected    string
	}{
		{0, "0 B/s"},
		{512, "512 B/s"},
		{1024, "1.00 KB/s"},
		{1024 * 1024, "1.00 MB/s"},
		{1024 * 1024 * 1024, "1.00 GB/s"},
		{1.5 * 1024 * 1024, "1.50 MB/s"},
		{2.25 * 1024 * 1024 * 1024, "2.25 GB/s"},
	}

	for _, test := range tests {
		result := formatThroughput(test.bytesPerSec)
		if result != test.expected {
			t.Errorf("formatThroughput(%.0f) = %s, expected %s", 
				test.bytesPerSec, result, test.expected)
		}
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
		{1536, "1.50 KB"},                                    // 1.5 KB
		{int64(2.5 * 1024 * 1024 * 1024), "2.50 GB"},       // 2.5 GB
	}

	for _, test := range tests {
		result := formatBytes(test.bytes)
		if result != test.expected {
			t.Errorf("formatBytes(%d) = %s, expected %s", 
				test.bytes, result, test.expected)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{0, "0s"},
		{-time.Second, "∞"},
		{30 * time.Second, "30s"},
		{90 * time.Second, "1m30s"},
		{2 * time.Minute, "2m"},
		{65 * time.Minute, "1h5m"},
		{2 * time.Hour, "2h"},
		{25 * time.Hour, "1d1h"},
		{48 * time.Hour, "2d"},
	}

	for _, test := range tests {
		result := formatDuration(test.duration)
		if result != test.expected {
			t.Errorf("formatDuration(%v) = %s, expected %s", 
				test.duration, result, test.expected)
		}
	}
}

func TestFormatProgressBar(t *testing.T) {
	var buf bytes.Buffer
	pr := NewProgressReporter(1000, false, &buf)

	tests := []struct {
		percent  float64
		width    int
		contains string
	}{
		{0, 10, "[>"},
		{50, 10, "[=====>"},
		{100, 10, "[==========]"},
		{25, 20, "[=====>"},
	}

	for _, test := range tests {
		result := pr.formatProgressBar(test.percent, test.width)
		if !strings.Contains(result, test.contains) {
			t.Errorf("formatProgressBar(%.1f, %d) = %s, should contain %s", 
				test.percent, test.width, result, test.contains)
		}
		// Check that result has correct brackets
		if !strings.HasPrefix(result, "[") || !strings.HasSuffix(result, "]") {
			t.Errorf("progress bar should be enclosed in brackets: %s", result)
		}
	}
}

func TestProgressReporterConcurrency(t *testing.T) {
	var buf bytes.Buffer
	totalSize := int64(2 * 1024 * 1024 * 1024) // 2GB
	pr := NewProgressReporter(totalSize, false, &buf)

	var written int64
	getWritten := func() int64 {
		return atomic.LoadInt64(&written)
	}

	// Start multiple goroutines that call Start/Stop
	for i := 0; i < 10; i++ {
		go func() {
			pr.Start(getWritten)
			time.Sleep(10 * time.Millisecond)
			pr.Stop()
		}()
	}

	// Update written value concurrently
	go func() {
		for j := int64(0); j < totalSize; j += totalSize / 100 {
			atomic.StoreInt64(&written, j)
			time.Sleep(time.Millisecond)
		}
	}()

	time.Sleep(200 * time.Millisecond)
	pr.Stop()

	// Test should complete without data races or panics
}

func TestProgressReporterEdgeCases(t *testing.T) {
	// Test with zero total size
	var buf bytes.Buffer
	pr := NewProgressReporter(0, true, &buf)
	
	getWritten := func() int64 { return 0 }
	pr.Start(getWritten)
	time.Sleep(50 * time.Millisecond)
	pr.Stop()

	// Test with written > total (shouldn't panic)
	buf.Reset()
	pr = NewProgressReporter(100, true, &buf)
	
	getWritten = func() int64 { return 150 } // More than total
	pr.Start(getWritten)
	time.Sleep(50 * time.Millisecond)
	pr.Stop()

	output := buf.String()
	if strings.Contains(output, "101.") || strings.Contains(output, "150.") {
		t.Error("progress should be capped at 100%")
	}
}

func TestProgressReporterDoubleStartStop(t *testing.T) {
	var buf bytes.Buffer
	totalSize := int64(1024 * 1024 * 1024) // 1GB
	pr := NewProgressReporter(totalSize, false, &buf)

	getWritten := func() int64 { return 0 }

	// Test double start
	pr.Start(getWritten)
	if !pr.IsRunning() {
		t.Error("should be running after first Start()")
	}
	
	pr.Start(getWritten) // Second start should be ignored
	if !pr.IsRunning() {
		t.Error("should still be running after second Start()")
	}

	// Test double stop
	pr.Stop()
	if pr.IsRunning() {
		t.Error("should not be running after first Stop()")
	}
	
	pr.Stop() // Second stop should not cause issues
	if pr.IsRunning() {
		t.Error("should still not be running after second Stop()")
	}
}

func TestProgressReporterNilWriter(t *testing.T) {
	// Test with nil writer (should not panic)
	pr := NewProgressReporter(1024*1024*1024, true, nil)
	
	getWritten := func() int64 { return 0 }
	pr.Start(getWritten)
	time.Sleep(50 * time.Millisecond)
	pr.Stop()
	
	// Should complete without panicking
}

// Benchmark tests
func BenchmarkProgressUpdate(b *testing.B) {
	var buf bytes.Buffer
	totalSize := int64(1024 * 1024 * 1024) // 1GB
	pr := NewProgressReporter(totalSize, false, &buf)

	getWritten := func() int64 { return totalSize / 2 }

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pr.update(getWritten())
	}
}

func BenchmarkFormatThroughput(b *testing.B) {
	throughput := 150.5 * 1024 * 1024 // 150.5 MB/s
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		formatThroughput(throughput)
	}
}

func BenchmarkFormatProgressBar(b *testing.B) {
	var buf bytes.Buffer
	pr := NewProgressReporter(1000, false, &buf)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pr.formatProgressBar(45.5, 40)
	}
}