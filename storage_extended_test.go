package pprofio

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHTTPStorage_ErrorConditions(t *testing.T) {
	t.Run("MissingURL", func(t *testing.T) {
		storage := NewHTTPStorage("", "test-key", "local")
		
		_, err := storage.Upload(context.Background(), "test-file")
		if err == nil {
			t.Error("Expected error when URL is missing")
		}
		
		if !strings.Contains(err.Error(), "URL and APIKey are required") {
			t.Errorf("Expected 'URL and APIKey are required' error, got: %v", err)
		}
	})
	
	t.Run("MissingAPIKey", func(t *testing.T) {
		storage := NewHTTPStorage("https://test.com", "", "local")
		
		_, err := storage.Upload(context.Background(), "test-file")
		if err == nil {
			t.Error("Expected error when API key is missing")
		}
		
		if !strings.Contains(err.Error(), "URL and APIKey are required") {
			t.Errorf("Expected 'URL and APIKey are required' error, got: %v", err)
		}
	})
	
	t.Run("InvalidURL", func(t *testing.T) {
		storage := NewHTTPStorage("invalid-url", "test-key", "local")
		
		// Create a temporary file first
		tmpFile, err := os.CreateTemp("", "test-profile")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile.Name())
		
		tmpFile.WriteString("test data")
		tmpFile.Close()
		
		_, err = storage.Upload(context.Background(), tmpFile.Name())
		if err == nil {
			t.Error("Expected error for invalid URL")
		}
		
		if !strings.Contains(err.Error(), "invalid URL") {
			t.Errorf("Expected 'invalid URL' error, got: %v", err)
		}
	})
	
	t.Run("HTTPSRequired", func(t *testing.T) {
		storage := NewHTTPStorage("http://test.com", "test-key", "production")
		
		_, err := storage.Upload(context.Background(), "test-file")
		if err == nil {
			t.Error("Expected error for HTTP URL in non-local environment")
		}
		
		if !strings.Contains(err.Error(), "HTTPS is required") {
			t.Errorf("Expected 'HTTPS is required' error, got: %v", err)
		}
	})
	
	t.Run("FileNotFound", func(t *testing.T) {
		storage := NewHTTPStorage("https://test.com", "test-key", "local")
		
		_, err := storage.Upload(context.Background(), "non-existent-file.txt")
		if err == nil {
			t.Error("Expected error for non-existent file")
		}
		
		if !strings.Contains(err.Error(), "failed to open file") {
			t.Errorf("Expected 'failed to open file' error, got: %v", err)
		}
	})
	
	t.Run("AuthenticationFailed", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}))
		defer server.Close()
		
		storage := NewHTTPStorage(server.URL, "invalid-key", "local")
		
		// Create a temporary file
		tmpFile, err := os.CreateTemp("", "test-profile")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile.Name())
		
		tmpFile.WriteString("test data")
		tmpFile.Close()
		
		_, err = storage.Upload(context.Background(), tmpFile.Name())
		if err == nil {
			t.Error("Expected authentication error")
		}
		
		if !strings.Contains(err.Error(), "authentication failed") {
			t.Errorf("Expected 'authentication failed' error, got: %v", err)
		}
	})
	
	t.Run("ServerError", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()
		
		storage := NewHTTPStorage(server.URL, "test-key", "local")
		storage.Retries = 1 // Reduce retries for faster testing
		
		// Create a temporary file
		tmpFile, err := os.CreateTemp("", "test-profile")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile.Name())
		
		tmpFile.WriteString("test data")
		tmpFile.Close()
		
		_, err = storage.Upload(context.Background(), tmpFile.Name())
		if err == nil {
			t.Error("Expected server error")
		}
		
		if !strings.Contains(err.Error(), "upload failed after") {
			t.Errorf("Expected 'upload failed after' error, got: %v", err)
		}
	})
	
	t.Run("TooManyRequests", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusTooManyRequests)
		}))
		defer server.Close()
		
		storage := NewHTTPStorage(server.URL, "test-key", "local")
		storage.Retries = 1 // Reduce retries for faster testing
		
		// Create a temporary file
		tmpFile, err := os.CreateTemp("", "test-profile")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile.Name())
		
		tmpFile.WriteString("test data")
		tmpFile.Close()
		
		_, err = storage.Upload(context.Background(), tmpFile.Name())
		if err == nil {
			t.Error("Expected rate limit error")
		}
	})
	
	t.Run("UnexpectedStatusCode", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusTeapot) // 418
		}))
		defer server.Close()
		
		storage := NewHTTPStorage(server.URL, "test-key", "local")
		
		// Create a temporary file
		tmpFile, err := os.CreateTemp("", "test-profile")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile.Name())
		
		tmpFile.WriteString("test data")
		tmpFile.Close()
		
		_, err = storage.Upload(context.Background(), tmpFile.Name())
		if err == nil {
			t.Error("Expected unexpected status code error")
		}
		
		if !strings.Contains(err.Error(), "unexpected status code: 418") {
			t.Errorf("Expected 'unexpected status code: 418' error, got: %v", err)
		}
	})
}

func TestFileStorage_ErrorConditions(t *testing.T) {
	t.Run("EmptyDirectory", func(t *testing.T) {
		_, err := NewFileStorage("")
		if err == nil {
			t.Error("Expected error for empty directory")
		}
		
		if !strings.Contains(err.Error(), "directory is required") {
			t.Errorf("Expected 'directory is required' error, got: %v", err)
		}
	})
	
	t.Run("InvalidDirectory", func(t *testing.T) {
		// Try to create storage in a path that can't be created (like root on most systems)
		_, err := NewFileStorage("/invalid/path/that/cannot/be/created")
		if err == nil {
			t.Error("Expected error for invalid directory path")
		}
	})
	
	t.Run("UploadWithEmptyDirectory", func(t *testing.T) {
		storage := &FileStorage{Directory: ""}
		
		_, err := storage.Upload(context.Background(), "test-file")
		if err == nil {
			t.Error("Expected error when directory is empty")
		}
		
		if !strings.Contains(err.Error(), "directory is required") {
			t.Errorf("Expected 'directory is required' error, got: %v", err)
		}
	})
	
	t.Run("UploadNonExistentFile", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "pprofio-test")
		if err != nil {
			t.Fatalf("Failed to create temp directory: %v", err)
		}
		defer os.RemoveAll(tempDir)
		
		storage, err := NewFileStorage(tempDir)
		if err != nil {
			t.Fatalf("Failed to create storage: %v", err)
		}
		
		_, err = storage.Upload(context.Background(), "non-existent-file.txt")
		if err == nil {
			t.Error("Expected error for non-existent file")
		}
		
		if !strings.Contains(err.Error(), "failed to open source file") {
			t.Errorf("Expected 'failed to open source file' error, got: %v", err)
		}
	})
}

func TestStdoutStorage_EdgeCases(t *testing.T) {
	t.Run("NonExistentFile", func(t *testing.T) {
		storage := NewStdoutStorage()
		
		_, err := storage.Upload(context.Background(), "non-existent-file.txt")
		if err == nil {
			t.Error("Expected error for non-existent file")
		}
		
		if !strings.Contains(err.Error(), "failed to read profile file") {
			t.Errorf("Expected 'failed to read profile file' error, got: %v", err)
		}
	})
	
	t.Run("DisplayPprofDataVariousTypes", func(t *testing.T) {
		storage := NewStdoutStorage()
		
		// Test different profile types based on filename
		profileTypes := []string{
			"cpu.pprof",
			"memory.pprof",
			"heap.pprof",
			"goroutine.pprof",
			"mutex.pprof",
			"block.pprof",
			"unknown.pprof",
		}
		
		for _, profileType := range profileTypes {
			// Create temporary file with specific name
			tmpFile, err := os.CreateTemp("", profileType)
			if err != nil {
				t.Fatalf("Failed to create temp file for %s: %v", profileType, err)
			}
			
			tmpFile.WriteString("test pprof data")
			tmpFile.Close()
			
			// Test displayPprofData
			err = storage.displayPprofData(tmpFile.Name())
			if err != nil {
				t.Errorf("displayPprofData failed for %s: %v", profileType, err)
			}
			
			os.Remove(tmpFile.Name())
		}
	})
	
	t.Run("DisplayPprofDataFileStatError", func(t *testing.T) {
		storage := NewStdoutStorage()
		
		// Test with non-existent file
		err := storage.displayPprofData("non-existent-file.pprof")
		if err == nil {
			t.Error("Expected error for non-existent file in displayPprofData")
		}
	})
	
	t.Run("OutputMetadataError", func(t *testing.T) {
		storage := NewStdoutStorage()
		
		// Create metadata that can't be marshaled (contains invalid data)
		invalidMetadata := map[string]string{
			"invalid": string([]byte{0xff, 0xfe, 0xfd}), // Invalid UTF-8
		}
		
		// This should work fine actually, since Go's JSON encoder handles this
		err := storage.OutputMetadata(invalidMetadata)
		if err != nil {
			t.Errorf("OutputMetadata should handle any string data: %v", err)
		}
	})
}

func TestHTTPStorage_SuccessfulUpload(t *testing.T) {
	responseBody := "Upload successful"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		if r.Header.Get("Content-Type") != "application/octet-stream" {
			t.Errorf("Expected Content-Type 'application/octet-stream', got '%s'", r.Header.Get("Content-Type"))
		}
		
		if r.Header.Get("Content-Encoding") != "gzip" {
			t.Errorf("Expected Content-Encoding 'gzip', got '%s'", r.Header.Get("Content-Encoding"))
		}
		
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("Expected Authorization 'Bearer test-key', got '%s'", r.Header.Get("Authorization"))
		}
		
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(responseBody))
	}))
	defer server.Close()
	
	storage := NewHTTPStorage(server.URL, "test-key", "local")
	
	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "test-profile")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	
	tmpFile.WriteString("test profile data")
	tmpFile.Close()
	
	result, err := storage.Upload(context.Background(), tmpFile.Name())
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}
	
	if result != responseBody {
		t.Errorf("Expected response '%s', got '%s'", responseBody, result)
	}
}

func TestFileStorage_SuccessfulUpload(t *testing.T) {
	// Create temporary directories
	sourceDir, err := os.MkdirTemp("", "pprofio-source")
	if err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}
	defer os.RemoveAll(sourceDir)
	
	targetDir, err := os.MkdirTemp("", "pprofio-target")
	if err != nil {
		t.Fatalf("Failed to create target directory: %v", err)
	}
	defer os.RemoveAll(targetDir)
	
	// Create source file
	sourceFile := filepath.Join(sourceDir, "test-profile.pprof")
	testContent := "test profile data"
	err = os.WriteFile(sourceFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}
	
	// Create storage
	storage, err := NewFileStorage(targetDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	
	// Upload file
	result, err := storage.Upload(context.Background(), sourceFile)
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}
	
	expectedPath := filepath.Join(targetDir, "test-profile.pprof")
	if result != expectedPath {
		t.Errorf("Expected result path '%s', got '%s'", expectedPath, result)
	}
	
	// Verify file was copied
	copiedContent, err := os.ReadFile(result)
	if err != nil {
		t.Fatalf("Failed to read copied file: %v", err)
	}
	
	if string(copiedContent) != testContent {
		t.Errorf("Expected content '%s', got '%s'", testContent, string(copiedContent))
	}
}