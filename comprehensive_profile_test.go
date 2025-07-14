package pprofio

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestComprehensiveProfileCollection tests that all profile types are properly collected and transmitted
func TestComprehensiveProfileCollection(t *testing.T) {
	// Create comprehensive configuration with stdout storage to avoid network issues
	config := ComprehensiveConfig("test-key", "https://api.pprofio.com", "test-service")
	config.SampleRate = 100 * time.Millisecond
	config.ProfileDuration = 50 * time.Millisecond
	config.OutputToStdout = true // Use stdout storage to avoid HTTPS validation

	// Create and start the profiler
	p, err := New(config)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Capture stderr to verify logging
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := p.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Create some workload to ensure mutex and block profiles have data
	var testMutex sync.Mutex
	go func() {
		for i := 0; i < 100; i++ {
			testMutex.Lock()
			time.Sleep(time.Microsecond)
			testMutex.Unlock()
			time.Sleep(time.Microsecond)
		}
	}()

	// Create blocking workload
	ch := make(chan bool)
	go func() {
		for i := 0; i < 10; i++ {
			select {
			case <-ch:
			case <-time.After(time.Millisecond):
			}
		}
	}()

	// Let the profiler run and collect samples
	time.Sleep(500 * time.Millisecond)

	// Stop the profiler
	p.Stop()

	// Restore stderr and capture logs
	w.Close()
	os.Stderr = oldStderr
	buf := make([]byte, 2048)
	n, _ := r.Read(buf)
	logs := string(buf[:n])

	// Verify all profile types were started
	expectedProfileTypes := []string{"CPU", "Memory", "Goroutine", "Mutex", "Block"}
	for _, profileType := range expectedProfileTypes {
		expected := fmt.Sprintf("%s profiling started", profileType)
		if !strings.Contains(logs, expected) {
			t.Errorf("Expected log message '%s' not found in output: %s", expected, logs)
		}
	}

	// Verify that all profile types were started
	expectedMessage := "Total profile types started: 7" // Updated: 6 original + 1 HTTP metrics
	if !strings.Contains(logs, expectedMessage) {
		t.Errorf("Expected '%s' not found in logs: %s", expectedMessage, logs)
	} else {
		t.Log("Successfully verified all profile types are started and configured")
	}

	// Verify runtime settings were configured
	expectedRuntimeMessages := []string{
		"Memory profiling enabled with rate:",
		"Mutex profiling enabled with fraction:",
		"Block profiling enabled with rate:",
	}
	for _, expected := range expectedRuntimeMessages {
		if !strings.Contains(logs, expected) {
			t.Errorf("Expected runtime configuration message '%s' not found in logs: %s", expected, logs)
		}
	}
}

// TestDefaultConfigEnablesAllProfiles tests that the updated DefaultConfig enables all profile types
func TestDefaultConfigEnablesAllProfiles(t *testing.T) {
	config := DefaultConfig("test-key", "https://api.pprofio.com", "test-service")

	expectedEnabled := map[string]bool{
		"EnableCPU":       true,
		"EnableMemory":    true,
		"EnableGoroutine": true,
		"EnableMutex":     true,
		"EnableBlock":     true,
	}

	actualEnabled := map[string]bool{
		"EnableCPU":       config.EnableCPU,
		"EnableMemory":    config.EnableMemory,
		"EnableGoroutine": config.EnableGoroutine,
		"EnableMutex":     config.EnableMutex,
		"EnableBlock":     config.EnableBlock,
	}

	for profile, expected := range expectedEnabled {
		if actualEnabled[profile] != expected {
			t.Errorf("DefaultConfig %s = %v, want %v", profile, actualEnabled[profile], expected)
		}
	}
}

// TestRuntimeSettingsConfiguration tests that runtime settings are properly configured
func TestRuntimeSettingsConfiguration(t *testing.T) {
	// Store original settings
	originalMemRate := runtime.MemProfileRate

	config := ComprehensiveConfig("test-key", "https://api.pprofio.com", "test-service")
	config.OutputToStdout = true // Use stdout to avoid network calls

	profiler, err := New(config)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := profiler.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Verify memory profile rate was set
	if runtime.MemProfileRate != config.MemProfileRate {
		t.Errorf("MemProfileRate = %d, want %d", runtime.MemProfileRate, config.MemProfileRate)
	}

	// Stop profiler
	profiler.Stop()

	// Verify original settings were restored
	if runtime.MemProfileRate != originalMemRate {
		t.Errorf("After stop, MemProfileRate = %d, want original %d", runtime.MemProfileRate, originalMemRate)
	}
}

// TestFileStorageJSONResponse tests that FileStorage returns proper JSON responses
func TestFileStorageJSONResponse(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "pprofio-file-storage-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	storage, err := NewFileStorage(tempDir)
	if err != nil {
		t.Fatalf("NewFileStorage() error = %v", err)
	}

	// Create a test profile file
	testFile := filepath.Join(tempDir, "cpu.pprof")
	if err := os.WriteFile(testFile, []byte("test profile data"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Upload the file
	result, err := storage.Upload(context.Background(), testFile)
	if err != nil {
		t.Fatalf("Upload() error = %v", err)
	}

	// Parse the JSON response
	var response map[string]string
	if err := json.Unmarshal([]byte(result), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	// Verify the response structure
	requiredFields := []string{"profile_id", "profile_url", "type"}
	for _, field := range requiredFields {
		if _, exists := response[field]; !exists {
			t.Errorf("JSON response missing required field '%s'", field)
		}
	}

	// Verify the type was correctly inferred
	if response["type"] != "cpu" {
		t.Errorf("Type = %s, want cpu", response["type"])
	}
}
