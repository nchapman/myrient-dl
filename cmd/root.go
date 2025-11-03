// Package cmd contains the CLI command implementation for myrient-dl.
package cmd

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/nchapman/myrient-dl/internal/downloader"
	"github.com/nchapman/myrient-dl/internal/matcher"
	"github.com/nchapman/myrient-dl/internal/parser"
	"github.com/spf13/cobra"
)

var (
	outputDir      string
	includePattern string
	excludePattern string
	parallel       int
	dryRun         bool
	verbose        bool
	retryAttempts  int
)

var rootCmd = &cobra.Command{
	Use:   "myrient-dl [URL]",
	Short: "Download files from Myrient directory listings",
	Long: `A fast and friendly CLI tool to download files from Myrient.

Downloads files from Myrient directory listings with support for include/exclude patterns,
parallel downloads, and beautiful progress tracking.`,
	Args: cobra.ExactArgs(1),
	RunE: run,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringVarP(&outputDir, "output", "o", "", "Output directory (defaults to last path component of URL)")
	rootCmd.Flags().StringVarP(&includePattern, "include", "i", "*", "Include pattern (glob syntax)")
	rootCmd.Flags().StringVarP(&excludePattern, "exclude", "e", "", "Exclude pattern (glob syntax)")
	rootCmd.Flags().IntVarP(&parallel, "parallel", "p", 1, "Number of parallel downloads")
	rootCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be downloaded without downloading")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
	rootCmd.Flags().IntVarP(&retryAttempts, "retry", "r", 3, "Number of retry attempts for failed downloads")
}

func run(_ *cobra.Command, args []string) error {
	targetURL := args[0]

	// Validate URL
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Determine output directory if not specified
	if outputDir == "" {
		outputDir = getDefaultOutputDir(parsedURL)
	}

	if verbose {
		fmt.Printf("Target URL: %s\n", targetURL)
		fmt.Printf("Output directory: %s\n", outputDir)
		fmt.Printf("Include pattern: %s\n", includePattern)
		if excludePattern != "" {
			fmt.Printf("Exclude pattern: %s\n", excludePattern)
		}
		fmt.Printf("Parallel downloads: %d\n", parallel)
		fmt.Println()
	}

	// Parse directory listing
	fmt.Println("Fetching directory listing...")
	files, err := parser.ParseDirectoryListing(targetURL)
	if err != nil {
		return fmt.Errorf("failed to parse directory listing: %w", err)
	}

	if len(files) == 0 {
		return fmt.Errorf("no files found in directory listing")
	}

	if verbose {
		fmt.Printf("Found %d files\n", len(files))
	}

	// Filter files based on patterns
	m := matcher.New(includePattern, excludePattern)
	filtered := m.Filter(files)

	if len(filtered) == 0 {
		fmt.Println("No files match the specified patterns")
		return nil
	}

	// Calculate total size
	var totalSize int64
	for _, f := range filtered {
		totalSize += f.Size
	}

	fmt.Printf("\nMatched %d files (total size: %s)\n", len(filtered), formatBytes(totalSize))

	if dryRun {
		fmt.Println("\nFiles to download (dry-run mode):")
		for _, f := range filtered {
			fmt.Printf("  - %s (%s)\n", f.Name, formatBytes(f.Size))
		}
		return nil
	}

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil { //nolint:gosec // 0755 is appropriate for download directories
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Download files
	fmt.Println("\nStarting downloads...")
	dl := downloader.New(downloader.Config{
		OutputDir:     outputDir,
		Parallel:      parallel,
		RetryAttempts: retryAttempts,
		Verbose:       verbose,
	})

	if err := dl.DownloadAll(filtered); err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	fmt.Println("\n✓ All downloads completed!")
	return nil
}

// getDefaultOutputDir extracts the last meaningful path component from the URL
func getDefaultOutputDir(u *url.URL) string {
	// Clean the path and remove trailing slashes
	cleanPath := strings.TrimSuffix(u.Path, "/")

	// Get the last component
	lastComponent := path.Base(cleanPath)

	// Decode URL encoding (e.g., %20 -> space)
	decoded, err := url.QueryUnescape(lastComponent)
	if err != nil {
		decoded = lastComponent
	}

	// Sanitize for filesystem
	sanitized := sanitizeFilename(decoded)

	// Fallback if we got nothing useful
	if sanitized == "" || sanitized == "." || sanitized == "/" {
		return "myrient-downloads"
	}

	return "./" + sanitized
}

// sanitizeFilename removes or replaces characters that are problematic for filenames
func sanitizeFilename(name string) string {
	// Replace problematic characters with underscores
	replacer := strings.NewReplacer(
		":", "_",
		"|", "_",
		"<", "_",
		">", "_",
		"\"", "_",
		"?", "_",
		"*", "_",
	)
	return replacer.Replace(name)
}

// formatBytes formats byte sizes in human-readable format
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
