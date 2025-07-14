package pprofio

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	TestSampleRate     = 1.0
	TestServiceName    = "test-service"
	TestVersion        = "1.0.0"
	TestEnvironment    = "test"
	TestRegion         = "us-west-1"
	TestTeam           = "backend"
	TestComponent      = "auth"
	TestUserPath       = "/users/123"
	TestUserPattern    = "/users/:id"
	TestResponse       = "test response"
	TestPanicPath      = "/panic"
	TestPanicMessage   = "test panic"
	TestRequestCount   = 1000
	MaxAverageOverhead = 100 * time.Microsecond
)

// Test framework-agnostic middleware interface
func TestMiddlewareAdapter_Interface(t *testing.T) {
	config := MiddlewareConfig{
		Enabled:          true,
		SampleRate:       TestSampleRate,
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
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(TestResponse))
	})

	wrappedHandler := httpAdapter(testHandler)
	require.NotNil(t, wrappedHandler)

	// Verify the wrapped handler works
	req := httptest.NewRequest("GET", "/test", http.NoBody)
	w := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, TestResponse, w.Body.String())
}

// Test middleware context and request ID propagation
func TestMiddlewareAdapter_ContextPropagation(t *testing.T) {
	config := MiddlewareConfig{
		Enabled:    true,
		SampleRate: TestSampleRate,
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

	req := httptest.NewRequest("GET", "/test", http.NoBody)
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
		SampleRate: TestSampleRate,
	}

	collector := NewMetricsCollector(config)
	var capturedMetrics *RequestMetrics

	collector.SetMetricsHandler(func(metrics *RequestMetrics) {
		capturedMetrics = metrics
	})

	adapter := NewMiddlewareAdapter(collector)

	// Add custom tags including common fields that should be moved to top-level
	adapter.WithTags(map[string]string{
		"service":     TestServiceName, // Should move to top-level
		"version":     TestVersion,     // Should move to top-level
		"environment": TestEnvironment, // Should move to top-level
		"region":      TestRegion,      // Should move to top-level
		"team":        TestTeam,        // Should remain in tags
		"component":   TestComponent,   // Should remain in tags
	})

	httpAdapter := adapter.ForHTTP()

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := httpAdapter(testHandler)

	req := httptest.NewRequest("GET", "/test", http.NoBody)
	w := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w, req)

	// Verify common fields were moved to top-level
	require.NotNil(t, capturedMetrics)
	assert.Equal(t, TestServiceName, capturedMetrics.Service)
	assert.Equal(t, TestVersion, capturedMetrics.Version)
	assert.Equal(t, TestEnvironment, capturedMetrics.Environment)
	assert.Equal(t, TestRegion, capturedMetrics.Region)

	// Verify remaining tags stayed in tags field
	assert.Equal(t, TestTeam, capturedMetrics.Tags["team"])
	assert.Equal(t, TestComponent, capturedMetrics.Tags["component"])

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
		SampleRate: TestSampleRate,
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
		if path == TestUserPath {
			return TestUserPattern
		}
		return path
	})

	httpAdapter := adapter.ForHTTP()

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := httpAdapter(testHandler)

	req := httptest.NewRequest("GET", TestUserPath, http.NoBody)
	w := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w, req)

	// Verify route pattern was extracted
	require.NotNil(t, capturedMetrics)
	assert.Equal(t, TestUserPattern, capturedMetrics.Path)
}

// Test adapter error handling
func TestMiddlewareAdapter_ErrorHandling(t *testing.T) {
	config := MiddlewareConfig{
		Enabled:    true,
		SampleRate: TestSampleRate,
	}

	collector := NewMetricsCollector(config)
	adapter := NewMiddlewareAdapter(collector)

	// Test that adapter handles panics gracefully
	httpAdapter := adapter.ForHTTP()

	panicHandler := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		panic(TestPanicMessage)
	})

	wrappedHandler := httpAdapter(panicHandler)

	// This should not panic - the middleware should recover
	assert.NotPanics(t, func() {
		req := httptest.NewRequest("GET", TestPanicPath, http.NoBody)
		w := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(w, req)
	})
}

// Test adapter performance with many requests
func TestMiddlewareAdapter_Performance(t *testing.T) {
	config := MiddlewareConfig{
		Enabled:    true,
		SampleRate: TestSampleRate,
	}

	collector := NewMetricsCollector(config)
	adapter := NewMiddlewareAdapter(collector)
	httpAdapter := adapter.ForHTTP()

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := httpAdapter(testHandler)

	// Benchmark many requests
	start := time.Now()
	numRequests := TestRequestCount

	for i := 0; i < numRequests; i++ {
		req := httptest.NewRequest("GET", "/test", http.NoBody)
		w := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(w, req)
	}

	duration := time.Since(start)
	averagePerRequest := duration / time.Duration(numRequests)

	// Should be very fast - less than 100μs per request on average
	assert.Less(t, averagePerRequest, MaxAverageOverhead,
		"Middleware should add minimal overhead")

	t.Logf("Average overhead per request: %v", averagePerRequest)
}
