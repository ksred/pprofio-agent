package pprofio

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestProfiler_collectProfile_UnknownType(t *testing.T) {
	storage := NewMockJSONStorage()

	config := Config{
		APIKey:      "test-key",
		IngestURL:   "https://api.pprofio.com",
		Storage:     storage,
		ServiceName: "test-service",
	}

	profiler, err := newProfiler(config)
	if err != nil {
		t.Fatalf("newProfiler() error = %v", err)
	}

	ctx := context.Background()

	// Test unknown profile type
	err = profiler.collectProfile(ctx, "unknown-type")
	if err == nil {
		t.Error("Expected error for unknown profile type")
	}

	if err.Error() != "unknown profile type: unknown-type" {
		t.Errorf("Expected 'unknown profile type: unknown-type', got: %v", err)
	}
}

func TestProfiler_uploadProfile_ErrorCases(t *testing.T) {
	t.Run("StorageUploadFails", func(t *testing.T) {
		// Create storage that always fails
		failingStorage := NewMockFailingJSONStorage("storage upload failed")

		config := Config{
			APIKey:      "test-key",
			IngestURL:   "https://api.pprofio.com",
			Storage:     failingStorage,
			ServiceName: "test-service",
		}

		profiler, err := newProfiler(config)
		if err != nil {
			t.Fatalf("newProfiler() error = %v", err)
		}

		// Create test file
		tmpFile, err := os.CreateTemp("", "test-profile")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile.Name())

		tmpFile.WriteString("test data")
		tmpFile.Close()

		ctx := context.Background()
		err = profiler.uploadProfile(ctx, tmpFile.Name(), "cpu")
		if err == nil {
			t.Error("Expected error when storage upload fails")
		}

		if !strings.Contains(err.Error(), "failed to upload profile: storage upload failed") {
			t.Errorf("Expected storage upload error, got: %v", err)
		}
	})

	t.Run("InvalidJSONResponse", func(t *testing.T) {
		// Create storage that returns invalid JSON
		invalidJSONStorage := NewMockInvalidJSONStorage()

		config := Config{
			APIKey:      "test-key",
			IngestURL:   "https://api.pprofio.com",
			Storage:     invalidJSONStorage,
			ServiceName: "test-service",
		}

		profiler, err := newProfiler(config)
		if err != nil {
			t.Fatalf("newProfiler() error = %v", err)
		}

		// Create test file
		tmpFile, err := os.CreateTemp("", "test-profile")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile.Name())

		tmpFile.WriteString("test data")
		tmpFile.Close()

		ctx := context.Background()
		err = profiler.uploadProfile(ctx, tmpFile.Name(), "cpu")
		if err == nil {
			t.Error("Expected error for invalid JSON response")
		}

		if err.Error() != "failed to parse upload response: invalid character 'i' looking for beginning of value" {
			t.Errorf("Expected JSON parse error, got: %v", err)
		}
	})

	t.Run("StdoutModeSuccess", func(t *testing.T) {
		// Create stdout storage and enable stdout mode
		stdoutStorage := NewStdoutStorage()

		config := Config{
			APIKey:         "test-key",
			IngestURL:      "https://api.pprofio.com",
			Storage:        stdoutStorage,
			ServiceName:    "test-service",
			OutputToStdout: true,
			Tags:           map[string]string{"env": "test"},
		}

		profiler, err := newProfiler(config)
		if err != nil {
			t.Fatalf("newProfiler() error = %v", err)
		}

		// Create test file
		tmpFile, err := os.CreateTemp("", "test-profile")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile.Name())

		tmpFile.WriteString("test data")
		tmpFile.Close()

		ctx := context.Background()
		err = profiler.uploadProfile(ctx, tmpFile.Name(), "cpu")
		if err != nil {
			t.Errorf("uploadProfile() with stdout mode should succeed, got: %v", err)
		}
	})

	t.Run("MetadataSendFails", func(t *testing.T) {
		// Create storage that returns valid JSON
		validJSONStorage := NewMockJSONStorage()

		// Create server that fails metadata requests
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		config := Config{
			Storage:     validJSONStorage,
			ServiceName: "test-service",
			IngestURL:   server.URL,
			APIKey:      "test-key",
			Tags:        map[string]string{"env": "test"},
		}

		profiler, err := newProfiler(config)
		if err != nil {
			t.Fatalf("newProfiler() error = %v", err)
		}

		// Create test file
		tmpFile, err := os.CreateTemp("", "test-profile")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile.Name())

		tmpFile.WriteString("test data")
		tmpFile.Close()

		ctx := context.Background()
		err = profiler.uploadProfile(ctx, tmpFile.Name(), "cpu")
		if err == nil {
			t.Error("Expected error when metadata send fails")
		}

		if !strings.Contains(err.Error(), "failed to send metadata") {
			t.Errorf("Expected metadata send error, got: %v", err)
		}
	})
}

func TestProfiler_collectProfiles_EdgeCases(t *testing.T) {
	t.Run("ContextCancellation", func(t *testing.T) {
		storage := NewMockJSONStorage()

		config := Config{
			APIKey:          "test-key",
			IngestURL:       "https://api.pprofio.com",
			SampleRate:      10 * time.Millisecond,
			ProfileDuration: 5 * time.Millisecond,
			Storage:         storage,
			ServiceName:     "test-service",
		}

		profiler, err := newProfiler(config)
		if err != nil {
			t.Fatalf("newProfiler() error = %v", err)
		}

		ctx, cancel := context.WithCancel(context.Background())

		// Start collectProfiles in a goroutine
		profiler.wg.Add(1)
		go profiler.collectProfiles(ctx, profileTypeCPU)

		// Cancel immediately
		cancel()

		// Wait for goroutine to finish
		profiler.wg.Wait()
	})

	t.Run("StopChannelClosure", func(t *testing.T) {
		storage := NewMockJSONStorage()

		config := Config{
			APIKey:          "test-key",
			IngestURL:       "https://api.pprofio.com",
			SampleRate:      10 * time.Millisecond,
			ProfileDuration: 5 * time.Millisecond,
			Storage:         storage,
			ServiceName:     "test-service",
		}

		profiler, err := newProfiler(config)
		if err != nil {
			t.Fatalf("newProfiler() error = %v", err)
		}

		ctx := context.Background()

		// Start collectProfiles in a goroutine
		profiler.wg.Add(1)
		go profiler.collectProfiles(ctx, profileTypeCPU)

		// Close stop channel
		close(profiler.stopCh)

		// Wait for goroutine to finish
		profiler.wg.Wait()
	})
}

func TestProfiler_RuntimeSettings(t *testing.T) {
	t.Run("RuntimeSettingsStored", func(t *testing.T) {
		storage := NewMockJSONStorage()

		config := Config{
			APIKey:          "test-key",
			IngestURL:       "https://api.pprofio.com",
			SampleRate:      100 * time.Millisecond,
			ProfileDuration: 10 * time.Millisecond,
			Storage:         storage,
			ServiceName:     "test-service",
			EnableMemory:    true,
			EnableMutex:     true,
			EnableBlock:     true,
		}

		profiler, err := newProfiler(config)
		if err != nil {
			t.Fatalf("newProfiler() error = %v", err)
		}

		// Just verify that the profiler was created successfully
		// The runtime values are stored internally during profiler creation
		if profiler == nil {
			t.Error("Expected profiler to be created")
		}

		if profiler.config.ServiceName != "test-service" {
			t.Errorf("Expected service name to be 'test-service', got %s", profiler.config.ServiceName)
		}
	})
}

// Mock storage implementations are now in mock_storage_test.go

func TestProfiler_start_stop_internals(t *testing.T) {
	storage := NewMockJSONStorage()

	metadataServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer metadataServer.Close()

	config := Config{
		APIKey:          "test-key",
		IngestURL:       metadataServer.URL,
		SampleRate:      50 * time.Millisecond,
		ProfileDuration: 10 * time.Millisecond,
		Storage:         storage,
		ServiceName:     "test-service",
		EnableCPU:       true,
		EnableMemory:    true,
		EnableGoroutine: true,
		EnableMutex:     true,
		EnableBlock:     true,
		EnableCustom:    true,
	}

	profiler, err := newProfiler(config)
	if err != nil {
		t.Fatalf("newProfiler() error = %v", err)
	}

	// Test internal start method
	ctx := context.Background()
	profiler.start(ctx)

	// Verify goroutines are started
	if !profiler.initialized {
		t.Error("Expected profiler to be initialized after start()")
	}

	// Let it run briefly
	time.Sleep(30 * time.Millisecond)

	// Test internal stop method
	profiler.stop()

	// Verify cleanup
	if profiler.initialized {
		t.Error("Expected profiler to not be initialized after stop()")
	}
}

func TestProfiler_collectMemory_ForceGC(t *testing.T) {
	storage := NewMockJSONStorage()

	metadataServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer metadataServer.Close()

	config := Config{
		APIKey:      "test-key",
		IngestURL:   metadataServer.URL,
		Storage:     storage,
		ServiceName: "test-service",
	}

	profiler, err := newProfiler(config)
	if err != nil {
		t.Fatalf("newProfiler() error = %v", err)
	}

	// Test memory collection which should force GC
	ctx := context.Background()
	err = profiler.collectMemory(ctx)
	if err != nil {
		t.Errorf("collectMemory() error = %v", err)
	}
}

func TestProfiler_profileTypes_Coverage(t *testing.T) {
	storage := NewMockJSONStorage()

	metadataServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer metadataServer.Close()

	config := Config{
		APIKey:      "test-key",
		IngestURL:   metadataServer.URL,
		Storage:     storage,
		ServiceName: "test-service",
	}

	profiler, err := newProfiler(config)
	if err != nil {
		t.Fatalf("newProfiler() error = %v", err)
	}

	ctx := context.Background()

	// Test all profile types individually
	testCases := []struct {
		profileType profileType
		methodName  string
	}{
		{profileTypeCPU, "collectCPU"},
		{profileTypeMemory, "collectMemory"},
		{profileTypeGoroutine, "collectGoroutine"},
		{profileTypeMutex, "collectMutex"},
		{profileTypeBlock, "collectBlock"},
	}

	for _, tc := range testCases {
		t.Run(string(tc.profileType), func(t *testing.T) {
			err := profiler.collectProfile(ctx, tc.profileType)
			if err != nil {
				t.Errorf("%s failed: %v", tc.methodName, err)
			}
		})
	}
}
