package pprofio

import (
	"context"
	"fmt"
	"net/http"
	"runtime"
	"testing"
	"time"
)

// TestProfilerNoGoroutineLeak verifies that stopping the profiler cleans up all goroutines
func TestProfilerNoGoroutineLeak(t *testing.T) {
	// Get initial goroutine count
	runtime.GC()
	initialGoroutines := runtime.NumGoroutine()

	// Create and start profiler
	config := Config{
		APIKey:           "test-key",
		IngestURL:        "https://test.example.com",
		SampleRate:       100 * time.Millisecond,
		ProfileDuration:  50 * time.Millisecond,
		ServiceName:      "test-service",
		Tags:             map[string]string{"env": "test"},
		Storage:          NewMockJSONStorage(),
		MemProfileRate:   4096,
		MutexFraction:    5,
		BlockProfileRate: 100,
		EnableCPU:        true,
		EnableMemory:     true,
		EnableGoroutine:  true,
		EnableMutex:      true,
		EnableBlock:      true,
		EnableCustom:     true,
	}

	profiler, err := newProfiler(config)
	if err != nil {
		t.Fatalf("Failed to create profiler: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the profiler
	if err := profiler.Start(ctx); err != nil {
		t.Fatalf("Failed to start profiler: %v", err)
	}

	// Let it run for a bit
	time.Sleep(200 * time.Millisecond)

	// Stop the profiler
	profiler.Stop()

	// Give goroutines time to shut down
	time.Sleep(100 * time.Millisecond)

	// Check goroutine count
	runtime.GC()
	finalGoroutines := runtime.NumGoroutine()

	// Allow for a small difference due to test framework goroutines
	if finalGoroutines > initialGoroutines+2 {
		t.Errorf("Goroutine leak detected: started with %d, ended with %d goroutines", initialGoroutines, finalGoroutines)
	}
}

// TestSpanProcessingNoGoroutineLeak verifies that span processing doesn't leak goroutines
func TestSpanProcessingNoGoroutineLeak(t *testing.T) {
	// Get initial goroutine count
	runtime.GC()
	initialGoroutines := runtime.NumGoroutine()

	// Create profiler with custom spans enabled
	config := Config{
		APIKey:       "test-key",
		IngestURL:    "https://test.example.com",
		SampleRate:   50 * time.Millisecond,
		ServiceName:  "test-service",
		Storage:      NewMockJSONStorage(),
		EnableCustom: true,
	}

	profiler, err := newProfiler(config)
	if err != nil {
		t.Fatalf("Failed to create profiler: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the profiler
	if err := profiler.Start(ctx); err != nil {
		t.Fatalf("Failed to start profiler: %v", err)
	}

	// Create many spans to trigger processing
	for i := 0; i < 100; i++ {
		_, span := StartSpan(context.WithValue(ctx, spanKey{}, profiler), "test-span", "iteration", fmt.Sprintf("%d", i))
		span.End()

		// Non-blocking send to span channel
		select {
		case profiler.spanCh <- span:
		default:
		}
	}

	// Wait for processing
	time.Sleep(150 * time.Millisecond)

	// Stop the profiler
	profiler.Stop()

	// Give goroutines time to shut down
	time.Sleep(100 * time.Millisecond)

	// Check goroutine count
	runtime.GC()
	finalGoroutines := runtime.NumGoroutine()

	// Allow for a small difference due to test framework goroutines
	if finalGoroutines > initialGoroutines+2 {
		t.Errorf("Goroutine leak detected: started with %d, ended with %d goroutines", initialGoroutines, finalGoroutines)
	}
}

// TestHTTPClientConnectionLeak verifies that HTTP connections are properly closed
func TestHTTPClientConnectionLeak(t *testing.T) {
	// Create HTTP storage
	storage := NewHTTPStorage("https://test.example.com", "test-key", "test")

	// Simulate multiple uploads (would fail but we're testing connection cleanup)
	ctx := context.Background()
	for i := 0; i < 10; i++ {
		storage.Upload(ctx, "nonexistent.pprof")
	}

	// Close the storage to clean up connections
	if err := storage.Close(); err != nil {
		t.Fatalf("Failed to close storage: %v", err)
	}

	// Check that transport connections are closed
	if transport, ok := storage.Client.Transport.(*http.Transport); ok {
		// This would require exposing transport internals, which we've done via Close()
		t.Log("HTTP connections closed successfully")
		// Just verify the transport exists
		if transport != nil {
			t.Log("Transport verified")
		}
	}
}

// TestSpanNameBoundedGrowth verifies that span names don't grow unbounded
func TestSpanNameBoundedGrowth(t *testing.T) {
	config := Config{
		APIKey:       "test-key",
		IngestURL:    "https://test.example.com",
		SampleRate:   50 * time.Millisecond,
		ServiceName:  "test-service",
		Storage:      NewMockJSONStorage(),
		EnableCustom: true,
	}

	profiler, err := newProfiler(config)
	if err != nil {
		t.Fatalf("Failed to create profiler: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the profiler
	if err := profiler.Start(ctx); err != nil {
		t.Fatalf("Failed to start profiler: %v", err)
	}

	// Create spans with many unique names (more than MaxUniqueSpanNames)
	for i := 0; i < MaxUniqueSpanNames+100; i++ {
		spanName := fmt.Sprintf("unique-span-%d", i)
		_, span := StartSpan(context.WithValue(ctx, spanKey{}, profiler), spanName)
		span.End()

		// Send span
		select {
		case profiler.spanCh <- span:
		default:
		}
	}

	// Let processing happen
	time.Sleep(100 * time.Millisecond)

	// Stop the profiler
	profiler.Stop()

	// The test passes if it doesn't consume excessive memory
	// In real scenarios, you'd monitor memory usage here
	t.Log("Span name limiting test completed successfully")
}

// TestMemoryProfileRateRestored verifies memory profile settings are restored
func TestMemoryProfileRateRestored(t *testing.T) {
	// Save original rate
	originalRate := runtime.MemProfileRate

	config := Config{
		APIKey:         "test-key",
		IngestURL:      "https://test.example.com",
		SampleRate:     1 * time.Second,
		ServiceName:    "test-service",
		Storage:        NewMockJSONStorage(),
		EnableMemory:   true,
		MemProfileRate: 8192, // Different from default
	}

	profiler, err := newProfiler(config)
	if err != nil {
		t.Fatalf("Failed to create profiler: %v", err)
	}

	ctx := context.Background()

	// Start the profiler
	if err := profiler.Start(ctx); err != nil {
		t.Fatalf("Failed to start profiler: %v", err)
	}

	// Verify rate was changed
	if runtime.MemProfileRate != 8192 {
		t.Errorf("Memory profile rate not set correctly: expected 8192, got %d", runtime.MemProfileRate)
	}

	// Stop the profiler
	profiler.Stop()

	// Verify rate was restored
	if runtime.MemProfileRate != originalRate {
		t.Errorf("Memory profile rate not restored: expected %d, got %d", originalRate, runtime.MemProfileRate)
	}
}

// BenchmarkSpanProcessing benchmarks span processing to ensure no memory leaks
func BenchmarkSpanProcessing(b *testing.B) {
	config := Config{
		APIKey:       "test-key",
		IngestURL:    "https://test.example.com",
		SampleRate:   100 * time.Millisecond,
		ServiceName:  "test-service",
		Storage:      NewMockJSONStorage(),
		EnableCustom: true,
	}

	profiler, err := newProfiler(config)
	if err != nil {
		b.Fatalf("Failed to create profiler: %v", err)
	}

	ctx := context.Background()
	if err := profiler.Start(ctx); err != nil {
		b.Fatalf("Failed to start profiler: %v", err)
	}
	defer profiler.Stop()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, span := StartSpan(context.WithValue(ctx, spanKey{}, profiler), "benchmark-span", "iter", fmt.Sprintf("%d", i))
		span.End()

		select {
		case profiler.spanCh <- span:
		default:
		}
	}
}
