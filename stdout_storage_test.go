package pprofio

import (
	"context"
	"os"
	"strings"
	"testing"
)

func TestStdoutStorage_Upload(t *testing.T) {
	// Create a test file with profile data
	content := "test profile data for stdout"
	tmpFile, err := os.CreateTemp("", "cpu.pprof")
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

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Create stdout storage
	storage := NewStdoutStorage()

	// Upload the file
	result, err := storage.Upload(context.Background(), tmpFile.Name())
	if err != nil {
		t.Fatalf("StdoutStorage.Upload() error = %v", err)
	}

	// Close write pipe and restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read captured output
	buf := make([]byte, 2048)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	// Verify result
	if result != "stdout" {
		t.Errorf("StdoutStorage.Upload() result = %q, want %q", result, "stdout")
	}

	// Verify output contains structured profile information
	if !strings.Contains(output, "PROFILE_DATA") {
		t.Error("StdoutStorage output should contain PROFILE_DATA header")
	}

	if !strings.Contains(output, "Type: CPU Profile") {
		t.Error("StdoutStorage output should identify CPU profile type")
	}

	if !strings.Contains(output, "Size:") {
		t.Error("StdoutStorage output should show file size")
	}
}

func TestStdoutStorage_OutputsMetadata(t *testing.T) {
	// Create a test file
	tmpFile, err := os.CreateTemp("", "memory.pprof")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString("memory profile data"); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Create stdout storage and upload
	storage := NewStdoutStorage()
	_, err = storage.Upload(context.Background(), tmpFile.Name())
	if err != nil {
		t.Fatalf("StdoutStorage.Upload() error = %v", err)
	}

	// Output metadata
	metadata := map[string]string{
		"profile_id":   "test-123",
		"service_name": "test-service",
		"profile_type": "memory",
		"profile_url":  "https://storage.pprofio.com/profiles/test-123.pprof",
	}
	err = storage.OutputMetadata(metadata)
	if err != nil {
		t.Fatalf("StdoutStorage.OutputMetadata() error = %v", err)
	}

	// Close write pipe and restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read captured output
	buf := make([]byte, 2048)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	// Verify profile output
	if !strings.Contains(output, "Type: Memory/Heap Profile") {
		t.Error("Should identify memory profile type")
	}

	// Verify metadata output
	if !strings.Contains(output, "METADATA:") {
		t.Error("Should contain METADATA marker")
	}

	if !strings.Contains(output, "https://storage.pprofio.com/profiles/test-123.pprof") {
		t.Error("Should contain profile_url in metadata")
	}

	if !strings.Contains(output, "test-service") {
		t.Error("Should contain service in metadata")
	}
}

func TestConfig_WithStdoutOutput(t *testing.T) {
	cfg := Config{
		APIKey:         "test-key",
		IngestURL:      "https://api.pprofio.com",
		ServiceName:    "test-service",
		OutputToStdout: true,
	}

	err := cfg.validate()
	if err != nil {
		t.Fatalf("Config.validate() with OutputToStdout should not error: %v", err)
	}

	// Test that when OutputToStdout is true, APIKey and IngestURL are not required
	cfg2 := Config{
		ServiceName:    "test-service",
		OutputToStdout: true,
	}

	err = cfg2.validate()
	if err != nil {
		t.Fatalf("Config.validate() with OutputToStdout=true should not require APIKey/IngestURL: %v", err)
	}
}

func TestNew_WithStdoutStorage(t *testing.T) {
	cfg := Config{
		ServiceName:    "test-service",
		OutputToStdout: true,
		EnableCPU:      true,
	}

	profiler, err := New(cfg)
	if err != nil {
		t.Fatalf("New() with OutputToStdout config should not error: %v", err)
	}

	// Verify stdout storage was created
	if profiler.config.Storage == nil {
		t.Error("Profiler should have stdout storage when OutputToStdout is true")
	}

	// Type assertion to verify it's stdout storage
	if _, ok := profiler.config.Storage.(*StdoutStorage); !ok {
		t.Error("Profiler should use StdoutStorage when OutputToStdout is true")
	}
}
