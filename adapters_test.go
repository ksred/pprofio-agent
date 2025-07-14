package pprofio

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test framework-agnostic middleware interface
func TestMiddlewareAdapter_Interface(t *testing.T) {
	config := MiddlewareConfig{
		Enabled:          true,
		SampleRate:       1.0,
		CollectUserAgent: true,
	}

	// Test that our MetricsCollector implements the MiddlewareAdapter interface
	collector := NewMetricsCollector(config)

	// Should be able to get different adapter types
	adapter := NewMiddlewareAdapter(collector)
	require.NotNil(t, adapter)

	// Test standard HTTP adapter
	httpAdapter := adapter.ForHTTP()
	require.NotNil(t, httpAdapter)

	// Test that the adapter can wrap handlers
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	})

	wrappedHandler := httpAdapter(testHandler)
	require.NotNil(t, wrappedHandler)

	// Verify the wrapped handler works
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "test response", w.Body.String())
}

// Test middleware context and request ID propagation
func TestMiddlewareAdapter_ContextPropagation(t *testing.T) {
	config := MiddlewareConfig{
		Enabled:    true,
		SampleRate: 1.0,
	}

	collector := NewMetricsCollector(config)
	var capturedMetrics *RequestMetrics

	collector.SetMetricsHandler(func(metrics *RequestMetrics) {
		capturedMetrics = metrics
	})

	adapter := NewMiddlewareAdapter(collector)
	httpAdapter := adapter.ForHTTP()

	// Handler that checks for request ID in context
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if request ID is available in context
		requestID := GetRequestIDFromContext(r.Context())
		assert.NotEmpty(t, requestID)

		w.Header().Set("X-Request-ID", requestID)
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := httpAdapter(testHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, w.Header().Get("X-Request-ID"))

	// Verify metrics were captured
	require.NotNil(t, capturedMetrics)
	assert.Equal(t, w.Header().Get("X-Request-ID"), capturedMetrics.RequestID)
}

// Test adapter with custom tags
func TestMiddlewareAdapter_CustomTags(t *testing.T) {
	config := MiddlewareConfig{
		Enabled:    true,
		SampleRate: 1.0,
	}

	collector := NewMetricsCollector(config)
	var capturedMetrics *RequestMetrics

	collector.SetMetricsHandler(func(metrics *RequestMetrics) {
		capturedMetrics = metrics
	})

	adapter := NewMiddlewareAdapter(collector)

	// Add custom tags including common fields that should be moved to top-level
	adapter.WithTags(map[string]string{
		"service":     "test-service", // Should move to top-level
		"version":     "1.0.0",        // Should move to top-level
		"environment": "test",         // Should move to top-level
		"region":      "us-west-1",    // Should move to top-level
		"team":        "backend",      // Should remain in tags
		"component":   "auth",         // Should remain in tags
	})

	httpAdapter := adapter.ForHTTP()

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := httpAdapter(testHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w, req)

	// Verify common fields were moved to top-level
	require.NotNil(t, capturedMetrics)
	assert.Equal(t, "test-service", capturedMetrics.Service)
	assert.Equal(t, "1.0.0", capturedMetrics.Version)
	assert.Equal(t, "test", capturedMetrics.Environment)
	assert.Equal(t, "us-west-1", capturedMetrics.Region)

	// Verify remaining tags stayed in tags field
	assert.Equal(t, "backend", capturedMetrics.Tags["team"])
	assert.Equal(t, "auth", capturedMetrics.Tags["component"])

	// Verify common fields are NOT in tags field
	assert.NotContains(t, capturedMetrics.Tags, "service")
	assert.NotContains(t, capturedMetrics.Tags, "version")
	assert.NotContains(t, capturedMetrics.Tags, "environment")
	assert.NotContains(t, capturedMetrics.Tags, "region")

	// Verify duration_ns is set correctly
	assert.Greater(t, capturedMetrics.DurationNs, int64(0))
	assert.Equal(t, capturedMetrics.Duration.Nanoseconds(), capturedMetrics.DurationNs)
}

// Test adapter with route pattern extraction
func TestMiddlewareAdapter_RoutePatterns(t *testing.T) {
	config := MiddlewareConfig{
		Enabled:    true,
		SampleRate: 1.0,
	}

	collector := NewMetricsCollector(config)
	var capturedMetrics *RequestMetrics

	collector.SetMetricsHandler(func(metrics *RequestMetrics) {
		capturedMetrics = metrics
	})

	adapter := NewMiddlewareAdapter(collector)

	// Configure route pattern extractor
	adapter.WithRouteExtractor(func(r *http.Request) string {
		// Simple pattern matching for test
		path := r.URL.Path
		if path == "/users/123" {
			return "/users/:id"
		}
		return path
	})

	httpAdapter := adapter.ForHTTP()

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := httpAdapter(testHandler)

	req := httptest.NewRequest("GET", "/users/123", nil)
	w := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w, req)

	// Verify route pattern was extracted
	require.NotNil(t, capturedMetrics)
	assert.Equal(t, "/users/:id", capturedMetrics.Path)
}

// Test adapter error handling
func TestMiddlewareAdapter_ErrorHandling(t *testing.T) {
	config := MiddlewareConfig{
		Enabled:    true,
		SampleRate: 1.0,
	}

	collector := NewMetricsCollector(config)
	adapter := NewMiddlewareAdapter(collector)

	// Test that adapter handles panics gracefully
	httpAdapter := adapter.ForHTTP()

	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	wrappedHandler := httpAdapter(panicHandler)

	// This should not panic - the middleware should recover
	assert.NotPanics(t, func() {
		req := httptest.NewRequest("GET", "/panic", nil)
		w := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(w, req)
	})
}

// Test adapter performance with many requests
func TestMiddlewareAdapter_Performance(t *testing.T) {
	config := MiddlewareConfig{
		Enabled:    true,
		SampleRate: 1.0,
	}

	collector := NewMetricsCollector(config)
	adapter := NewMiddlewareAdapter(collector)
	httpAdapter := adapter.ForHTTP()

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := httpAdapter(testHandler)

	// Benchmark many requests
	start := time.Now()
	numRequests := 1000

	for i := 0; i < numRequests; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(w, req)
	}

	duration := time.Since(start)
	averagePerRequest := duration / time.Duration(numRequests)

	// Should be very fast - less than 100μs per request on average
	assert.Less(t, averagePerRequest, 100*time.Microsecond,
		"Middleware should add minimal overhead")

	t.Logf("Average overhead per request: %v", averagePerRequest)
}
