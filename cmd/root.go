package cmd

import (
	"fmt"
	"os"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/spf13/cobra"

	"github.com/maxkimambo/trasher/internal/checksum"
	"github.com/maxkimambo/trasher/internal/progress"
	"github.com/maxkimambo/trasher/internal/signal"
	"github.com/maxkimambo/trasher/internal/validation"
	"github.com/maxkimambo/trasher/internal/worker"
	"github.com/maxkimambo/trasher/internal/writer"
	"github.com/maxkimambo/trasher/pkg/generator"
	"github.com/maxkimambo/trasher/pkg/sizeparser"
)

var (
	size      string
	pattern   string
	output    string
	workers   int
	chunkSize string
	force     bool
	verbose   bool
	version   = "0.1.0"
)

var rootCmd = &cobra.Command{
	Use:   "trasher",
	Short: "A high-performance file generation tool",
	Long: `Trasher is a high-performance file generation tool that creates files of specified sizes
with configurable data patterns using concurrent workers for optimal performance.`,
	Version: version,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runTrasher()
	},
}

func runTrasher() error {
	// Create validation configuration
	config := validation.ValidationConfig{
		Size:       size,
		Pattern:    pattern,
		OutputPath: output,
		Workers:    workers,
		ChunkSize:  chunkSize,
		Force:      force,
	}

	// Run pre-flight validation
	validator := validation.NewValidator()
	if err := validator.ValidateAll(config); err != nil {
		return fmt.Errorf("validation failed: %v", err)
	}

	// Parse size and chunk size
	sizeBytes, err := sizeparser.Parse(size)
	if err != nil {
		return fmt.Errorf("failed to parse size: %v", err)
	}

	chunkSizeBytes, err := sizeparser.Parse(chunkSize)
	if err != nil {
		return fmt.Errorf("failed to parse chunk size: %v", err)
	}

	if verbose {
		fmt.Printf("Generating file: %s\n", output)
		fmt.Printf("Size: %s (%d bytes)\n", size, sizeBytes)
		fmt.Printf("Pattern: %s\n", pattern)
		fmt.Printf("Workers: %d\n", workers)
		fmt.Printf("Chunk size: %s (%d bytes)\n", chunkSize, chunkSizeBytes)
		fmt.Println()
	}

	// Create context and shutdown handler
	ctx, shutdownHandler := signal.WithShutdownHandler(os.Stdout)

	// Create file writer
	fileWriter, err := writer.NewFileWriter(output, sizeBytes, force)
	if err != nil {
		return fmt.Errorf("failed to create file writer: %v", err)
	}

	// Register cleanup for file writer
	shutdownHandler.RegisterCleanupFunc(func() error {
		return fileWriter.Close()
	})

	// Set writer in shutdown handler for progress reporting
	shutdownHandler.SetWriter(fileWriter)

	// Create progress reporter
	progressReporter := progress.NewProgressReporter(sizeBytes, verbose, os.Stdout)
	shutdownHandler.SetProgressReporter(progressReporter)

	// Create pattern generator
	gen, err := generator.NewGenerator(pattern)
	if err != nil {
		return fmt.Errorf("failed to create generator: %v", err)
	}

	// Create checksum generator
	checksumGen := checksum.NewChecksumGenerator(output, sizeBytes)

	// Create worker pool
	workerPool := worker.NewWorkerPool(ctx, workers, chunkSizeBytes)

	// Start progress reporting
	var writtenBytes int64
	getWritten := func() int64 {
		return atomic.LoadInt64(&writtenBytes)
	}
	progressReporter.Start(getWritten)

	// Start worker pool
	workerPool.Start(gen, sizeBytes)

	// Process results
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case result, ok := <-workerPool.Results():
				if !ok {
					// Channel closed, all work completed
					return
				}

				// Update checksum
				if err := checksumGen.UpdateWithChunk(result.Buffer, result.Offset); err != nil {
					fmt.Printf("\nChecksum error: %v\n", err)
					shutdownHandler.Stop()
					workerPool.ReturnBuffer(result.Buffer)
					return
				}

				// Write to file
				if err := fileWriter.WriteAt(result.Buffer, result.Offset); err != nil {
					fmt.Printf("\nFile write error: %v\n", err)
					shutdownHandler.Stop()
					workerPool.ReturnBuffer(result.Buffer)
					return
				}

				// Update written bytes counter
				atomic.AddInt64(&writtenBytes, int64(len(result.Buffer)))

				// Return buffer to pool
				workerPool.ReturnBuffer(result.Buffer)
			}
		}
	}()

	// Monitor for errors in a separate goroutine
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case err, ok := <-workerPool.Errors():
				if !ok {
					// Error channel closed, work completed
					return
				}
				if err != nil {
					fmt.Printf("\nError: %v\n", err)
					shutdownHandler.Stop()
					return
				}
			}
		}
	}()

	// Wait for workers to complete, then close channels
	workerPool.Wait()
	// Wait for result processing to complete
	wg.Wait()

	// Stop progress reporting immediately after work completion
	progressReporter.Stop()

	// Check if operation was cancelled
	select {
	case <-ctx.Done():
		return fmt.Errorf("operation cancelled")
	default:
		// Operation completed successfully
	}

	// Close file writer
	if err := fileWriter.Close(); err != nil {
		return fmt.Errorf("failed to close file: %v", err)
	}

	// Write checksum file
	if err := checksumGen.WriteChecksumFile(); err != nil {
		return fmt.Errorf("failed to write checksum file: %v", err)
	}

	if verbose {
		fmt.Printf("\nFile generation completed successfully!\n")
		fmt.Printf("Output file: %s\n", output)
		fmt.Printf("Checksum file: %s.checksum.txt\n", output)
	} else {
		fmt.Printf("Successfully generated %s\n", output)
	}

	return nil
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringVarP(&size, "size", "s", "", "Size of file to generate (required)")
	rootCmd.Flags().StringVarP(&pattern, "pattern", "p", "random", "Data pattern to generate (random, sequential, zero, mixed)")
	rootCmd.Flags().StringVarP(&output, "output", "o", "", "Output file path (required)")
	rootCmd.Flags().IntVarP(&workers, "workers", "w", runtime.NumCPU(), "Number of worker goroutines")
	rootCmd.Flags().StringVarP(&chunkSize, "chunk-size", "c", "64MB", "Size of data chunks per worker")
	rootCmd.Flags().BoolVarP(&force, "force", "f", false, "Overwrite existing files without confirmation")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")

	rootCmd.MarkFlagRequired("size")
	rootCmd.MarkFlagRequired("output")
}