package parser

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseDirectoryListing(t *testing.T) {
	// Create a mock Apache directory listing
	html := `<!DOCTYPE HTML PUBLIC "-//W3C//DTD HTML 3.2 Final//EN">
<html>
 <head>
  <title>Index of /files/arcade</title>
 </head>
 <body>
<h1>Index of /files/arcade</h1>
  <table id="list">
   <tr><th valign="top">&nbsp;</th><th><a href="?C=N;O=D">Name</a></th><th><a href="?C=M;O=A">Last modified</a></th><th><a href="?C=S;O=A">Size</a></th></tr>
   <tr><th colspan="4"><hr></th></tr>
<tr><td valign="top">&nbsp;</td><td><a href="../">Parent Directory</a></td><td>&nbsp;</td><td align="right">  - </td></tr>
<tr><td valign="top">&nbsp;</td><td><a href="arkanoid.zip">arkanoid.zip</a></td><td align="right">2023-09-11 09:52  </td><td align="right"> 70.5 KiB</td></tr>
<tr><td valign="top">&nbsp;</td><td><a href="mario.zip">mario.zip</a></td><td align="right">2023-09-11 10:30  </td><td align="right">  1.2 MiB</td></tr>
<tr><td valign="top">&nbsp;</td><td><a href="sonic.zip">sonic.zip</a></td><td align="right">2023-09-11 11:00  </td><td align="right">500 B</td></tr>
<tr><td valign="top">&nbsp;</td><td><a href="large.zip">large.zip</a></td><td align="right">2023-09-11 12:00  </td><td align="right">  2.5 GiB</td></tr>
<tr><td valign="top">&nbsp;</td><td><a href="subdir/">subdir/</a></td><td align="right">2023-09-11 13:00  </td><td align="right">  - </td></tr>
   <tr><th colspan="4"><hr></th></tr>
</table>
</body></html>`

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(html))
	}))
	defer server.Close()

	// Parse the directory listing
	files, err := ParseDirectoryListing(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("failed to parse directory listing: %v", err)
	}

	// Should find 4 files (excluding parent directory and subdirectory)
	expectedFiles := 4
	if len(files) != expectedFiles {
		t.Errorf("expected %d files, got %d", expectedFiles, len(files))
	}

	// Check file details
	tests := []struct {
		name         string
		expectedSize int64
	}{
		{"arkanoid.zip", 72192},   // 70.5 KiB
		{"mario.zip", 1258291},    // 1.2 MiB
		{"sonic.zip", 500},        // 500 B
		{"large.zip", 2684354560}, // 2.5 GiB
	}

	for i, tt := range tests {
		if i >= len(files) {
			t.Errorf("missing expected file: %s", tt.name)
			continue
		}

		file := files[i]
		if file.Name != tt.name {
			t.Errorf("expected file name %s, got %s", tt.name, file.Name)
		}

		// Check URL is absolute
		if !strings.HasPrefix(file.URL, "http") {
			t.Errorf("expected absolute URL, got %s", file.URL)
		}

		// Check size (allow 1% tolerance for rounding)
		tolerance := int64(float64(tt.expectedSize) * 0.01)
		if file.Size < tt.expectedSize-tolerance || file.Size > tt.expectedSize+tolerance {
			t.Errorf("expected size around %d bytes for %s, got %d", tt.expectedSize, tt.name, file.Size)
		}
	}
}

func TestParseDirectoryListing_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	_, err := ParseDirectoryListing(context.Background(), server.URL)
	if err == nil {
		t.Error("expected error for server error response, got nil")
	}
}

func TestParseDirectoryListing_EmptyListing(t *testing.T) {
	html := `<!DOCTYPE HTML>
<html>
<body>
<h1>Index of /empty</h1>
<table id="list">
<tr><td><a href="../">Parent Directory</a></td></tr>
</table>
</body>
</html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(html))
	}))
	defer server.Close()

	files, err := ParseDirectoryListing(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(files) != 0 {
		t.Errorf("expected 0 files for empty listing, got %d", len(files))
	}
}

func TestParseSizeString(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"70.5 KiB", 72192},
		{"1.2 MiB", 1258291},
		{"500 B", 500},
		{"2.5 GiB", 2684354560},
		{"1 TiB", 1099511627776},
		{"invalid", 0},
		{"", 0},
		{"123", 0}, // no unit
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseSizeString(tt.input)
			// Allow 1% tolerance for floating point math
			tolerance := int64(float64(tt.expected) * 0.01)
			if tt.expected == 0 {
				tolerance = 0
			}
			if result < tt.expected-tolerance || result > tt.expected+tolerance {
				t.Errorf("parseSizeString(%q) = %d, want ~%d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestBuildAbsoluteURL(t *testing.T) {
	tests := []struct {
		base     string
		relative string
		expected string
	}{
		{
			"http://example.com/files/",
			"file.zip",
			"http://example.com/files/file.zip",
		},
		{
			"http://example.com/files",
			"file.zip",
			"http://example.com/file.zip",
		},
		{
			"http://example.com/dir/",
			"../file.zip",
			"http://example.com/file.zip",
		},
		{
			"http://example.com/",
			"http://other.com/file.zip",
			"http://other.com/file.zip",
		},
	}

	for _, tt := range tests {
		t.Run(tt.base+"+"+tt.relative, func(t *testing.T) {
			result, err := buildAbsoluteURL(tt.base, tt.relative)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("buildAbsoluteURL(%q, %q) = %q, want %q", tt.base, tt.relative, result, tt.expected)
			}
		})
	}
}

func TestBuildAbsoluteURL_InvalidBase(t *testing.T) {
	// url.Parse is very lenient and rarely errors, so test with URL that contains
	// invalid characters that would cause parse to fail
	_, err := buildAbsoluteURL("ht\ntp://example.com", "file.zip")
	if err == nil {
		t.Error("expected error for invalid base URL, got nil")
	}
}

func TestParseDirectoryListing_IgnoresLinksOutsideTable(t *testing.T) {
	// Test that links outside table#list are ignored (e.g., navigation links)
	html := `<!DOCTYPE HTML>
<html>
<head>
  <title>Index of /files</title>
</head>
<body>
<h1>Index of /files</h1>
<nav>
  <a href="https://discord.gg/example">Discord</a>
  <a href="https://t.me/example">Telegram</a>
  <a href="https://hshop.example">hShop</a>
</nav>
<table id="list">
  <tr><th><a href="?C=N;O=D">Name</a></th><th><a href="?C=S;O=A">Size</a></th></tr>
  <tr><td><a href="../">Parent Directory</a></td><td>-</td></tr>
  <tr><td><a href="file1.zip">file1.zip</a></td><td>1.0 MiB</td></tr>
</table>
<footer>
  <a href="/about">About</a>
</footer>
</body>
</html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(html))
	}))
	defer server.Close()

	files, err := ParseDirectoryListing(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should only find file1.zip, not Discord, Telegram, hShop, or About links
	if len(files) != 1 {
		t.Errorf("expected 1 file, got %d", len(files))
	}

	if len(files) > 0 && files[0].Name != "file1.zip" {
		t.Errorf("expected file1.zip, got %s", files[0].Name)
	}
}
