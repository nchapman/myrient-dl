# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

myrient-dl is a CLI tool for downloading files from Myrient (https://myrient.erista.me/) directory listings. It's built in Go using the Cobra CLI framework and provides features like pattern matching, parallel downloads, progress tracking, and auto-retry.

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
  - Retry logic with exponential backoff
  - Progress bars using schollz/progressbar
  - Smart resume: HEAD request to check remote size, skips if local file matches
  - 30-minute timeout for large files

### Data Flow

1. User provides Myrient URL → cmd validates and parses flags
2. parser.ParseDirectoryListing() → fetches HTML, extracts FileInfo structs
3. matcher.Filter() → applies include/exclude patterns
4. downloader.DownloadAll() → downloads with progress tracking and retry

### Key Design Decisions

- **Default parallel=1**: Intentionally respectful to Myrient's servers
- **Auto-resume**: HEAD requests verify file size before re-downloading
- **Output directory**: Auto-extracted from URL's last path component, sanitized for filesystem
- **Error handling**: Retry with exponential backoff, detailed error wrapping

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

## Notes

- The tool targets Apache-style directory listings specifically (Myrient's format)
- File size extraction uses multiple strategies (table cells, rows, parent text) to handle HTML variations
- nolint directives are used for gosec warnings where the risk is acceptable (user-provided URLs, configured file paths)
