// Package downloader handles file downloads with progress tracking and retry logic.
package downloader

import (
	"context"
	"fmt"
	"io"
	"math"
	"math/rand/v2"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/nchapman/myrient-dl/internal/parser"
	"github.com/schollz/progressbar/v3"
)

// Config holds the downloader configuration
type Config struct {
	OutputDir     string
	Parallel      int
	RetryAttempts int
	Verbose       bool
}

// Downloader manages file downloads
type Downloader struct {
	config Config
	client *http.Client
}

// New creates a new Downloader with the given config
func New(config Config) *Downloader {
	return &Downloader{
		config: config,
		client: &http.Client{
			Timeout: 30 * time.Minute, // Long timeout for large files
		},
	}
}

// DownloadAll downloads all files with progress tracking
func (d *Downloader) DownloadAll(ctx context.Context, files []parser.FileInfo) error {
	total := len(files)

	if d.config.Parallel == 1 {
		// Serial downloads with detailed progress
		for i, file := range files {
			fmt.Printf("\n[%d/%d] Downloading: %s\n", i+1, total, file.Name)

			if err := d.downloadFileWithRetry(ctx, file); err != nil {
				return fmt.Errorf("failed to download %s: %w", file.Name, err)
			}
		}
	} else {
		// Parallel downloads
		return d.downloadParallel(ctx, files)
	}

	return nil
}

// downloadFileWithRetry downloads a single file with retry logic using exponential backoff with jitter
func (d *Downloader) downloadFileWithRetry(ctx context.Context, file parser.FileInfo) error {
	var lastErr error

	for attempt := 1; attempt <= d.config.RetryAttempts; attempt++ {
		err := d.downloadFile(ctx, file)
		if err == nil {
			return nil
		}

		lastErr = err
		if attempt < d.config.RetryAttempts {
			// Exponential backoff with jitter
			// Base delay: 1s, exponentially increases with each attempt
			// Jitter: ±25% randomization to prevent thundering herd
			baseDelay := time.Second * time.Duration(math.Pow(2, float64(attempt-1)))
			jitter := time.Duration(float64(baseDelay) * 0.25 * (2*rand.Float64() - 1)) //nolint:gosec // Non-cryptographic random for backoff jitter is acceptable
			backoff := baseDelay + jitter

			// Cap at 30 seconds
			if backoff > 30*time.Second {
				backoff = 30 * time.Second
			}

			fmt.Printf("  ⚠ Attempt %d failed, retrying in %v...\n", attempt, backoff.Round(time.Millisecond))

			// Wait with context support
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	return fmt.Errorf("failed after %d attempts: %w", d.config.RetryAttempts, lastErr)
}

// downloadFile downloads a single file with progress bar
func (d *Downloader) downloadFile(ctx context.Context, file parser.FileInfo) error {
	outputPath := filepath.Join(d.config.OutputDir, file.Name)

	// Get the actual file size from the server
	actualSize, err := d.getRemoteFileSize(ctx, file.URL)
	if err != nil {
		return fmt.Errorf("failed to get file size: %w", err)
	}

	// Check if file already exists with the correct size
	if info, err := os.Stat(outputPath); err == nil {
		if info.Size() == actualSize {
			fmt.Printf("  ✓ Already downloaded (skipping)\n")
			return nil
		}
		if d.config.Verbose {
			fmt.Printf("  ⚠ File exists but size mismatch (local: %d, remote: %d), re-downloading\n",
				info.Size(), actualSize)
		}
	}

	// Create the request with context
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, file.URL, nil)
	if err != nil {
		return err
	}

	// Set User-Agent for polite web scraping
	req.Header.Set("User-Agent", "myrient-dl/1.0 (https://github.com/nchapman/myrient-dl)")

	resp, err := d.client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	// Create temp file for atomic write
	tempPath := outputPath + ".tmp"
	out, err := os.Create(tempPath) //nolint:gosec // File path is controlled by config and filename from server
	if err != nil {
		return err
	}
	defer func() {
		_ = out.Close()
		// Clean up temp file if it still exists
		_ = os.Remove(tempPath)
	}()

	// Create progress bar
	bar := progressbar.DefaultBytes(
		resp.ContentLength,
		"  downloading",
	)

	// Copy with progress tracking
	_, err = io.Copy(io.MultiWriter(out, bar), resp.Body)
	if err != nil {
		return err
	}

	// Close before rename
	if err := out.Close(); err != nil {
		return err
	}

	// Atomic rename
	if err := os.Rename(tempPath, outputPath); err != nil {
		return err
	}

	fmt.Println() // New line after progress bar
	return nil
}

// getRemoteFileSize makes a HEAD request to get the actual file size from the server
func (d *Downloader) getRemoteFileSize(ctx context.Context, url string) (int64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return 0, err
	}

	// Set User-Agent for polite web scraping
	req.Header.Set("User-Agent", "myrient-dl/1.0 (https://github.com/nchapman/myrient-dl)")

	resp, err := d.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	return resp.ContentLength, nil
}

// downloadParallel downloads files in parallel
func (d *Downloader) downloadParallel(ctx context.Context, files []parser.FileInfo) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var (
		wg        sync.WaitGroup
		errCh     = make(chan error, len(files))
		semaphore = make(chan struct{}, d.config.Parallel)
	)

	total := len(files)
	completed := 0
	var mu sync.Mutex

	for _, file := range files {
		wg.Add(1)
		go func(f parser.FileInfo) {
			defer wg.Done()

			// Check if context is cancelled
			select {
			case <-ctx.Done():
				return
			default:
			}

			// Acquire semaphore
			select {
			case semaphore <- struct{}{}:
			case <-ctx.Done():
				return
			}
			defer func() { <-semaphore }()

			mu.Lock()
			current := completed + 1
			mu.Unlock()

			fmt.Printf("\n[%d/%d] Downloading: %s\n", current, total, f.Name)

			if err := d.downloadFileWithRetry(ctx, f); err != nil {
				errCh <- fmt.Errorf("failed to download %s: %w", f.Name, err)
				cancel() // Cancel all other downloads on first error
				return
			}

			mu.Lock()
			completed++
			mu.Unlock()
		}(file)
	}

	// Wait for all downloads to complete
	wg.Wait()
	close(errCh)

	// Collect all errors
	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}

	// Return first error if any
	if len(errs) > 0 {
		if len(errs) == 1 {
			return errs[0]
		}
		// Multiple errors - return first with count
		return fmt.Errorf("%w (and %d other error(s))", errs[0], len(errs)-1)
	}

	return nil
}
