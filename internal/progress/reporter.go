package progress

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

// ProgressReporter provides real-time progress reporting for file generation operations.
type ProgressReporter struct {
	totalSize     int64
	startTime     time.Time
	lastUpdate    time.Time
	lastWritten   int64
	verbose       bool
	done          chan struct{}
	writer        io.Writer
	mu            sync.Mutex
	running       bool
	showProgress  bool
}

// NewProgressReporter creates a new progress reporter.
// Progress is shown for files > 1GB or when verbose is true.
func NewProgressReporter(totalSize int64, verbose bool, writer io.Writer) *ProgressReporter {
	if writer == nil {
		writer = io.Discard
	}

	return &ProgressReporter{
		totalSize:    totalSize,
		verbose:      verbose,
		done:         make(chan struct{}),
		writer:       writer,
		showProgress: totalSize >= (1<<30) || verbose, // Show for files >= 1GB or verbose mode
	}
}

// Start begins progress reporting in a separate goroutine.
// getWrittenFunc should return the current number of bytes written.
func (p *ProgressReporter) Start(getWrittenFunc func() int64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		return
	}

	p.running = true
	p.startTime = time.Now()
	p.lastUpdate = p.startTime

	if !p.showProgress {
		return
	}

	go p.progressLoop(getWrittenFunc)
}

// progressLoop runs the progress reporting loop.
func (p *ProgressReporter) progressLoop(getWrittenFunc func() int64) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-p.done:
			p.printFinalStats(getWrittenFunc())
			return
		case <-ticker.C:
			p.update(getWrittenFunc())
		}
	}
}

// update refreshes the progress display.
func (p *ProgressReporter) update(written int64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(p.startTime)
	sinceLast := now.Sub(p.lastUpdate)

	// Skip update if too little time has passed
	if sinceLast < 50*time.Millisecond {
		return
	}

	// Calculate progress percentage
	percent := float64(written) / float64(p.totalSize) * 100
	if percent > 100 {
		percent = 100
	}

	// Calculate current throughput
	bytesWrittenSinceLast := written - p.lastWritten
	var throughput float64
	if sinceLast > 0 {
		throughput = float64(bytesWrittenSinceLast) / sinceLast.Seconds()
	}

	// Calculate ETA using recent throughput for better responsiveness
	var eta time.Duration
	if written > 0 && written < p.totalSize && throughput > 0 {
		remainingBytes := p.totalSize - written
		eta = time.Duration(float64(remainingBytes)/throughput) * time.Second
	}

	// Format output
	p.printProgress(percent, throughput, eta, elapsed, written)

	// Update last values
	p.lastUpdate = now
	p.lastWritten = written
}

// printProgress displays the current progress.
func (p *ProgressReporter) printProgress(percent float64, throughput float64, eta, elapsed time.Duration, written int64) {
	// Format throughput
	throughputStr := formatThroughput(throughput)

	if p.verbose {
		// Verbose mode: show detailed information
		fmt.Fprintf(p.writer, "\r%s | %.2f%% | %s | ETA: %s | Elapsed: %s | Written: %s / %s",
			p.formatProgressBar(percent, 30),
			percent,
			throughputStr,
			formatDuration(eta),
			formatDuration(elapsed),
			formatBytes(written),
			formatBytes(p.totalSize))
	} else {
		// Standard mode: show compact progress
		fmt.Fprintf(p.writer, "\r%s %.2f%% | %s | ETA: %s",
			p.formatProgressBar(percent, 40),
			percent,
			throughputStr,
			formatDuration(eta))
	}
}

// formatProgressBar creates a visual progress bar.
func (p *ProgressReporter) formatProgressBar(percent float64, width int) string {
	completed := int(float64(width) * percent / 100.0)
	if completed > width {
		completed = width
	}
	if completed < 0 {
		completed = 0
	}

	bar := strings.Repeat("=", completed)
	if completed < width {
		bar += ">"
		bar += strings.Repeat(" ", width-completed-1)
	}

	return fmt.Sprintf("[%s]", bar)
}

// Stop stops the progress reporting and prints final statistics.
func (p *ProgressReporter) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return
	}

	p.running = false
	select {
	case <-p.done:
		// Already closed
	default:
		close(p.done)
	}
}

// printFinalStats prints the final completion statistics.
func (p *ProgressReporter) printFinalStats(written int64) {
	if !p.showProgress {
		return
	}

	elapsed := time.Since(p.startTime)
	var avgThroughput float64
	if elapsed > 0 {
		avgThroughput = float64(written) / elapsed.Seconds()
	}

	throughputStr := formatThroughput(avgThroughput)

	// Clear the progress line and print final stats
	fmt.Fprintf(p.writer, "\r%s\n", strings.Repeat(" ", 80)) // Clear line
	fmt.Fprintf(p.writer, "Completed %s in %s (average %s)\n",
		formatBytes(written),
		formatDuration(elapsed),
		throughputStr)
}

// formatThroughput formats throughput in appropriate units.
func formatThroughput(bytesPerSecond float64) string {
	if bytesPerSecond == 0 {
		return "0 B/s"
	}

	if bytesPerSecond >= 1<<30 { // >= 1 GB/s
		return fmt.Sprintf("%.2f GB/s", bytesPerSecond/float64(1<<30))
	} else if bytesPerSecond >= 1<<20 { // >= 1 MB/s
		return fmt.Sprintf("%.2f MB/s", bytesPerSecond/float64(1<<20))
	} else if bytesPerSecond >= 1<<10 { // >= 1 KB/s
		return fmt.Sprintf("%.2f KB/s", bytesPerSecond/float64(1<<10))
	} else {
		return fmt.Sprintf("%.0f B/s", bytesPerSecond)
	}
}

// formatBytes formats byte count in human-readable format.
func formatBytes(bytes int64) string {
	if bytes == 0 {
		return "0 B"
	}

	if bytes >= 1<<40 { // >= 1 TB
		return fmt.Sprintf("%.2f TB", float64(bytes)/float64(1<<40))
	} else if bytes >= 1<<30 { // >= 1 GB
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(1<<30))
	} else if bytes >= 1<<20 { // >= 1 MB
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(1<<20))
	} else if bytes >= 1<<10 { // >= 1 KB
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(1<<10))
	} else {
		return fmt.Sprintf("%d B", bytes)
	}
}

// formatDuration formats duration in human-readable format.
func formatDuration(d time.Duration) string {
	if d == 0 {
		return "0s"
	}

	if d < 0 {
		return "∞"
	}

	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	} else if d < time.Hour {
		m := d / time.Minute
		s := (d % time.Minute) / time.Second
		if s == 0 {
			return fmt.Sprintf("%dm", m)
		}
		return fmt.Sprintf("%dm%ds", m, s)
	} else if d < 24*time.Hour {
		h := d / time.Hour
		m := (d % time.Hour) / time.Minute
		if m == 0 {
			return fmt.Sprintf("%dh", h)
		}
		return fmt.Sprintf("%dh%dm", h, m)
	} else {
		days := d / (24 * time.Hour)
		h := (d % (24 * time.Hour)) / time.Hour
		if h == 0 {
			return fmt.Sprintf("%dd", days)
		}
		return fmt.Sprintf("%dd%dh", days, h)
	}
}

// IsRunning returns whether the progress reporter is currently running.
func (p *ProgressReporter) IsRunning() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.running
}

// ShouldShowProgress returns whether progress should be displayed.
func (p *ProgressReporter) ShouldShowProgress() bool {
	return p.showProgress
}