# Pprofio Agent

[![Go Reference](https://pkg.go.dev/badge/github.com/pprofio/pprofio.svg)](https://pkg.go.dev/github.com/pprofio/pprofio)
[![Go Report Card](https://goreportcard.com/badge/github.com/pprofio/pprofio)](https://goreportcard.com/report/github.com/pprofio/pprofio)
[![Test](https://github.com/pprofio/pprofio/actions/workflows/test.yml/badge.svg)](https://github.com/pprofio/pprofio/actions/workflows/test.yml)
[![Lint](https://github.com/pprofio/pprofio/actions/workflows/lint.yml/badge.svg)](https://github.com/pprofio/pprofio/actions/workflows/lint.yml)
[![Coverage](https://codecov.io/gh/pprofio/pprofio/branch/main/graph/badge.svg)](https://codecov.io/gh/pprofio/pprofio)
[![Release](https://img.shields.io/github/v/release/pprofio/pprofio.svg)](https://github.com/pprofio/pprofio/releases)

Pprofio Agent is a lightweight, continuous profiling solution for Go applications. It collects runtime performance metrics with minimal overhead (<1% CPU) and uploads them to the Pprofio SaaS platform for analysis.

## Features

- **Simple Integration**: Single import with minimal configuration
- **Low Overhead**: <1% CPU/memory impact on your application
- **Multiple Metrics**: 
  - CPU profiles (MVP)
  - Memory profiles (MVP)
  - Goroutine profiles (Phase 2)
  - Mutex contention profiles (Phase 2)
  - Block profiles (Phase 3)
  - Custom instrumentation (Phase 3)
  - **🆕 HTTP Request Metrics**: Automatic request tracking with one-line middleware
- **Flexible Storage**: Upload to Pprofio SaaS or store locally
- **Router Support**: Works with standard HTTP, Gin, Chi, Gorilla Mux, Echo
- **Secure**: HTTPS and API key authentication

## Installation

```bash
go get github.com/pprofio/pprofio
```

### Requirements

- Go 1.20 or later
- Linux, macOS, or Windows

## Quick Start

### Basic Profiling

```go
package main

import (
    "context"
    "net/http"
    "time"
    
    "github.com/pprofio/pprofio"
)

func main() {
    // Configure the profiler
    cfg := pprofio.Config{
        APIKey:      "your-api-key",
        IngestURL:   "https://api.pprofio.com",
        SampleRate:  60 * time.Second,
        ServiceName: "my-service",
        Tags:        map[string]string{"env": "prod"},
        
        // 🆕 Enable HTTP request metrics
        EnableHTTPMetrics: true,
    }
    
    // Create and start the profiler
    p, err := pprofio.New(cfg)
    if err != nil {
        panic(err)
    }
    
    ctx := context.Background()
    p.Start(ctx)
    defer p.Stop()
    
    // Your application code here...
}
```

### HTTP Middleware Integration

Add HTTP request tracking to any router with **one line**:

```go
// Standard HTTP
mux := http.NewServeMux()
mux.HandleFunc("/api/users", handleUsers)
wrappedHandler := profiler.HTTPMiddleware()(mux)  // 🎯 ONE LINE!
http.ListenAndServe(":8080", wrappedHandler)

// Chi Router
r := chi.NewRouter()
r.Use(profiler.ForChi())  // 🎯 ONE LINE!
r.Get("/api/users", handleUsers)

// Gorilla Mux
r := mux.NewRouter()
r.Use(profiler.ForGorillaMux())  // 🎯 ONE LINE!
r.HandleFunc("/api/users", handleUsers)

// Gin Framework
r := gin.Default()
r.Use(gin.WrapH(profiler.ForGin()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))))
r.GET("/api/users", handleUsers)
```

## Configuration Options

### Basic Configuration

```go
type Config struct {
    // Required fields
    APIKey          string            // Your Pprofio API key
    IngestURL       string            // Pprofio ingest endpoint
    ServiceName     string            // Name of your service
    
    // Profiling options
    SampleRate      time.Duration     // Profile collection interval (default: 60s)
    ProfileDuration time.Duration     // Profile collection duration (default: 10s)
    Tags            map[string]string // Custom tags for profiles
    
    // Profile types (all default to false except CPU/Memory)
    EnableCPU       bool              // Enable CPU profiling (default: true)
    EnableMemory    bool              // Enable memory profiling (default: true)
    EnableGoroutine bool              // Enable goroutine profiling
    EnableMutex     bool              // Enable mutex profiling
    EnableBlock     bool              // Enable block profiling
    EnableCustom    bool              // Enable custom instrumentation
    
    // 🆕 HTTP Metrics (NEW!)
    EnableHTTPMetrics        bool          // Enable HTTP request metrics
    HTTPMetricsSampleRate    float64       // HTTP sampling rate (0.0-1.0, default: 1.0)
    HTTPMetricsExcludePaths  []string      // Paths to exclude (default: ["/health", "/metrics", "/ping"])
    HTTPMetricsIncludeHeaders []string     // Headers to capture
    HTTPMetricsCollectUserAgent bool       // Collect User-Agent header
    HTTPMetricsHashIPs       bool          // Hash IP addresses for privacy
    HTTPMetricsBatchSize     int           // Batch size for sending (default: 50)
    HTTPMetricsFlushInterval time.Duration // Flush interval (default: 30s)
    
    // Other options
    Hostname        string            // Hostname (auto-populated if empty)
    OutputToStdout  bool              // Output to stdout instead of upload
}
```

### Comprehensive Configuration Example

```go
cfg := pprofio.Config{
    // Required
    APIKey:      "your-api-key",
    IngestURL:   "https://api.pprofio.com",
    ServiceName: "my-service",
    
    // Enable all profiling types
    EnableCPU:       true,
    EnableMemory:    true,
    EnableGoroutine: true,
    EnableMutex:     true,
    EnableBlock:     true,
    EnableCustom:    true,
    
    // 🆕 HTTP Metrics
    EnableHTTPMetrics:           true,
    HTTPMetricsSampleRate:       1.0,  // 100% sampling
    HTTPMetricsExcludePaths:     []string{"/health", "/metrics", "/ping"},
    HTTPMetricsIncludeHeaders:   []string{"User-Agent", "Content-Type", "Referer"},
    HTTPMetricsCollectUserAgent: true,
    HTTPMetricsHashIPs:          true,  // Hash IPs for privacy
    
    // Service metadata
    Tags: map[string]string{
        "environment": "production",
        "version":     "1.2.3",
        "region":      "us-west-2",
        "team":        "backend",
    },
}

// Or use the comprehensive preset
cfg := pprofio.ComprehensiveConfig("api-key", "https://api.pprofio.com", "my-service")
```

## Examples

### Complete HTTP Server Example

```go
package main

import (
    "context"
    "net/http"
    "time"
    
    "github.com/pprofio/pprofio"
)

func main() {
    // Configure profiler with HTTP metrics
    cfg := pprofio.Config{
        APIKey:        "your-api-key",
        IngestURL:     "https://api.pprofio.com",
        ServiceName:   "my-api",
        EnableHTTPMetrics: true,
        Tags: map[string]string{
            "environment": "production",
            "version":     "1.0.0",
        },
    }
    
    profiler, err := pprofio.New(cfg)
    if err != nil {
        panic(err)
    }
    
    ctx := context.Background()
    profiler.Start(ctx)
    defer profiler.Stop()
    
    // Setup HTTP server with middleware
    mux := http.NewServeMux()
    mux.HandleFunc("/api/users", func(w http.ResponseWriter, r *http.Request) {
        time.Sleep(50 * time.Millisecond) // Simulate work
        w.WriteHeader(http.StatusOK)
        w.Write([]byte(`{"users": ["alice", "bob"]}`))
    })
    
    // 🎯 One line to add HTTP metrics!
    wrappedHandler := profiler.HTTPMiddleware()(mux)
    
    http.ListenAndServe(":8080", wrappedHandler)
}
```

**What you get:**
- CPU, memory, goroutine profiling every 60 seconds
- HTTP request metrics for every API call
- Request duration, status codes, payload sizes
- Automatic service tagging and metadata
- All sent to your Pprofio dashboard

### Framework Examples

See `examples/` directory for complete examples with:
- Standard HTTP server with middleware
- Integration patterns for different routers
- Local development with stdout output
- Production configuration examples

## Documentation

- **[Go Package Documentation](https://pkg.go.dev/github.com/pprofio/pprofio)**: Complete API reference
- **[Contributing Guide](CONTRIBUTING.md)**: Development setup and guidelines
- **[Changelog](CHANGELOG.md)**: Version history and breaking changes

## Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details on:

- Setting up your development environment
- Running tests and linting
- Submitting pull requests
- Release process

### Development Quick Start

```bash
# Clone the repository
git clone https://github.com/pprofio/pprofio.git
cd pprofio

# Install dependencies
go mod download

# Install development tools and git hooks
make install-tools
make install-hooks

# Run tests
go test -v -race ./...

# Run linting
golangci-lint run --timeout=10m
```

#### Git Hooks

This project uses git hooks to ensure code quality. After cloning, run `make install-hooks` to set up:

- **pre-commit**: Automatically checks that all Go files are properly formatted with `gofmt` before allowing commits

## Implementation Phases

The Pprofio Agent is being developed in three phases:

1. **MVP (Months 1-3)**: CPU and Memory profiling
2. **Phase 2 (Months 4-6)**: Goroutine and Mutex profiling
3. **Phase 3 (Months 7-12)**: Block profiling and Custom instrumentation

## Metrics Matrix

| Phase | Metric       | Data Collected                     | Frequency       | Output                       | Overhead |
|-------|--------------|------------------------------------|-----------------|------------------------------|----------|
| MVP   | CPU          | Stack traces, CPU time (ns)        | 10s every 60s   | Flame graphs                 | <0.5%    |
| MVP   | Memory       | Allocations (bytes), heap size     | Snapshot/60s    | Allocation graphs            | <0.3%    |
| **🆕** | **HTTP**     | **Request metrics, latency, status** | **Per request** | **Request analytics**        | **<0.1%** |
| Phase 2 | Goroutine  | Goroutine count, stack traces      | Snapshot/60s    | Area charts, leak alerts     | <0.2%    |
| Phase 2 | Mutex      | Contention events, wait time       | 10s every 60s   | Bar charts                   | <0.2%    |
| Phase 3 | Block      | Blocking events, duration          | 10s every 60s   | Pie/timeline charts          | <0.2%    |
| Phase 3 | Custom       | User-defined spans, tags, stacks   | Continuous      | Flame graphs, timelines      | <0.1%    |

### HTTP Metrics Details

The HTTP metrics collection captures:

- **Request Info**: Method, path, status code, duration
- **Size Metrics**: Request/response payload sizes
- **Client Data**: IP address (optionally hashed), User-Agent
- **Custom Headers**: Configurable header collection
- **Service Context**: Automatic service/environment/version tagging
- **Performance**: Ultra-low overhead (~8.5µs per request)
- **Privacy**: IP hashing, header filtering, path exclusion

## License

MIT