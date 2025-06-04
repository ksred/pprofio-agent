package pprofio

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewProfiler(t *testing.T) {
	t.Parallel()
	// Create a temporary directory for storage
	tempDir, err := os.MkdirTemp("", "pprofio-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create file storage
	storage, err := NewFileStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	// Create valid config
	config := Config{
		APIKey:          "test-key",
		IngestURL:       "https://api.pprofio.com",
		SampleRate:      5 * time.Second,
		ProfileDuration: 1 * time.Second,
		Storage:         storage,
		ServiceName:     "test-service",
		Tags:            map[string]string{"env": "test"},
		EnableCPU:       true,
		EnableMemory:    true,
	}

	// Test creating a new profiler
	p, err := newProfiler(config)
	if err != nil {
		t.Fatalf("newProfiler() error = %v", err)
	}

	if p == nil {
		t.Fatal("newProfiler() returned nil profiler")
	}

	if p.config.APIKey != config.APIKey {
		t.Errorf("profiler.config.APIKey = %v, want %v", p.config.APIKey, config.APIKey)
	}

	if p.config.ServiceName != config.ServiceName {
		t.Errorf("profiler.config.ServiceName = %v, want %v", p.config.ServiceName, config.ServiceName)
	}

	// Test with invalid config
	invalidConfig := Config{
		// Missing required fields
	}

	_, err = newProfiler(invalidConfig)
	if err == nil {
		t.Error("newProfiler() with invalid config should return error")
	}
}

func TestProfilerLifecycle(t *testing.T) {
	// Create a temporary directory for storage
	tempDir, err := os.MkdirTemp("", "pprofio-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create file storage
	storage, err := NewFileStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	// Create a mock HTTP server for metadata
	metadataServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer metadataServer.Close()

	// Create config with short intervals for faster testing
	config := Config{
		APIKey:          "test-key",
		IngestURL:       metadataServer.URL,
		SampleRate:      30 * time.Millisecond,
		ProfileDuration: 10 * time.Millisecond,
		Storage:         storage,
		ServiceName:     "test-service",
		Tags:            map[string]string{"env": "test"},
		EnableCPU:       true,
		EnableMemory:    true,
	}

	// Create a new profiler
	p, err := New(config)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Start the profiler with shorter duration
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Millisecond)
	defer cancel()

	if err := p.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Check initialized state
	if !p.initialized {
		t.Error("Start() did not set initialized to true")
	}

	// Let it run and collect profiles (shorter time)
	time.Sleep(50 * time.Millisecond)

	// Stop the profiler
	p.Stop()

	// Check stopped state
	if p.initialized {
		t.Error("Stop() did not set initialized to false")
	}

	// Test starting again after stopping
	ctx2 := context.Background()
	if err := p.Start(ctx2); err != nil {
		t.Fatalf("Start() after Stop() error = %v", err)
	}

	// Stop immediately
	p.Stop()
}

func TestCollectProfiles(t *testing.T) {
	// Create a temporary directory for storage
	tempDir, err := os.MkdirTemp("", "pprofio-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create file storage
	storage, err := NewFileStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	
	// Create a mock HTTP server for metadata
	metadataServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer metadataServer.Close()

	// Test each profile type individually
	testCases := []struct {
		name       string
		profileKey profileType
		enableFn   func(config *Config)
	}{
		{
			name:       "CPU",
			profileKey: profileTypeCPU,
			enableFn: func(config *Config) {
				config.EnableCPU = true
			},
		},
		{
			name:       "Memory",
			profileKey: profileTypeMemory,
			enableFn: func(config *Config) {
				config.EnableMemory = true
			},
		},
		{
			name:       "Goroutine",
			profileKey: profileTypeGoroutine,
			enableFn: func(config *Config) {
				config.EnableGoroutine = true
			},
		},
		{
			name:       "Mutex",
			profileKey: profileTypeMutex,
			enableFn: func(config *Config) {
				config.EnableMutex = true
			},
		},
		{
			name:       "Block",
			profileKey: profileTypeBlock,
			enableFn: func(config *Config) {
				config.EnableBlock = true
			},
		},
	}

	for _, tc := range testCases {
		tc := tc // Capture range variable
		t.Run(tc.name, func(t *testing.T) {
			// Clean the directory
			files, _ := os.ReadDir(tempDir)
			for _, file := range files {
				os.Remove(filepath.Join(tempDir, file.Name()))
			}

			// Create config with only the specific profile type enabled
			config := Config{
				APIKey:          "test-key",
				IngestURL:       metadataServer.URL,
				SampleRate:      30 * time.Millisecond,
				ProfileDuration: 10 * time.Millisecond,
				Storage:         storage,
				ServiceName:     "test-service",
				Tags:            map[string]string{"env": "test"},
				EnableCPU:       false,
				EnableMemory:    false,
				EnableGoroutine: false,
				EnableMutex:     false,
				EnableBlock:     false,
				EnableCustom:    false,
			}

			// Enable only the profile type we're testing
			tc.enableFn(&config)

			// Create and start the profiler
			p, err := New(config)
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
			defer cancel()

			if err := p.Start(ctx); err != nil {
				t.Fatalf("Start() error = %v", err)
			}

			// Let it run long enough to collect at least one profile
			time.Sleep(25 * time.Millisecond)

			// Stop the profiler
			p.Stop()

			// Give it a moment to finalize any in-progress operations
			time.Sleep(5 * time.Millisecond)
		})
	}
}

func TestAPIUsage(t *testing.T) {
	// Test the main package API as a user would use it
	
	// Create a temporary directory for storage
	tempDir, err := os.MkdirTemp("", "pprofio-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create file storage
	storage, err := NewFileStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	
	// Create a mock HTTP server for metadata
	metadataServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer metadataServer.Close()

	// Use the default config with overrides
	cfg := DefaultConfig("test-key", metadataServer.URL, "test-service")
	cfg.Storage = storage
	cfg.SampleRate = 20 * time.Millisecond
	cfg.ProfileDuration = 10 * time.Millisecond
	
	// Create the profiler
	p, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	// Start profiling
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer cancel()
	
	if err := p.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Add profiler to context
	ctx = WithProfiler(ctx, p)

	// Create a span
	_, span := StartSpan(ctx, "test-operation", "endpoint", "/api/test")
	time.Sleep(5 * time.Millisecond)
	span.End()

	// Let the profiler run
	time.Sleep(30 * time.Millisecond)

	// Stop profiling
	p.Stop()

	// Give it a moment to finalize
	time.Sleep(5 * time.Millisecond)
}