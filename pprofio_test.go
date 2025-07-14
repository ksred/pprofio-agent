package pprofio

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("ValidConfig", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "pprofio-test")
		if err != nil {
			t.Fatalf("Failed to create temp directory: %v", err)
		}
		defer os.RemoveAll(tempDir)

		storage, err := NewFileStorage(tempDir)
		if err != nil {
			t.Fatalf("Failed to create storage: %v", err)
		}

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

		profiler, err := New(config)
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}

		if profiler == nil {
			t.Fatal("New() returned nil profiler")
		}

		if profiler.config.ServiceName != config.ServiceName {
			t.Errorf("Expected service name %v, got %v", config.ServiceName, profiler.config.ServiceName)
		}
	})

	t.Run("InvalidConfig", func(t *testing.T) {
		config := Config{
			// Missing required fields
		}

		profiler, err := New(config)
		if err == nil {
			t.Error("New() with invalid config should return error")
		}

		if profiler != nil {
			t.Error("New() with invalid config should return nil profiler")
		}
	})

	t.Run("ConfigWithMissingStorageAndCredentials", func(t *testing.T) {
		config := Config{
			SampleRate:      5 * time.Second,
			ProfileDuration: 1 * time.Second,
			ServiceName:     "test-service",
			OutputToStdout:  false, // Explicitly set to false so Storage is required
			// Missing Storage, APIKey, and IngestURL
		}

		profiler, err := New(config)
		if err == nil {
			t.Error("New() with missing storage and credentials should return error")
		}

		if profiler != nil {
			t.Error("New() with missing storage and credentials should return nil profiler")
		}
	})
}

func TestProfiler_Start(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "pprofio-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	storage, err := NewFileStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

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
		Tags:            map[string]string{"env": "test"},
		EnableCPU:       true,
		EnableMemory:    true,
		EnableGoroutine: true,
		EnableMutex:     true,
		EnableBlock:     true,
		EnableCustom:    true,
	}

	profiler, err := New(config)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	t.Run("StartSuccess", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		err := profiler.Start(ctx)
		if err != nil {
			t.Fatalf("Start() error = %v", err)
		}

		if !profiler.initialized {
			t.Error("Expected profiler to be initialized after Start()")
		}

		// Stop to clean up
		profiler.Stop()
	})

	t.Run("StartAlreadyInitialized", func(t *testing.T) {
		ctx := context.Background()

		// Start first time
		err := profiler.Start(ctx)
		if err != nil {
			t.Fatalf("First Start() error = %v", err)
		}

		// Try to start again while already running
		err = profiler.Start(ctx)
		if err == nil {
			t.Error("Start() on already initialized profiler should return error")
		}

		// Stop to clean up
		profiler.Stop()
	})
}

func TestProfiler_Stop(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "pprofio-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	storage, err := NewFileStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

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
		Tags:            map[string]string{"env": "test"},
		EnableCPU:       true,
		EnableMemory:    true,
	}

	profiler, err := New(config)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	t.Run("StopAfterStart", func(t *testing.T) {
		ctx := context.Background()

		// Start the profiler
		err := profiler.Start(ctx)
		if err != nil {
			t.Fatalf("Start() error = %v", err)
		}

		// Verify it's initialized
		if !profiler.initialized {
			t.Error("Expected profiler to be initialized after Start()")
		}

		// Stop the profiler
		profiler.Stop()

		// Verify it's no longer initialized
		if profiler.initialized {
			t.Error("Expected profiler to not be initialized after Stop()")
		}
	})

	t.Run("StopWithoutStart", func(t *testing.T) {
		// This should not panic or cause issues
		profiler.Stop()
	})
}

func TestStartSpan(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "pprofio-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	storage, err := NewFileStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	config := Config{
		APIKey:          "test-key",
		IngestURL:       "https://api.pprofio.com",
		SampleRate:      50 * time.Millisecond,
		ProfileDuration: 10 * time.Millisecond,
		Storage:         storage,
		ServiceName:     "test-service",
		EnableCustom:    true,
	}

	profiler, err := New(config)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	t.Run("StartSpanWithProfiler", func(t *testing.T) {
		ctx := WithProfiler(context.Background(), profiler)

		resultCtx, span := StartSpan(ctx, "test-operation", "endpoint", "/api/test")

		// Verify span is created
		if span == nil {
			t.Fatal("StartSpan() returned nil span")
		}

		if span.Name != "test-operation" {
			t.Errorf("Expected span name 'test-operation', got '%s'", span.Name)
		}

		if span.Tags["endpoint"] != "/api/test" {
			t.Errorf("Expected endpoint tag '/api/test', got '%s'", span.Tags["endpoint"])
		}

		// Verify context is returned
		if resultCtx == nil {
			t.Fatal("StartSpan() returned nil context")
		}

		// End the span
		span.End()

		// Verify duration is set
		if span.Duration == 0 {
			t.Error("Expected span duration to be set after End()")
		}
	})

	t.Run("StartSpanWithoutProfiler", func(t *testing.T) {
		ctx := context.Background() // No profiler in context

		resultCtx, span := StartSpan(ctx, "test-operation", "endpoint", "/api/test")

		// Should still return a span, but it won't be processed
		if span == nil {
			t.Fatal("StartSpan() returned nil span")
		}

		if resultCtx == nil {
			t.Fatal("StartSpan() returned nil context")
		}
	})
}

func TestWithProfiler(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "pprofio-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	storage, err := NewFileStorage(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	config := Config{
		APIKey:          "test-key",
		IngestURL:       "https://api.pprofio.com",
		SampleRate:      50 * time.Millisecond,
		ProfileDuration: 10 * time.Millisecond,
		Storage:         storage,
		ServiceName:     "test-service",
	}

	profiler, err := New(config)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	t.Run("AddProfilerToContext", func(t *testing.T) {
		ctx := context.Background()

		// Add profiler to context
		ctxWithProfiler := WithProfiler(ctx, profiler)

		// Verify we can retrieve the profiler
		retrievedProfiler := ctxWithProfiler.Value(spanKey{})
		if retrievedProfiler == nil {
			t.Error("Failed to retrieve profiler from context")
		}

		if retrievedProfiler != profiler {
			t.Error("Retrieved profiler is not the same as the original")
		}
	})
}
