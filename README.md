# myrient-dl

A fast and friendly CLI tool for downloading files from [Myrient](https://myrient.erista.me/) directory listings.

## Features

- 🎯 **Smart defaults** - Just paste a URL and go
- 🔍 **Pattern matching** - Include/exclude files with glob patterns
- 📊 **Beautiful progress** - Real-time download progress with speed and ETA
- 🔄 **Auto-retry** - Automatically retries failed downloads
- ⚡ **Parallel downloads** - Optional concurrent downloads (defaults to 1 to be server-friendly)
- ✅ **Resume support** - Skips already downloaded files
- 🧪 **Dry run** - Preview what will be downloaded

## Installation

```bash
go install github.com/nchapman/myrient-dl@latest
```

Or build from source:

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

## Usage

```
myrient-dl [URL] [flags]
```

### Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--output` | `-o` | Auto-detected | Output directory |
| `--include` | `-i` | `*` | Include pattern (glob) |
| `--exclude` | `-e` | None | Exclude pattern (glob) |
| `--parallel` | `-p` | `1` | Number of parallel downloads |
| `--dry-run` | | `false` | Show what would be downloaded |
| `--verbose` | `-v` | `false` | Verbose output |
| `--retry` | `-r` | `3` | Number of retry attempts |

## Examples

### Download specific files

Download only files starting with "mario":

```bash
myrient-dl https://myrient.erista.me/.../arcade/ --include "mario*"
```

### Exclude files

Download all except prototype versions:

```bash
myrient-dl https://myrient.erista.me/.../arcade/ --exclude "proto*"
```

### Multiple patterns

Download only .zip files, exclude Japan releases:

```bash
myrient-dl https://myrient.erista.me/.../arcade/ \
  --include "*.zip" \
  --exclude "*japan*"
```

### Parallel downloads

Download 5 files at once (use responsibly!):

```bash
myrient-dl https://myrient.erista.me/.../arcade/ --parallel 5
```

### Custom output directory

```bash
myrient-dl https://myrient.erista.me/.../arcade/ --output ~/roms/arcade
```

### Preview before downloading

```bash
myrient-dl https://myrient.erista.me/.../arcade/ \
  --include "mario*" \
  --dry-run
```

## Default Behavior

The tool is designed with sensible defaults:

- **Output directory**: Automatically uses the last path component from the URL
  - Example: `.../arcade/` → `./arcade/`
- **Include pattern**: `*` (all files)
- **Exclude pattern**: None
- **Parallel downloads**: `1` (to be respectful to Myrient's servers)
- **Skip existing**: Automatically skips files that already exist with the same size

## Tips

1. **Start with a dry run** to verify your patterns:
   ```bash
   myrient-dl <url> --include "pattern*" --dry-run
   ```

2. **Be nice to the servers** - The default of 1 parallel download is intentional. Only increase if downloading many small files.

3. **Resume interrupted downloads** - Just run the same command again. Already downloaded files will be skipped.

4. **Combine patterns** - Use include and exclude together for precise control:
   ```bash
   myrient-dl <url> --include "*.zip" --exclude "*beta*"
   ```

## License

MIT
