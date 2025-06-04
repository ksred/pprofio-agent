package pprofio

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestHTTPStorage_Upload(t *testing.T) {
	t.Parallel() // Run in parallel with other tests
	// Create a test file
	content := "test profile data"
	tmpFile, err := os.CreateTemp("", "profile.pprof")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	// Create a test server
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check headers
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("Expected Authorization header 'Bearer test-key', got %q", r.Header.Get("Authorization"))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if r.Header.Get("Content-Encoding") != "gzip" {
			t.Errorf("Expected Content-Encoding header 'gzip', got %q", r.Header.Get("Content-Encoding"))
		}

		// Respond with success
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("https://storage.pprofio.com/profile123"))
	}))
	defer server.Close()

	// Create HTTP storage
	storage := &HTTPStorage{
		URL:     server.URL,
		APIKey:  "test-key",
		Client:  server.Client(),
		Retries: 1,
	}

	// Upload the file
	url, err := storage.Upload(context.Background(), tmpFile.Name())
	if err != nil {
		t.Fatalf("Storage.Upload() error = %v", err)
	}

	// Check the result
	if url != "https://storage.pprofio.com/profile123" {
		t.Errorf("Storage.Upload() returned %q, want %q", url, "https://storage.pprofio.com/profile123")
	}
}

func TestFileStorage_Upload(t *testing.T) {
	// Create a test file
	content := "test profile data"
	tmpFile, err := os.CreateTemp("", "profile.pprof")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	// Create a directory for storage
	tmpDir, err := os.MkdirTemp("", "pprofio-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create file storage
	storage, err := NewFileStorage(tmpDir)
	if err != nil {
		t.Fatalf("NewFileStorage() error = %v", err)
	}

	// Upload the file
	path, err := storage.Upload(context.Background(), tmpFile.Name())
	if err != nil {
		t.Fatalf("Storage.Upload() error = %v", err)
	}

	// Check the result
	expectedPath := filepath.Join(tmpDir, filepath.Base(tmpFile.Name()))
	if path != expectedPath {
		t.Errorf("Storage.Upload() returned %q, want %q", path, expectedPath)
	}

	// Check the file was copied
	copiedContent, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read copied file: %v", err)
	}

	if string(copiedContent) != content {
		t.Errorf("Copied file has content %q, want %q", string(copiedContent), content)
	}
}

func TestNewFileStorage_Error(t *testing.T) {
	// Test with empty directory
	_, err := NewFileStorage("")
	if err == nil {
		t.Error("NewFileStorage() with empty directory should return error")
	}

	// Test with invalid directory (file exists with same name)
	tmpFile, err := os.CreateTemp("", "not-a-dir")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	_, err = NewFileStorage(tmpFile.Name())
	if err == nil {
		t.Error("NewFileStorage() with file path should return error")
	}
}