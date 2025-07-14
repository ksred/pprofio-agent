package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/pprofio/pprofio"
)

// This example demonstrates how to use the pprofio HTTP middleware
// Note: You'll need to adjust the import above and the type references below
// based on your actual module structure

func main() {
	// Configure the middleware
	middlewareConfig := pprofio.MiddlewareConfig{
		Enabled:          true,
		SampleRate:       1.0, // Monitor 100% of requests
		ExcludedPaths:    []string{"/health", "/metrics"},
		IncludeHeaders:   []string{"User-Agent", "Content-Type"},
		MaxPayloadSize:   1024 * 1024, // 1MB
		CollectUserAgent: true,
		HashIPs:          false, // Set to true for production
	}

	// Create metrics collector
	collector := pprofio.NewMetricsCollector(middlewareConfig)

	// Create adapter and add service information
	adapter := pprofio.NewMiddlewareAdapter(collector)

	// Add service information - these will be moved to top-level fields in JSON
	adapter.WithTags(map[string]string{
		"service":     "user-api",    // Moves to top-level 'service' field
		"environment": "development", // Moves to top-level 'environment' field
		"version":     "1.0.0",       // Moves to top-level 'version' field
		"region":      "us-west-2",   // Moves to top-level 'region' field
		"team":        "backend",     // Stays in 'tags' field
		"component":   "auth",        // Stays in 'tags' field
	})

	// Set up metrics handler to process collected metrics
	collector.SetMetricsHandler(func(metrics *pprofio.RequestMetrics) {
		fmt.Printf("Request: %s %s -> %d (%v / %dns)\n",
			metrics.Method,
			metrics.Path,
			metrics.StatusCode,
			metrics.Duration,
			metrics.DurationNs) // NEW: duration in nanoseconds for aggregation

		fmt.Printf("  Service: %s, Environment: %s, Version: %s\n",
			metrics.Service,     // NEW: top-level field
			metrics.Environment, // NEW: top-level field
			metrics.Version)     // NEW: top-level field

		// In production, you would send these metrics to your monitoring system
		// JSON structure now includes duration_ns and common fields at top-level
	})

	// Create your HTTP handlers
	mux := http.NewServeMux()

	mux.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		// Simulate some processing
		time.Sleep(10 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "Hello, World!"}`))
	})

	mux.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		// Simulate database query
		time.Sleep(50 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"id": 1, "name": "John"}, {"id": 2, "name": "Jane"}]`))
	})

	mux.HandleFunc("/error", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Something went wrong"))
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		// This endpoint is excluded from monitoring
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Create adapter and wrap your handler with the middleware
	wrappedHandler := adapter.ForHTTP()(mux)

	fmt.Println("Server starting on :8080")
	fmt.Println("Try these endpoints:")
	fmt.Println("  GET  http://localhost:8080/hello")
	fmt.Println("  GET  http://localhost:8080/users")
	fmt.Println("  GET  http://localhost:8080/error")
	fmt.Println("  GET  http://localhost:8080/health  (excluded from metrics)")

	log.Fatal(http.ListenAndServe(":8080", wrappedHandler))
}
