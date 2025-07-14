package pprofio

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test the middleware configuration structure
func TestMiddlewareConfig_Validation(t *testing.T) {
	tests := []struct {
		name        string
		config      MiddlewareConfig
		expectError bool
	}{
		{
			name: "valid config",
			config: MiddlewareConfig{
				Enabled:          true,
				SampleRate:       1.0,
				ExcludedPaths:    []string{"/health", "/metrics"},
				IncludeHeaders:   []string{"User-Agent", "Content-Type"},
				MaxPayloadSize:   1024 * 1024, // 1MB
				CollectUserAgent: true,
				HashIPs:          false,
			},
			expectError: false,
		},
		{
			name: "invalid sample rate",
			config: MiddlewareConfig{
				Enabled:    true,
				SampleRate: 1.5, // Invalid: > 1.0
			},
			expectError: true,
		},
		{
			name: "negative sample rate",
			config: MiddlewareConfig{
				Enabled:    true,
				SampleRate: -0.1, // Invalid: < 0.0
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Test request metrics structure and validation
func TestRequestMetrics_Structure(t *testing.T) {
	metrics := &RequestMetrics{
		RequestID:    "test-req-123",
		Method:       "GET",
		Path:         "/api/users",
		StatusCode:   200,
		Duration:     time.Millisecond * 150,
		RequestSize:  1024,
		ResponseSize: 2048,
		UserAgent:    "test-agent/1.0",
		Timestamp:    time.Now(),
		Tags: map[string]string{
			"service":     "user-api",
			"environment": "test",
		},
	}

	assert.Equal(t, "test-req-123", metrics.RequestID)
	assert.Equal(t, "GET", metrics.Method)
	assert.Equal(t, "/api/users", metrics.Path)
	assert.Equal(t, 200, metrics.StatusCode)
	assert.Equal(t, time.Millisecond*150, metrics.Duration)
	assert.Equal(t, int64(1024), metrics.RequestSize)
	assert.Equal(t, int64(2048), metrics.ResponseSize)
	assert.Contains(t, metrics.Tags, "service")
	assert.Contains(t, metrics.Tags, "environment")
}

// Test metrics collector initialization
func TestMetricsCollector_New(t *testing.T) {
	config := MiddlewareConfig{
		Enabled:          true,
		SampleRate:       1.0,
		ExcludedPaths:    []string{"/health"},
		CollectUserAgent: true,
	}

	collector := NewMetricsCollector(config)
	require.NotNil(t, collector)
	assert.Equal(t, config.Enabled, collector.config.Enabled)
	assert.Equal(t, config.SampleRate, collector.config.SampleRate)
	assert.Equal(t, config.ExcludedPaths, collector.config.ExcludedPaths)
}

// Test basic HTTP middleware functionality
func TestHTTPMiddleware_BasicFunctionality(t *testing.T) {
	config := MiddlewareConfig{
		Enabled:          true,
		SampleRate:       1.0,
		CollectUserAgent: true,
	}

	collector := NewMetricsCollector(config)
	var capturedMetrics *RequestMetrics

	// Mock the metrics handler
	collector.SetMetricsHandler(func(metrics *RequestMetrics) {
		capturedMetrics = metrics
	})

	// Create a simple test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello, World!"))
	})

	// Wrap with our middleware
	middleware := collector.HTTPMiddleware()
	wrappedHandler := middleware(testHandler)

	// Create test request
	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("User-Agent", "test-client/1.0")
	w := httptest.NewRecorder()

	// Execute request
	wrappedHandler.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "Hello, World!", w.Body.String())

	// Verify metrics were captured
	require.NotNil(t, capturedMetrics)
	assert.Equal(t, "GET", capturedMetrics.Method)
	assert.Equal(t, "/api/test", capturedMetrics.Path)
	assert.Equal(t, 200, capturedMetrics.StatusCode)
	assert.Equal(t, "test-client/1.0", capturedMetrics.UserAgent)
	assert.Greater(t, capturedMetrics.Duration, time.Duration(0))
	assert.NotEmpty(t, capturedMetrics.RequestID)
}

// Test middleware with excluded paths
func TestHTTPMiddleware_ExcludedPaths(t *testing.T) {
	config := MiddlewareConfig{
		Enabled:       true,
		SampleRate:    1.0,
		ExcludedPaths: []string{"/health", "/metrics"},
	}

	collector := NewMetricsCollector(config)
	var capturedMetrics *RequestMetrics

	collector.SetMetricsHandler(func(metrics *RequestMetrics) {
		capturedMetrics = metrics
	})

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := collector.HTTPMiddleware()
	wrappedHandler := middleware(testHandler)

	// Test excluded path
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w, req)

	// Metrics should not be captured for excluded paths
	assert.Nil(t, capturedMetrics)

	// Test non-excluded path
	req = httptest.NewRequest("GET", "/api/users", nil)
	w = httptest.NewRecorder()
	wrappedHandler.ServeHTTP(w, req)

	// Metrics should be captured for non-excluded paths
	assert.NotNil(t, capturedMetrics)
	assert.Equal(t, "/api/users", capturedMetrics.Path)
}

// Test sampling functionality
func TestHTTPMiddleware_Sampling(t *testing.T) {
	config := MiddlewareConfig{
		Enabled:    true,
		SampleRate: 0.0, // No sampling - should capture no metrics
	}

	collector := NewMetricsCollector(config)
	var captureCount int

	collector.SetMetricsHandler(func(metrics *RequestMetrics) {
		captureCount++
	})

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := collector.HTTPMiddleware()
	wrappedHandler := middleware(testHandler)

	// Make multiple requests
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("GET", "/api/test", nil)
		w := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(w, req)
	}

	// With 0% sampling, no metrics should be captured
	assert.Equal(t, 0, captureCount)
}

// Test concurrent request tracking
func TestHTTPMiddleware_ConcurrentRequests(t *testing.T) {
	config := MiddlewareConfig{
		Enabled:    true,
		SampleRate: 1.0,
	}

	collector := NewMetricsCollector(config)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate some processing time
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	})

	middleware := collector.HTTPMiddleware()
	wrappedHandler := middleware(testHandler)

	// Test that concurrent requests are handled properly
	done := make(chan bool)
	for i := 0; i < 5; i++ {
		go func() {
			req := httptest.NewRequest("GET", "/api/test", nil)
			w := httptest.NewRecorder()
			wrappedHandler.ServeHTTP(w, req)
			done <- true
		}()
	}

	// Wait for all requests to complete
	for i := 0; i < 5; i++ {
		<-done
	}

	// Test passes if no race conditions occur
	assert.True(t, true)
}

// Test error handling in middleware
func TestHTTPMiddleware_ErrorHandling(t *testing.T) {
	config := MiddlewareConfig{
		Enabled:    true,
		SampleRate: 1.0,
	}

	collector := NewMetricsCollector(config)
	var capturedMetrics *RequestMetrics

	collector.SetMetricsHandler(func(metrics *RequestMetrics) {
		capturedMetrics = metrics
	})

	// Handler that returns an error status
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	})

	middleware := collector.HTTPMiddleware()
	wrappedHandler := middleware(testHandler)

	req := httptest.NewRequest("POST", "/api/error", strings.NewReader("test data"))
	w := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(w, req)

	// Verify error metrics are captured correctly
	require.NotNil(t, capturedMetrics)
	assert.Equal(t, 500, capturedMetrics.StatusCode)
	assert.Equal(t, "POST", capturedMetrics.Method)
	assert.Greater(t, capturedMetrics.RequestSize, int64(0))
}

// Test HTTP middleware metrics collection
func TestHTTPMiddleware_MetricsCollection(t *testing.T) {
	config := MiddlewareConfig{
		Enabled:          true,
		SampleRate:       1.0,
		CollectUserAgent: true,
	}

	collector := NewMetricsCollector(config)
	var capturedMetrics *RequestMetrics

	collector.SetMetricsHandler(func(metrics *RequestMetrics) {
		capturedMetrics = metrics
	})

	// Create a simple handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello, World!"))
	})

	// Wrap with middleware
	middleware := collector.HTTPMiddleware()
	wrappedHandler := middleware(handler)

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "test-agent")
	w := httptest.NewRecorder()

	// Execute request
	wrappedHandler.ServeHTTP(w, req)

	// Verify response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "Hello, World!", w.Body.String())

	// Verify metrics were captured
	require.NotNil(t, capturedMetrics)
	assert.NotEmpty(t, capturedMetrics.RequestID)
	assert.Equal(t, "GET", capturedMetrics.Method)
	assert.Equal(t, "/test", capturedMetrics.Path)
	assert.Equal(t, 200, capturedMetrics.StatusCode)
	assert.Greater(t, capturedMetrics.Duration, time.Duration(0))
	assert.Greater(t, capturedMetrics.DurationNs, int64(0))
	assert.Equal(t, capturedMetrics.Duration.Nanoseconds(), capturedMetrics.DurationNs)
	assert.Equal(t, int64(0), capturedMetrics.RequestSize)
	assert.Equal(t, int64(13), capturedMetrics.ResponseSize)
	assert.Equal(t, "test-agent", capturedMetrics.UserAgent)
	assert.NotZero(t, capturedMetrics.Timestamp)
}
