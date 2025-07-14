package pprofio

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

const (
	StorageTestAPIKey           = "test-key"
	StorageTestProfileData      = "test profile data"
	StorageTestProfileFilename  = "profile.pprof"
	StorageTestDirectoryPrefix  = "pprofio-test"
	StorageTestExpectedURL      = "https://storage.pprofio.com/profile123"
	StorageTestExpectedAuth     = "Bearer test-key"
	StorageTestExpectedEncoding = "gzip"
	StorageTestRetries          = 1
)

func TestHTTPStorage_Upload(t *testing.T) {
	t.Parallel() // Run in parallel with other tests
	// Create a test file
	content := StorageTestProfileData
	tmpFile, err := os.CreateTemp("", StorageTestProfileFilename)
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
		if r.Header.Get("Authorization") != StorageTestExpectedAuth {
			t.Errorf("Expected Authorization header '%s', got %q", StorageTestExpectedAuth, r.Header.Get("Authorization"))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if r.Header.Get("Content-Encoding") != StorageTestExpectedEncoding {
			t.Errorf("Expected Content-Encoding header '%s', got %q", StorageTestExpectedEncoding, r.Header.Get("Content-Encoding"))
		}

		// Respond with success
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(StorageTestExpectedURL))
	}))
	defer server.Close()

	// Create HTTP storage
	storage := &HTTPStorage{
		URL:     server.URL,
		APIKey:  StorageTestAPIKey,
		Client:  server.Client(),
		Retries: StorageTestRetries,
	}

	// Upload the file
	url, err := storage.Upload(context.Background(), tmpFile.Name())
	if err != nil {
		t.Fatalf("Storage.Upload() error = %v", err)
	}

	// Check the result
	if url != StorageTestExpectedURL {
		t.Errorf("Storage.Upload() returned %q, want %q", url, StorageTestExpectedURL)
	}
}

func TestFileStorage_Upload(t *testing.T) {
	// Create a test file
	content := StorageTestProfileData
	tmpFile, err := os.CreateTemp("", StorageTestProfileFilename)
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
	tmpDir, err := os.MkdirTemp("", StorageTestDirectoryPrefix)
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

	// Parse the JSON response
	var response map[string]string
	if err := json.Unmarshal([]byte(path), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	// Check the response has required fields
	if response["profile_id"] == "" {
		t.Error("Response missing profile_id")
	}
	if response["profile_url"] == "" {
		t.Error("Response missing profile_url")
	}
	if response["type"] == "" {
		t.Error("Response missing type")
	}

	// Check the file was actually copied to the expected location
	expectedPath := filepath.Join(tmpDir, filepath.Base(tmpFile.Name()))
	copiedContent, err := os.ReadFile(expectedPath)
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
