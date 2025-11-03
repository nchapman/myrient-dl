# myrient-dl

A fast and friendly CLI tool for downloading files from [Myrient](https://myrient.erista.me/) directory listings.

## Installation

**Requires Go 1.22 or later**

```bash
go install github.com/nchapman/myrient-dl@latest
```

Or clone and build manually:

```bash
git clone https://github.com/nchapman/myrient-dl.git
cd myrient-dl
go build -o myrient-dl
```

## Quick Start

Download all files from a directory:

```bash
myrient-dl https://myrient.erista.me/files/Internet%20Archive/chadmaster/fbnarcade-fullnonmerged/arcade/
```

This will:
- Create a directory named `arcade/` (from the URL)
- Download all files into it
- Show progress for each file
- Skip files that already exist

## Features

- **Smart defaults** - Just paste a URL and go
- **Pattern matching** - Include/exclude files with glob patterns
- **Beautiful progress** - Real-time download progress with speed and ETA
- **Auto-retry** - Automatically retries failed downloads
- **Parallel downloads** - Optional concurrent downloads (defaults to 1 to be server-friendly)
- **Resume support** - Skips already downloaded files
- **Dry run** - Preview what will be downloaded

## Common Usage

### Download specific files

```bash
# Only files starting with "mario"
myrient-dl <url> --include "mario*"

# Only .zip files
myrient-dl <url> --include "*.zip"
```

### Exclude files

```bash
# Everything except prototypes
myrient-dl <url> --exclude "proto*"

# Combine include and exclude
myrient-dl <url> --include "*.zip" --exclude "*japan*"
```

### Preview before downloading

```bash
myrient-dl <url> --include "mario*" --dry-run
```

### Custom output directory

```bash
myrient-dl <url> --output ~/roms/arcade
```

### Faster downloads (use responsibly)

```bash
# Download 5 files at once
myrient-dl <url> --parallel 5
```

## All Options

```
myrient-dl [URL] [flags]
```

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--output` | `-o` | Auto-detected | Output directory |
| `--include` | `-i` | `*` | Include pattern (glob) |
| `--exclude` | `-e` | None | Exclude pattern (glob) |
| `--parallel` | `-p` | `1` | Number of parallel downloads |
| `--dry-run` | | `false` | Preview what will be downloaded |
| `--verbose` | `-v` | `false` | Verbose output |
| `--retry` | `-r` | `3` | Number of retry attempts |

## How It Works

The tool is designed with sensible defaults:

- **Output directory**: Automatically extracted from the URL (e.g., `.../arcade/` → `./arcade/`)
- **Include pattern**: `*` (all files by default)
- **Parallel downloads**: `1` (to be respectful to Myrient's servers)
- **Resume support**: Automatically skips files that already exist with the same size

## Tips

- **Test your patterns first**: Use `--dry-run` to preview what will be downloaded
- **Be server-friendly**: The default of 1 parallel download is intentional. Only increase for many small files.
- **Resume interrupted downloads**: Just run the same command again. Already downloaded files will be skipped.

## License

MIT
