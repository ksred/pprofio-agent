package pprofio

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

func TestStdoutIntegration_EndToEnd(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Configure profiler with stdout output
	cfg := Config{
		SampleRate:      1 * time.Second,        // Fast for testing
		ProfileDuration: 100 * time.Millisecond, // Very short for testing
		ServiceName:     "integration-test-service",
		Tags:            map[string]string{"test": "integration", "env": "testing"},
		EnableCPU:       true,
		EnableMemory:    true,
		OutputToStdout:  true,
	}

	// Create and start profiler
	profiler, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create profiler: %v", err)
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Start profiling (this will run in background)
	if err := profiler.Start(ctx); err != nil {
		t.Fatalf("Failed to start profiler: %v", err)
	}

	// Let it run briefly to generate some profiles
	time.Sleep(300 * time.Millisecond)

	// Stop profiling
	profiler.Stop()

	// Close write pipe and restore stdout
	w.Close()
	os.Stdout = oldStdout

	// Read captured output
	buf, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("Failed to read captured output: %v", err)
	}
	output := string(buf)

	// Verify we got profile output with the new format
	if !strings.Contains(output, "PROFILE_DATA") {
		t.Error("Should contain PROFILE_DATA header")
	}

	// Should contain readable profile information
	if !strings.Contains(output, "Type:") {
		t.Error("Should contain profile type information")
	}

	if !strings.Contains(output, "Size:") {
		t.Error("Should contain profile size information")
	}

	// Verify metadata output
	if !strings.Contains(output, "METADATA:") {
		t.Error("Should contain metadata output")
	}

	// Should contain service configuration
	if !strings.Contains(output, "integration-test-service") {
		t.Error("Should contain service name in metadata")
	}

	// Should contain our custom tags
	if !strings.Contains(output, "integration") {
		t.Error("Should contain test tag in metadata")
	}

	if !strings.Contains(output, "testing") {
		t.Error("Should contain env tag in metadata")
	}

	// Parse metadata to ensure it's valid JSON
	lines := strings.Split(output, "\n")
	var foundValidMetadata bool
	for _, line := range lines {
		if strings.HasPrefix(line, "METADATA:") {
			metadataJSON := strings.TrimSpace(strings.TrimPrefix(line, "METADATA:"))
			var metadata map[string]string
			if err := json.Unmarshal([]byte(metadataJSON), &metadata); err == nil {
				foundValidMetadata = true

				// Verify key metadata fields
				if metadata["service"] != "integration-test-service" {
					t.Error("Service name should match configuration")
				}

				if metadata["test"] != "integration" {
					t.Error("Custom tag 'test' should be present")
				}

				break
			}
		}
	}

	if !foundValidMetadata {
		t.Error("Should contain valid JSON metadata")
	}
}
