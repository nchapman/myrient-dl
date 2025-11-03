package downloader

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/nchapman/myrient-dl/internal/parser"
)

func TestDownloader_GetRemoteFileSize(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodHead {
			t.Errorf("expected HEAD request, got %s", r.Method)
		}
		w.Header().Set("Content-Length", "12345")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	dl := New(Config{})
	size, err := dl.getRemoteFileSize(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if size != 12345 {
		t.Errorf("expected size 12345, got %d", size)
	}
}

func TestDownloader_GetRemoteFileSize_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	dl := New(Config{})
	_, err := dl.getRemoteFileSize(context.Background(), server.URL)
	if err == nil {
		t.Error("expected error for 404 response, got nil")
	}
}

func TestDownloader_DownloadFile(t *testing.T) {
	testContent := []byte("test file content")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodHead:
			w.Header().Set("Content-Length", "17")
			w.WriteHeader(http.StatusOK)
		case http.MethodGet:
			w.Header().Set("Content-Length", "17")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(testContent)
		}
	}))
	defer server.Close()

	// Create temp directory for test
	tmpDir := t.TempDir()

	dl := New(Config{
		OutputDir:     tmpDir,
		RetryAttempts: 1,
		Verbose:       false,
	})

	file := parser.FileInfo{
		Name: "test.zip",
		URL:  server.URL + "/test.zip",
		Size: 17,
	}

	err := dl.downloadFile(context.Background(), file)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify file was created
	outputPath := filepath.Join(tmpDir, "test.zip")
	content, err := os.ReadFile(outputPath) //nolint:gosec // Test file path is safe (from t.TempDir)
	if err != nil {
		t.Fatalf("failed to read downloaded file: %v", err)
	}

	if string(content) != string(testContent) {
		t.Errorf("expected content %q, got %q", testContent, content)
	}
}

func TestDownloader_SkipExistingFile(t *testing.T) {
	testContent := []byte("existing content")

	// Create temp directory with existing file
	tmpDir := t.TempDir()
	existingFile := filepath.Join(tmpDir, "existing.zip")
	if err := os.WriteFile(existingFile, testContent, 0600); err != nil { //nolint:gosec // Test file permissions can be restrictive
		t.Fatalf("failed to create existing file: %v", err)
	}

	downloadCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodHead:
			w.Header().Set("Content-Length", "16")
			w.WriteHeader(http.StatusOK)
		case http.MethodGet:
			downloadCalled = true
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	dl := New(Config{
		OutputDir:     tmpDir,
		RetryAttempts: 1,
		Verbose:       false,
	})

	file := parser.FileInfo{
		Name: "existing.zip",
		URL:  server.URL + "/existing.zip",
		Size: 16,
	}

	err := dl.downloadFile(context.Background(), file)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify download was NOT called (file was skipped)
	if downloadCalled {
		t.Error("expected existing file to be skipped, but download was called")
	}

	// Verify original content is unchanged
	content, err := os.ReadFile(existingFile) //nolint:gosec // Test file path is safe (from t.TempDir)
	if err != nil {
		t.Fatalf("failed to read existing file: %v", err)
	}

	if string(content) != string(testContent) {
		t.Error("existing file content was modified")
	}
}

func TestDownloader_RedownloadWrongSize(t *testing.T) {
	testContent := []byte("new content")

	// Create temp directory with existing file of wrong size
	tmpDir := t.TempDir()
	existingFile := filepath.Join(tmpDir, "wrong.zip")
	if err := os.WriteFile(existingFile, []byte("old"), 0600); err != nil { //nolint:gosec // Test file permissions can be restrictive
		t.Fatalf("failed to create existing file: %v", err)
	}

	downloadCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodHead:
			w.Header().Set("Content-Length", "11")
			w.WriteHeader(http.StatusOK)
		case http.MethodGet:
			downloadCalled = true
			w.Header().Set("Content-Length", "11")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(testContent)
		}
	}))
	defer server.Close()

	dl := New(Config{
		OutputDir:     tmpDir,
		RetryAttempts: 1,
		Verbose:       false,
	})

	file := parser.FileInfo{
		Name: "wrong.zip",
		URL:  server.URL + "/wrong.zip",
		Size: 11,
	}

	err := dl.downloadFile(context.Background(), file)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify download WAS called (file had wrong size)
	if !downloadCalled {
		t.Error("expected file with wrong size to be re-downloaded")
	}

	// Verify new content
	content, err := os.ReadFile(existingFile) //nolint:gosec // Test file path is safe (from t.TempDir)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	if string(content) != string(testContent) {
		t.Errorf("expected new content %q, got %q", testContent, content)
	}
}

func TestDownloader_DownloadAll(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodHead:
			w.Header().Set("Content-Length", "5")
			w.WriteHeader(http.StatusOK)
		case http.MethodGet:
			w.Header().Set("Content-Length", "5")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("hello"))
		}
	}))
	defer server.Close()

	tmpDir := t.TempDir()

	dl := New(Config{
		OutputDir:     tmpDir,
		Parallel:      1,
		RetryAttempts: 1,
		Verbose:       false,
	})

	files := []parser.FileInfo{
		{Name: "file1.zip", URL: server.URL + "/file1.zip", Size: 5},
		{Name: "file2.zip", URL: server.URL + "/file2.zip", Size: 5},
	}

	err := dl.DownloadAll(context.Background(), files)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify both files were created
	for _, file := range files {
		path := filepath.Join(tmpDir, file.Name)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist", file.Name)
		}
	}
}

func TestNew(t *testing.T) {
	config := Config{
		OutputDir:     "/tmp/test",
		Parallel:      3,
		RetryAttempts: 5,
		Verbose:       true,
	}

	dl := New(config)

	if dl.config.OutputDir != config.OutputDir {
		t.Errorf("expected OutputDir %s, got %s", config.OutputDir, dl.config.OutputDir)
	}

	if dl.config.Parallel != config.Parallel {
		t.Errorf("expected Parallel %d, got %d", config.Parallel, dl.config.Parallel)
	}

	if dl.client == nil {
		t.Error("expected client to be initialized")
	}
}
