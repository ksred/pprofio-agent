package pprofio

import (
	"context"
	"testing"
	"time"
)

func TestSpan_End(t *testing.T) {
	t.Parallel()
	
	span := &Span{
		Name:  "test-span",
		Start: time.Now(),
		Tags:  map[string]string{"key": "value"},
	}
	
	// Call End method
	span.End()
	
	// Verify duration is set
	if span.Duration == 0 {
		t.Error("Expected span duration to be set after End()")
	}
	
	// Verify duration is positive
	if span.Duration < 0 {
		t.Error("Expected span duration to be positive")
	}
}

func TestProfiler_processCustomSpans(t *testing.T) {
	// Create a test profiler
	config := Config{
		APIKey:       "test-key",
		IngestURL:    "https://api.pprofio.com",
		SampleRate:   100 * time.Millisecond,
		ServiceName:  "test-service",
		EnableCustom: true,
	}
	
	profiler, err := newProfiler(config)
	if err != nil {
		t.Fatalf("Failed to create profiler: %v", err)
	}
	
	// Set up context and channels
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	
	// Start processCustomSpans in a goroutine
	profiler.wg.Add(1)
	go profiler.processCustomSpans(ctx)
	
	// Send test spans
	testSpan1 := &Span{
		Name:     "test-span-1",
		Start:    time.Now(),
		Duration: 10 * time.Millisecond,
		Tags:     map[string]string{"type": "test"},
	}
	
	testSpan2 := &Span{
		Name:     "test-span-2",
		Start:    time.Now(),
		Duration: 20 * time.Millisecond,
		Tags:     map[string]string{"type": "test"},
	}
	
	// Send spans to the channel
	select {
	case profiler.spanCh <- testSpan1:
	case <-time.After(50 * time.Millisecond):
		t.Error("Failed to send test span 1")
	}
	
	select {
	case profiler.spanCh <- testSpan2:
	case <-time.After(50 * time.Millisecond):
		t.Error("Failed to send test span 2")
	}
	
	// Wait for context to timeout or processing to complete
	<-ctx.Done()
	
	// Close the stop channel to trigger shutdown
	close(profiler.stopCh)
	
	// Wait for goroutine to finish
	profiler.wg.Wait()
}

func TestProfiler_processCustomSpans_StopChannel(t *testing.T) {
	config := Config{
		APIKey:       "test-key",
		IngestURL:    "https://api.pprofio.com",
		SampleRate:   100 * time.Millisecond,
		ServiceName:  "test-service",
		EnableCustom: true,
	}
	
	profiler, err := newProfiler(config)
	if err != nil {
		t.Fatalf("Failed to create profiler: %v", err)
	}
	
	ctx := context.Background()
	
	// Start processCustomSpans in a goroutine
	profiler.wg.Add(1)
	go profiler.processCustomSpans(ctx)
	
	// Close stop channel immediately
	close(profiler.stopCh)
	
	// Wait for goroutine to finish
	profiler.wg.Wait()
}

func TestProfiler_processCustomSpans_ContextDone(t *testing.T) {
	config := Config{
		APIKey:       "test-key",
		IngestURL:    "https://api.pprofio.com",
		SampleRate:   100 * time.Millisecond,
		ServiceName:  "test-service",
		EnableCustom: true,
	}
	
	profiler, err := newProfiler(config)
	if err != nil {
		t.Fatalf("Failed to create profiler: %v", err)
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	// Start processCustomSpans in a goroutine
	profiler.wg.Add(1)
	go profiler.processCustomSpans(ctx)
	
	// Cancel context immediately
	cancel()
	
	// Wait for goroutine to finish
	profiler.wg.Wait()
}

func TestProfiler_processSpans(t *testing.T) {
	config := Config{
		APIKey:       "test-key",
		IngestURL:    "https://api.pprofio.com",
		SampleRate:   100 * time.Millisecond,
		ServiceName:  "test-service",
		EnableCustom: true,
	}
	
	profiler, err := newProfiler(config)
	if err != nil {
		t.Fatalf("Failed to create profiler: %v", err)
	}
	
	ctx := context.Background()
	
	// Create test spans map
	spans := map[string][]*Span{
		"test-operation": {
			{
				Name:     "test-operation",
				Start:    time.Now(),
				Duration: 10 * time.Millisecond,
				Tags:     map[string]string{"endpoint": "/api/test"},
			},
			{
				Name:     "test-operation",
				Start:    time.Now(),
				Duration: 15 * time.Millisecond,
				Tags:     map[string]string{"endpoint": "/api/test2"},
			},
		},
		"another-operation": {
			{
				Name:     "another-operation",
				Start:    time.Now(),
				Duration: 5 * time.Millisecond,
				Tags:     map[string]string{"endpoint": "/api/other"},
			},
		},
	}
	
	// Call processSpans (this is a placeholder implementation, so it should return nil)
	err = profiler.processSpans(ctx, spans)
	if err != nil {
		t.Errorf("processSpans() error = %v, want nil", err)
	}
}

func TestProfiler_processCustomSpans_FlushTicker(t *testing.T) {
	config := Config{
		APIKey:       "test-key",
		IngestURL:    "https://api.pprofio.com",
		SampleRate:   50 * time.Millisecond, // Short interval for faster testing
		ServiceName:  "test-service",
		EnableCustom: true,
	}
	
	profiler, err := newProfiler(config)
	if err != nil {
		t.Fatalf("Failed to create profiler: %v", err)
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	
	// Start processCustomSpans in a goroutine
	profiler.wg.Add(1)
	go profiler.processCustomSpans(ctx)
	
	// Send a test span
	testSpan := &Span{
		Name:     "flush-test-span",
		Start:    time.Now(),
		Duration: 10 * time.Millisecond,
		Tags:     map[string]string{"test": "flush"},
	}
	
	select {
	case profiler.spanCh <- testSpan:
	case <-time.After(25 * time.Millisecond):
		t.Error("Failed to send test span")
	}
	
	// Wait for the flush ticker to trigger at least once
	time.Sleep(80 * time.Millisecond)
	
	// Wait for context to timeout
	<-ctx.Done()
	
	// Close the stop channel and wait for cleanup
	close(profiler.stopCh)
	profiler.wg.Wait()
}