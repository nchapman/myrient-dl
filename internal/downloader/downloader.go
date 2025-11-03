// Package downloader handles file downloads with progress tracking and retry logic.
package downloader

import (
	"fmt"
	"io"
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
func (d *Downloader) DownloadAll(files []parser.FileInfo) error {
	total := len(files)

	if d.config.Parallel == 1 {
		// Serial downloads with detailed progress
		for i, file := range files {
			fmt.Printf("\n[%d/%d] Downloading: %s\n", i+1, total, file.Name)

			if err := d.downloadFileWithRetry(file); err != nil {
				return fmt.Errorf("failed to download %s: %w", file.Name, err)
			}
		}
	} else {
		// Parallel downloads
		return d.downloadParallel(files)
	}

	return nil
}

// downloadFileWithRetry downloads a single file with retry logic
func (d *Downloader) downloadFileWithRetry(file parser.FileInfo) error {
	var lastErr error

	for attempt := 1; attempt <= d.config.RetryAttempts; attempt++ {
		err := d.downloadFile(file)
		if err == nil {
			return nil
		}

		lastErr = err
		if attempt < d.config.RetryAttempts {
			fmt.Printf("  ⚠ Attempt %d failed, retrying...\n", attempt)
			time.Sleep(time.Second * time.Duration(attempt)) // Exponential backoff
		}
	}

	return fmt.Errorf("failed after %d attempts: %w", d.config.RetryAttempts, lastErr)
}

// downloadFile downloads a single file with progress bar
func (d *Downloader) downloadFile(file parser.FileInfo) error {
	outputPath := filepath.Join(d.config.OutputDir, file.Name)

	// Get the actual file size from the server
	actualSize, err := d.getRemoteFileSize(file.URL)
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

	// Create the request
	resp, err := d.client.Get(file.URL)
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	// Create the output file
	out, err := os.Create(outputPath) //nolint:gosec // File path is controlled by config and filename from server
	if err != nil {
		return err
	}
	defer func() {
		_ = out.Close()
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

	fmt.Println() // New line after progress bar
	return nil
}

// getRemoteFileSize makes a HEAD request to get the actual file size from the server
func (d *Downloader) getRemoteFileSize(url string) (int64, error) {
	resp, err := d.client.Head(url)
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
func (d *Downloader) downloadParallel(files []parser.FileInfo) error {
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

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			mu.Lock()
			current := completed + 1
			mu.Unlock()

			fmt.Printf("\n[%d/%d] Downloading: %s\n", current, total, f.Name)

			if err := d.downloadFileWithRetry(f); err != nil {
				errCh <- fmt.Errorf("failed to download %s: %w", f.Name, err)
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

	// Check for errors
	if len(errCh) > 0 {
		return <-errCh
	}

	return nil
}
