# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

myrient-dl is a CLI tool for downloading files from Myrient (https://myrient.erista.me/) directory listings. It's built in Go using the Cobra CLI framework and provides features like pattern matching, parallel downloads, progress tracking, and auto-retry.

## Recent Improvements (2025-11-03)

The codebase has been significantly improved with the following enhancements:

### High Priority Improvements
- **Context Support**: Added `context.Context` throughout the codebase for proper cancellation and timeout support
- **Atomic File Writes**: Downloads now write to temporary files and atomically rename on success, preventing corrupted partial files
- **Improved Error Handling**: Parallel downloads now properly cancel on errors and report multiple failures
- **Enhanced Path Sanitization**: Strengthened file path sanitization to prevent directory traversal, null bytes, and hidden files
- **Graceful Shutdown**: Added signal handling (SIGINT/SIGTERM) for clean cancellation of downloads
- **User-Agent Header**: Added proper User-Agent to all HTTP requests for web scraping etiquette

### Medium Priority Improvements
- **Better Retry Backoff**: Implemented exponential backoff with jitter (instead of linear) for retries
- **Build Version Info**: Added version, commit, and build time information via ldflags
- **Dependency Management**: Added dependabot.yml for automated dependency updates
- **Linting**: Enabled gosec linter for security checks
- **Go Version**: Updated from Go 1.21 to Go 1.23

## Build and Development Commands

```bash
# Build the binary
make build           # Creates ./myrient-dl executable
go build -o myrient-dl .

# Run tests
make test           # Run all tests with race detection
go test -v -race ./...

# Run a single test
go test -v -race ./internal/parser -run TestParseSizeString

# Coverage report
make coverage       # Generates coverage.html

# Linting (requires golangci-lint)
make lint

# Format code
make fmt

# Tidy dependencies
make tidy           # go mod tidy + go mod verify

# Install to GOPATH/bin
make install        # go install

# Clean artifacts
make clean          # Remove binary, coverage files

# Run all checks (format, lint, test, build)
make all

# CI checks (lint + test)
make ci
```

## Architecture

The codebase follows a clean architecture with three main internal packages:

### Package Structure

- **cmd/root.go**: Cobra command implementation, CLI flag parsing, orchestration logic
  - Handles URL validation and output directory determination
  - Coordinates the parse → filter → download pipeline
  - Implements utility functions (formatBytes, sanitizeFilename)

- **internal/parser**: HTML parsing for Apache-style directory listings
  - `ParseDirectoryListing()` fetches and parses Myrient directory pages
  - Uses goquery for HTML parsing
  - Extracts FileInfo (Name, URL, Size) from directory listings
  - Smart size parsing from Apache listing formats (handles B, KiB, MiB, GiB, TiB)

- **internal/matcher**: Pattern-based file filtering
  - Implements include/exclude glob pattern matching using filepath.Match
  - `Filter()` applies patterns to file lists

- **internal/downloader**: Download orchestration with progress tracking
  - Supports both serial and parallel downloads (semaphore-based concurrency)
  - Retry logic with exponential backoff and jitter
  - Progress bars using schollz/progressbar
  - Smart resume: HEAD request to check remote size, skips if local file matches
  - 30-minute timeout for large files
  - Atomic file writes (write to .tmp, rename on success)
  - Context-aware cancellation

- **internal/version**: Version information
  - Provides version, git commit, and build time
  - Populated via ldflags during build

### Data Flow

1. User provides Myrient URL → cmd validates and parses flags, sets up context with signal handling
2. parser.ParseDirectoryListing(ctx) → fetches HTML with context, extracts FileInfo structs
3. matcher.Filter() → applies include/exclude patterns
4. downloader.DownloadAll(ctx) → downloads with progress tracking, retry, and context cancellation

### Key Design Decisions

- **Default parallel=1**: Intentionally respectful to Myrient's servers
- **Auto-resume**: HEAD requests verify file size before re-downloading
- **Output directory**: Auto-extracted from URL's last path component, sanitized for filesystem
- **Error handling**: Retry with exponential backoff and jitter, detailed error wrapping
- **Context-driven**: All network operations support cancellation via context
- **Atomic writes**: Prevents partial file corruption on interruption
- **Path safety**: Strong sanitization against directory traversal and malicious filenames

## Testing

Test files mirror the package structure:
- internal/parser/parser_test.go
- internal/matcher/matcher_test.go
- internal/downloader/downloader_test.go

Tests use table-driven patterns and focus on parsing logic, pattern matching, and size conversions.

## Dependencies

- github.com/spf13/cobra: CLI framework
- github.com/PuerkitoBio/goquery: HTML parsing
- github.com/schollz/progressbar/v3: Progress visualization

## Version Information

The binary includes version information that can be viewed with `./myrient-dl --version`:
- Version: Git tag or commit hash
- Git Commit: Short commit hash
- Build Time: UTC timestamp

This information is injected at build time via ldflags in the Makefile.

## Notes

- The tool targets Apache-style directory listings specifically (Myrient's format)
- Parser searches for links within `table#list` only, ignoring navigation links elsewhere on the page
- File size extraction uses multiple strategies (table cells, rows, parent text) to handle HTML variations
- nolint directives are used for gosec warnings where the risk is acceptable (user-provided URLs, configured file paths)
