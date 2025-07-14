package pprofio

import (
	"crypto/md5"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// MiddlewareConfig defines the configuration for HTTP middleware
type MiddlewareConfig struct {
	Enabled          bool     `json:"enabled"`
	SampleRate       float64  `json:"sample_rate"`      // 0.0 to 1.0
	ExcludedPaths    []string `json:"excluded_paths"`   // Paths to exclude from monitoring
	IncludeHeaders   []string `json:"include_headers"`  // Headers to capture
	MaxPayloadSize   int64    `json:"max_payload_size"` // Maximum payload size to capture
	CollectUserAgent bool     `json:"collect_user_agent"`
	HashIPs          bool     `json:"hash_ips"` // Whether to hash IP addresses
}

// Validate validates the middleware configuration
func (c *MiddlewareConfig) Validate() error {
	if c.SampleRate < 0.0 || c.SampleRate > 1.0 {
		return fmt.Errorf("sample rate must be between 0.0 and 1.0, got %f", c.SampleRate)
	}
	if c.MaxPayloadSize < 0 {
		return fmt.Errorf("max payload size must be non-negative, got %d", c.MaxPayloadSize)
	}
	return nil
}

// RequestMetrics represents HTTP request metrics collected by the middleware
type RequestMetrics struct {
	RequestID    string            `json:"request_id"`
	Method       string            `json:"method"`
	Path         string            `json:"path"`
	StatusCode   int               `json:"status_code"`
	Duration     time.Duration     `json:"duration"`
	DurationNs   int64             `json:"duration_ns"` // NEW: for aggregation
	RequestSize  int64             `json:"request_size"`
	ResponseSize int64             `json:"response_size"`
	UserAgent    string            `json:"user_agent,omitempty"`
	ClientIP     string            `json:"client_ip,omitempty"`
	Headers      map[string]string `json:"headers,omitempty"`
	Timestamp    time.Time         `json:"timestamp"`

	// Common fields moved from tags to top-level
	Service     string `json:"service,omitempty"`     // MOVED: from tags
	Environment string `json:"environment,omitempty"` // MOVED: from tags
	Version     string `json:"version,omitempty"`     // MOVED: from tags
	Region      string `json:"region,omitempty"`      // MOVED: from tags

	Tags map[string]string `json:"tags,omitempty"` // REMAINING: for additional custom tags
}

// MetricsHandler is a function type for handling collected metrics
type MetricsHandler func(*RequestMetrics)

// MetricsCollector is responsible for collecting HTTP metrics
type MetricsCollector struct {
	config         MiddlewareConfig
	metricsHandler MetricsHandler
	activeRequests int64
	mu             sync.RWMutex
	rng            *rand.Rand
}

// NewMetricsCollector creates a new metrics collector with the given configuration
func NewMetricsCollector(config MiddlewareConfig) *MetricsCollector {
	return &MetricsCollector{
		config: config,
		rng:    rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// SetMetricsHandler sets the handler function for collected metrics
func (mc *MetricsCollector) SetMetricsHandler(handler MetricsHandler) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.metricsHandler = handler
}

// shouldSample determines if this request should be sampled based on the sample rate
func (mc *MetricsCollector) shouldSample() bool {
	if mc.config.SampleRate >= 1.0 {
		return true
	}
	if mc.config.SampleRate <= 0.0 {
		return false
	}
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return mc.rng.Float64() < mc.config.SampleRate
}

// isExcludedPath checks if the given path should be excluded from monitoring
func (mc *MetricsCollector) isExcludedPath(path string) bool {
	for _, excludedPath := range mc.config.ExcludedPaths {
		if path == excludedPath || strings.HasPrefix(path, excludedPath) {
			return true
		}
	}
	return false
}

// generateRequestID generates a unique request ID
func (mc *MetricsCollector) generateRequestID() string {
	return fmt.Sprintf("req_%d_%d", time.Now().UnixNano(), rand.Int63())
}

// hashIP creates a hash of the IP address for privacy
func (mc *MetricsCollector) hashIP(ip string) string {
	if !mc.config.HashIPs {
		return ip
	}
	hash := md5.Sum([]byte(ip))
	return fmt.Sprintf("%x", hash)
}

// extractClientIP extracts the client IP from the request
func (mc *MetricsCollector) extractClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}

	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}

// collectHeaders collects specified headers from the request
func (mc *MetricsCollector) collectHeaders(r *http.Request) map[string]string {
	if len(mc.config.IncludeHeaders) == 0 {
		return nil
	}

	headers := make(map[string]string)
	for _, headerName := range mc.config.IncludeHeaders {
		if value := r.Header.Get(headerName); value != "" {
			headers[headerName] = value
		}
	}

	if len(headers) == 0 {
		return nil
	}
	return headers
}

// responseWriter is a wrapper around http.ResponseWriter to capture metrics
type responseWriter struct {
	http.ResponseWriter
	statusCode   int
	responseSize int64
}

// newResponseWriter creates a new response writer wrapper
func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{
		ResponseWriter: w,
		statusCode:     200, // Default status code
	}
}

// WriteHeader captures the status code
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Write captures the response size
func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.responseSize += int64(n)
	return n, err
}

// HTTPMiddleware returns HTTP middleware that collects request metrics
func (mc *MetricsCollector) HTTPMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if middleware is enabled
			if !mc.config.Enabled {
				next.ServeHTTP(w, r)
				return
			}

			// Check if path is excluded
			if mc.isExcludedPath(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			// Check sampling
			if !mc.shouldSample() {
				next.ServeHTTP(w, r)
				return
			}

			// Track active requests
			atomic.AddInt64(&mc.activeRequests, 1)
			defer atomic.AddInt64(&mc.activeRequests, -1)

			// Start timing
			startTime := time.Now()

			// Generate request ID
			requestID := mc.generateRequestID()

			// Wrap response writer to capture metrics
			rw := newResponseWriter(w)

			// Get request size
			requestSize := r.ContentLength
			if requestSize < 0 {
				requestSize = 0
			}

			// Execute the handler
			next.ServeHTTP(rw, r)

			// Calculate duration
			duration := time.Since(startTime)

			// Collect metrics
			metrics := &RequestMetrics{
				RequestID:    requestID,
				Method:       r.Method,
				Path:         r.URL.Path,
				StatusCode:   rw.statusCode,
				Duration:     duration,
				DurationNs:   duration.Nanoseconds(),
				RequestSize:  requestSize,
				ResponseSize: rw.responseSize,
				Timestamp:    startTime,
			}

			// Add optional fields based on configuration
			if mc.config.CollectUserAgent {
				metrics.UserAgent = r.Header.Get("User-Agent")
			}

			// Add client IP
			metrics.ClientIP = mc.hashIP(mc.extractClientIP(r))

			// Add headers if configured
			metrics.Headers = mc.collectHeaders(r)

			// Send metrics to handler if configured
			mc.mu.RLock()
			handler := mc.metricsHandler
			mc.mu.RUnlock()

			if handler != nil {
				handler(metrics)
			}
		})
	}
}

// GetActiveRequestCount returns the current number of active requests
func (mc *MetricsCollector) GetActiveRequestCount() int64 {
	return atomic.LoadInt64(&mc.activeRequests)
}
