package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
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
		if verbose {
			fmt.Println("Running trasher with verbose output")
		}

		// Validate required flags
		if size == "" {
			return fmt.Errorf("--size flag is required")
		}
		if output == "" {
			return fmt.Errorf("--output flag is required")
		}

		// Check if output file exists and force flag is not set
		if !force {
			if _, err := os.Stat(output); err == nil {
				return fmt.Errorf("output file %s already exists, use --force to overwrite", output)
			}
		}

		fmt.Printf("Generating file: %s\n", output)
		fmt.Printf("Size: %s\n", size)
		fmt.Printf("Pattern: %s\n", pattern)
		fmt.Printf("Workers: %d\n", workers)
		fmt.Printf("Chunk size: %s\n", chunkSize)

		// TODO: Implement actual file generation logic
		fmt.Println("File generation logic will be implemented in subsequent tasks")

		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringVarP(&size, "size", "s", "", "Size of file to generate (required)")
	rootCmd.Flags().StringVarP(&pattern, "pattern", "p", "random", "Data pattern to generate (random, zeros, ones, custom)")
	rootCmd.Flags().StringVarP(&output, "output", "o", "", "Output file path (required)")
	rootCmd.Flags().IntVarP(&workers, "workers", "w", 4, "Number of worker goroutines")
	rootCmd.Flags().StringVarP(&chunkSize, "chunk-size", "c", "1MB", "Size of data chunks per worker")
	rootCmd.Flags().BoolVarP(&force, "force", "f", false, "Overwrite existing files without confirmation")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")

	rootCmd.MarkFlagRequired("size")
	rootCmd.MarkFlagRequired("output")
}