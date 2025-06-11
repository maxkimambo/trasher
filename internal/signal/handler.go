package signal

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/maxkimambo/trasher/internal/progress"
	"github.com/maxkimambo/trasher/internal/writer"
)

// CleanupFunc represents a cleanup function that can return an error.
type CleanupFunc func() error

// ShutdownHandler manages graceful shutdown on signal reception.
type ShutdownHandler struct {
	ctx          context.Context
	cancel       context.CancelFunc
	sigChan      chan os.Signal
	cleanupFns   []CleanupFunc
	writer       *writer.FileWriter
	progress     *progress.ProgressReporter
	output       io.Writer
	mu           sync.Mutex
	shutdownOnce sync.Once
	isShutdown   bool
}

// NewShutdownHandler creates a new shutdown handler.
func NewShutdownHandler(ctx context.Context, output io.Writer) *ShutdownHandler {
	if output == nil {
		output = os.Stdout
	}

	ctx, cancel := context.WithCancel(ctx)
	return &ShutdownHandler{
		ctx:     ctx,
		cancel:  cancel,
		sigChan: make(chan os.Signal, 1),
		output:  output,
	}
}

// SetWriter sets the file writer for progress reporting during shutdown.
func (h *ShutdownHandler) SetWriter(writer *writer.FileWriter) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.writer = writer
}

// SetProgressReporter sets the progress reporter for shutdown handling.
func (h *ShutdownHandler) SetProgressReporter(progress *progress.ProgressReporter) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.progress = progress
}

// RegisterCleanupFunc adds a cleanup function to be called during shutdown.
// Cleanup functions are called in reverse order (LIFO).
func (h *ShutdownHandler) RegisterCleanupFunc(fn CleanupFunc) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.cleanupFns = append(h.cleanupFns, fn)
}

// Start begins monitoring for shutdown signals.
// This should be called once at the beginning of the application.
func (h *ShutdownHandler) Start() {
	// Register signals to capture
	signal.Notify(h.sigChan, syscall.SIGINT, syscall.SIGTERM)

	go h.signalLoop()
}

// signalLoop runs in a goroutine and waits for signals or context cancellation.
func (h *ShutdownHandler) signalLoop() {
	select {
	case sig := <-h.sigChan:
		fmt.Fprintf(h.output, "\nReceived signal %v, shutting down gracefully...\n", sig)
		h.initiateShutdown()
	case <-h.ctx.Done():
		// Context was cancelled elsewhere, perform cleanup
		h.initiateShutdown()
	}
}

// initiateShutdown starts the graceful shutdown process.
func (h *ShutdownHandler) initiateShutdown() {
	h.shutdownOnce.Do(func() {
		h.mu.Lock()
		h.isShutdown = true
		h.mu.Unlock()

		// Cancel context to signal all components to stop
		h.cancel()

		// Perform cleanup
		h.performCleanup()
	})
}

// performCleanup executes all registered cleanup functions and reports progress.
func (h *ShutdownHandler) performCleanup() {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Stop progress reporting first
	if h.progress != nil {
		h.progress.Stop()
	}

	// Report partial progress if we have a writer
	if h.writer != nil {
		h.reportPartialProgress()
	}

	// Execute cleanup functions in reverse order (LIFO)
	if len(h.cleanupFns) > 0 {
		fmt.Fprintf(h.output, "Cleaning up resources...\n")
		
		for i := len(h.cleanupFns) - 1; i >= 0; i-- {
			if err := h.cleanupFns[i](); err != nil {
				fmt.Fprintf(h.output, "Warning: cleanup error: %v\n", err)
			}
		}
		
		fmt.Fprintf(h.output, "Cleanup completed.\n")
	}
}

// reportPartialProgress reports the current progress when interrupted.
func (h *ShutdownHandler) reportPartialProgress() {
	written := h.writer.Written()
	total := h.writer.TotalSize()
	
	if total > 0 {
		percent := float64(written) / float64(total) * 100
		fmt.Fprintf(h.output, "Operation interrupted at %.2f%% completion\n", percent)
		fmt.Fprintf(h.output, "Written: %s / %s\n", 
			formatBytes(written), formatBytes(total))
		
		if written > 0 {
			fmt.Fprintf(h.output, "Partial file saved to: %s\n", h.writer.Path())
		}
	} else {
		fmt.Fprintf(h.output, "Operation interrupted before any data was written\n")
	}
}

// Context returns the context that will be cancelled on shutdown.
func (h *ShutdownHandler) Context() context.Context {
	return h.ctx
}

// IsShutdown returns true if shutdown has been initiated.
func (h *ShutdownHandler) IsShutdown() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.isShutdown
}

// Stop manually triggers shutdown (useful for testing or programmatic shutdown).
func (h *ShutdownHandler) Stop() {
	h.initiateShutdown()
}

// Wait blocks until shutdown is complete.
func (h *ShutdownHandler) Wait() {
	<-h.ctx.Done()
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

// WithShutdownHandler is a convenience function that creates a shutdown handler
// and returns a context that will be cancelled on shutdown signals.
func WithShutdownHandler(output io.Writer) (context.Context, *ShutdownHandler) {
	ctx := context.Background()
	handler := NewShutdownHandler(ctx, output)
	handler.Start()
	return handler.Context(), handler
}

// WaitForShutdown blocks until a shutdown signal is received.
func WaitForShutdown(handler *ShutdownHandler) {
	if handler != nil {
		handler.Wait()
	} else {
		// Fallback: just wait for signals directly
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
	}
}