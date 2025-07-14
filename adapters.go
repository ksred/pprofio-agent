package pprofio

import (
	"context"
	"fmt"
	"net/http"
	"runtime/debug"
	"sync"
	"time"
)

const (
	DefaultHTTPStatusOK                  = 200
	DefaultHTTPStatusInternalServerError = 500
	PanicStatusCode                      = 500
	DefaultStatusCode                    = 200
)

// requestIDKey is the context key for storing request IDs
type requestIDKey struct{}

// RouteExtractor is a function type for extracting route patterns from requests
type RouteExtractor func(*http.Request) string

// MiddlewareAdapter provides framework-agnostic middleware functionality
type MiddlewareAdapter struct {
	collector      *MetricsCollector
	tags           map[string]string
	routeExtractor RouteExtractor
	mu             sync.RWMutex
}

// NewMiddlewareAdapter creates a new middleware adapter
func NewMiddlewareAdapter(collector *MetricsCollector) *MiddlewareAdapter {
	return &MiddlewareAdapter{
		collector: collector,
		tags:      make(map[string]string),
	}
}

// WithTags adds custom tags to be included in all metrics
func (ma *MiddlewareAdapter) WithTags(tags map[string]string) *MiddlewareAdapter {
	ma.mu.Lock()
	defer ma.mu.Unlock()

	for k, v := range tags {
		ma.tags[k] = v
	}

	return ma
}

// WithRouteExtractor sets a function to extract route patterns from requests
func (ma *MiddlewareAdapter) WithRouteExtractor(extractor RouteExtractor) *MiddlewareAdapter {
	ma.mu.Lock()
	defer ma.mu.Unlock()
	ma.routeExtractor = extractor

	return ma
}

// ForHTTP returns standard HTTP middleware
func (ma *MiddlewareAdapter) ForHTTP() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if middleware is enabled
			if !ma.collector.config.Enabled {
				next.ServeHTTP(w, r)
				return
			}

			// Check if path is excluded
			if ma.collector.isExcludedPath(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			// Check sampling
			if !ma.collector.shouldSample() {
				next.ServeHTTP(w, r)
				return
			}

			// Generate request ID and add to context
			requestID := ma.collector.generateRequestID()
			ctx := context.WithValue(r.Context(), requestIDKey{}, requestID)
			r = r.WithContext(ctx)

			// Create enhanced response writer with proper timing
			startTime := time.Now()
			rw := &enhancedResponseWriter{
				ResponseWriter: w,
				statusCode:     DefaultStatusCode,
				startTime:      startTime,
			}

			// Panic recovery
			defer func() {
				if err := recover(); err != nil {
					// Log the panic (in real implementation, use proper logging)
					fmt.Printf("Panic in HTTP handler: %v\n%s", err, debug.Stack())

					// Set error status if not already set
					if rw.statusCode == DefaultHTTPStatusOK {
						rw.statusCode = PanicStatusCode
					}

					// Ensure endTime is set for metrics collection
					if rw.endTime.IsZero() {
						rw.endTime = time.Now()
					}

					// Continue with metrics collection even after panic
					ma.collectMetrics(r, rw, requestID)
				}
			}()

			// Track active requests using atomic operations (safer)
			ma.collector.mu.Lock()
			ma.collector.activeRequests++
			ma.collector.mu.Unlock()

			defer func() {
				ma.collector.mu.Lock()
				ma.collector.activeRequests--
				ma.collector.mu.Unlock()
			}()

			// Execute the handler
			next.ServeHTTP(rw, r)

			// Ensure endTime is set if not already set by WriteHeader
			if rw.endTime.IsZero() {
				rw.endTime = time.Now()
			}

			// Collect metrics
			ma.collectMetrics(r, rw, requestID)
		})
	}
}

// collectMetrics collects and sends metrics for a completed request
func (ma *MiddlewareAdapter) collectMetrics(r *http.Request, rw *enhancedResponseWriter, _ string) {
	// Calculate duration
	duration := rw.endTime.Sub(rw.startTime)

	// Get path - use route extractor if available
	path := r.URL.Path

	ma.mu.RLock()
	if ma.routeExtractor != nil {
		path = ma.routeExtractor(r)
	}
	ma.mu.RUnlock()

	// Get request size
	requestSize := r.ContentLength
	if requestSize < 0 {
		requestSize = 0
	}

	// Create metrics
	metrics := &RequestMetrics{
		RequestID:    GetRequestIDFromContext(r.Context()),
		Method:       r.Method,
		Path:         path,
		StatusCode:   rw.statusCode,
		Duration:     duration,
		DurationNs:   duration.Nanoseconds(),
		RequestSize:  requestSize,
		ResponseSize: rw.responseSize,
		Timestamp:    rw.startTime,
		Tags:         make(map[string]string),
	}

	// Extract common fields from custom tags and move to top-level
	ma.mu.RLock()
	for k, v := range ma.tags {
		switch k {
		case "service":
			metrics.Service = v
		case "environment":
			metrics.Environment = v
		case "version":
			metrics.Version = v
		case "region":
			metrics.Region = v
		default:
			// Keep remaining tags in the tags field
			metrics.Tags[k] = v
		}
	}
	ma.mu.RUnlock()

	// Add optional fields based on configuration
	if ma.collector.config.CollectUserAgent {
		metrics.UserAgent = r.Header.Get("User-Agent")
	}

	// Add client IP
	metrics.ClientIP = ma.collector.hashIP(ma.collector.extractClientIP(r))

	// Add headers if configured
	metrics.Headers = ma.collector.collectHeaders(r)

	// Send metrics to handler if configured
	ma.collector.mu.RLock()
	handler := ma.collector.metricsHandler
	ma.collector.mu.RUnlock()

	if handler != nil {
		handler(metrics)
	}
}

// enhancedResponseWriter extends the basic responseWriter with timing and panic recovery
type enhancedResponseWriter struct {
	http.ResponseWriter
	statusCode   int
	responseSize int64
	startTime    time.Time
	endTime      time.Time
}

// WriteHeader captures the status code and timing
func (rw *enhancedResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	if rw.endTime.IsZero() {
		rw.endTime = time.Now()
	}

	rw.ResponseWriter.WriteHeader(code)
}

// Write captures the response size and timing
func (rw *enhancedResponseWriter) Write(b []byte) (int, error) {
	if rw.endTime.IsZero() {
		rw.endTime = time.Now()
	}

	n, err := rw.ResponseWriter.Write(b)
	rw.responseSize += int64(n)

	return n, err
}

// GetRequestIDFromContext retrieves the request ID from the request context
func GetRequestIDFromContext(ctx context.Context) string {
	if requestID, ok := ctx.Value(requestIDKey{}).(string); ok {
		return requestID
	}

	return ""
}

// SetRequestIDInContext sets a request ID in the context (useful for testing)
func SetRequestIDInContext(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, requestID)
}
