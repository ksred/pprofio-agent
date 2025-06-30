# pprofio API Documentation

This document provides comprehensive API documentation for the pprofio Go package.

## Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [Core Types](#core-types)
- [Storage Backends](#storage-backends)
- [Custom Spans](#custom-spans)
- [Metrics and Health](#metrics-and-health)
- [Error Handling](#error-handling)
- [Build Tags](#build-tags)
- [Examples](#examples)

## Installation

```bash
go get github.com/pprofio/pprofio
```

Minimum Go version: 1.18

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "log"
    "os"
    "os/signal"
    
    "github.com/pprofio/pprofio"
)

func main() {
    // Create configuration
    cfg := pprofio.DefaultConfig(
        os.Getenv("PPROFIO_API_KEY"),
        "https://api.pprofio.com",
        "my-service",
    )
    
    // Create profiler
    profiler, err := pprofio.New(cfg)
    if err != nil {
        log.Fatal("Failed to create profiler:", err)
    }
    
    // Start profiling
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    if err := profiler.Start(ctx); err != nil {
        log.Fatal("Failed to start profiler:", err)
    }
    defer profiler.Stop()
    
    // Wait for interrupt
    c := make(chan os.Signal, 1)
    signal.Notify(c, os.Interrupt)
    <-c
}
```

## Configuration

### Config Type

```go
type Config struct {
    // Required fields
    APIKey      string
    IngestURL   string
    ServiceName string
    
    // Optional timing configuration
    SampleRate      time.Duration // Default: 60s
    ProfileDuration time.Duration // Default: 10s
    
    // Storage backend (auto-created if nil)
    Storage Storage
    
    // Metadata
    Tags map[string]string
    Env  string
    
    // Profile type enablement
    EnableCPU       bool // Default: true
    EnableMemory    bool // Default: true
    EnableGoroutine bool // Default: false
    EnableMutex     bool // Default: false
    EnableBlock     bool // Default: false
    EnableCustom    bool // Default: false
    
    // Profiling detail levels
    MemProfileRate   int // Default: 4096
    MutexFraction    int // Default: 5
    BlockProfileRate int // Default: 100
    
    // Special modes
    OutputToStdout bool // Default: false
}
```

### Configuration Functions

#### `DefaultConfig(apiKey, ingestURL, serviceName string) Config`

Creates a production-ready configuration with sensible defaults.

```go
cfg := pprofio.DefaultConfig("your-api-key", "https://api.pprofio.com", "my-service")
cfg.Tags["env"] = "production"
cfg.Tags["version"] = "1.0.0"
```

#### `(c *Config) validate() error`

Validates configuration and sets defaults. Called automatically by `New()`.

## Core Types

### Profiler

The main profiler type that manages profile collection and upload.

#### `New(config Config) (*Profiler, error)`

Creates a new profiler instance with the given configuration.

**Parameters:**
- `config`: Configuration options

**Returns:**
- `*Profiler`: Configured profiler instance
- `error`: Configuration validation error

**Example:**
```go
cfg := pprofio.Config{
    APIKey:      "your-api-key",
    IngestURL:   "https://api.pprofio.com",
    ServiceName: "my-service",
    EnableCPU:   true,
    EnableMemory: true,
}

profiler, err := pprofio.New(cfg)
if err != nil {
    log.Fatal(err)
}
```

#### `(p *Profiler) Start(ctx context.Context) error`

Starts the profiler and begins collecting profiles.

**Parameters:**
- `ctx`: Context for cancellation

**Returns:**
- `error`: Start failure error

**Behavior:**
- Configures Go runtime profiling settings
- Starts background goroutines for enabled profile types
- Returns `ErrAlreadyStarted` if already running

#### `(p *Profiler) Stop()`

Stops the profiler and waits for pending operations.

**Behavior:**
- Stops all background goroutines
- Restores original runtime settings
- Waits for pending uploads to complete
- Safe to call multiple times

#### `(p *Profiler) GetMetrics() Metrics`

Returns current profiler metrics for monitoring.

**Returns:**
- `Metrics`: Snapshot of current metrics

#### `(p *Profiler) HealthCheck() HealthStatus`

Returns profiler health status based on metrics.

**Returns:**
- `HealthStatus`: "healthy", "degraded", or "unhealthy"

### Version Information

#### `GetBuildInfo() BuildInfo`

Returns comprehensive build and runtime information.

```go
info := pprofio.GetBuildInfo()
log.Printf("Using pprofio %s", info.Version)
```

#### `BuildInfo` Type

```go
type BuildInfo struct {
    Version   string `json:"version"`
    Commit    string `json:"commit"`
    BuildDate string `json:"build_date"`
    GoVersion string `json:"go_version"`
    OS        string `json:"os"`
    Arch      string `json:"arch"`
}
```

## Storage Backends

### Storage Interface

```go
type Storage interface {
    Upload(ctx context.Context, filePath string) (string, error)
}
```

### HTTPStorage

Uploads profiles to remote HTTP API with authentication and retry logic.

#### `NewHTTPStorage(url, apiKey, env string) *HTTPStorage`

Creates HTTP storage with default settings.

**Parameters:**
- `url`: Upload endpoint URL
- `apiKey`: Authentication token
- `env`: Environment identifier

**Example:**
```go
storage := pprofio.NewHTTPStorage(
    "https://api.pprofio.com/upload",
    "your-api-key",
    "production",
)
```

### FileStorage

Saves profiles to local filesystem.

#### `NewFileStorage(directory string) (*FileStorage, error)`

Creates file storage for specified directory.

**Parameters:**
- `directory`: Directory path for storing profiles

**Example:**
```go
storage, err := pprofio.NewFileStorage("/var/log/profiles")
if err != nil {
    log.Fatal(err)
}
```

### StdoutStorage

Outputs profile information to stdout for debugging.

#### `NewStdoutStorage() *StdoutStorage`

Creates stdout storage instance.

**Example:**
```go
cfg := pprofio.Config{
    ServiceName:    "my-service",
    OutputToStdout: true,
    Storage:        pprofio.NewStdoutStorage(),
}
```

## Custom Spans

### Span Type

```go
type Span struct {
    Name     string
    Start    time.Time
    Duration time.Duration
    Tags     map[string]string
}
```

### Span Functions

#### `StartSpan(ctx context.Context, name string, tags ...string) (context.Context, *Span)`

Creates and starts a new span for custom instrumentation.

**Parameters:**
- `ctx`: Context (should contain profiler via `WithProfiler`)
- `name`: Operation name
- `tags`: Key-value pairs as alternating strings

**Returns:**
- `context.Context`: Updated context
- `*Span`: Created span

**Example:**
```go
ctx = pprofio.WithProfiler(ctx, profiler)
ctx, span := pprofio.StartSpan(ctx, "database_query", 
    "table", "users",
    "operation", "select",
)
defer span.End()

// ... perform database operation ...
```

#### `WithProfiler(ctx context.Context, p *Profiler) context.Context`

Attaches profiler to context for span collection.

#### `(s *Span) End()`

Marks span completion and records duration.

## Metrics and Health

### Metrics Type

```go
type Metrics struct {
    // Profile collection
    ProfilesCollected   int64
    ProfilesUploaded    int64
    ProfilesFailed      int64
    UploadRetries       int64
    
    // Timing
    LastCollectionTime  int64 // Unix timestamp
    TotalCollectionTime int64 // Milliseconds
    AverageUploadTime   int64 // Milliseconds
    
    // Errors
    ConfigErrors        int64
    NetworkErrors       int64
    StorageErrors       int64
    
    // Custom spans
    SpansCreated        int64
    SpansProcessed      int64
    SpansDropped        int64
    
    // Resources
    GoroutinesRunning   int64
    MemoryAllocated     int64
}
```

### Health Status

```go
type HealthStatus string

const (
    HealthStatusHealthy   HealthStatus = "healthy"
    HealthStatusDegraded  HealthStatus = "degraded" 
    HealthStatusUnhealthy HealthStatus = "unhealthy"
)
```

## Error Handling

### Error Types

#### `ConfigError`

Configuration validation errors with field-specific details.

```go
type ConfigError struct {
    Field   string
    Message string
}
```

#### `UploadError`

Upload failures with retry and status information.

```go
type UploadError struct {
    URL        string
    StatusCode int
    Attempts   int
    Underlying error
}
```

#### `ProfileError`

Profile collection errors with type and operation context.

```go
type ProfileError struct {
    Type       string
    Operation  string
    Underlying error
}
```

### Error Constants

```go
var (
    ErrAlreadyStarted     = errors.New("profiler is already started")
    ErrNotStarted         = errors.New("profiler is not started")
    ErrInvalidConfig      = errors.New("invalid configuration")
    ErrUploadFailed       = errors.New("profile upload failed")
    ErrAuthenticationFailed = errors.New("authentication failed")
    ErrUnsecureConnection = errors.New("HTTPS is required for secure uploads")
)
```

### Error Handling Example

```go
if err := profiler.Start(ctx); err != nil {
    var configErr *pprofio.ConfigError
    if errors.As(err, &configErr) {
        log.Printf("Configuration error in %s: %s", configErr.Field, configErr.Message)
    } else if errors.Is(err, pprofio.ErrAlreadyStarted) {
        log.Println("Profiler already running")
    } else {
        log.Printf("Failed to start profiler: %v", err)
    }
}
```

## Build Tags

### Production Build (default)

```bash
go build .
```

Features:
- Optimized performance
- Debug logging disabled
- Minimal overhead

### Development Build

```bash
go build -tags=dev .
go build -tags=debug .
```

Features:
- Debug logging enabled (set `PPROFIO_DEBUG=1`)
- Additional configuration validation
- Development warnings

### Build Constants

```go
const (
    IsDebugBuild bool // true in debug/dev builds
    IsDevBuild   bool // true in debug/dev builds  
    IsProdBuild  bool // true in production builds
)
```

## Examples

### Web Server Integration

```go
func main() {
    // Configure profiler
    cfg := pprofio.DefaultConfig(
        os.Getenv("PPROFIO_API_KEY"),
        "https://api.pprofio.com",
        "web-server",
    )
    cfg.Tags["service"] = "api"
    cfg.Tags["version"] = "1.0.0"
    cfg.EnableCustom = true
    
    profiler, err := pprofio.New(cfg)
    if err != nil {
        log.Fatal(err)
    }
    
    ctx := context.Background()
    ctx = pprofio.WithProfiler(ctx, profiler)
    
    if err := profiler.Start(ctx); err != nil {
        log.Fatal(err)
    }
    defer profiler.Stop()
    
    // HTTP handler with custom spans
    http.HandleFunc("/api/users", func(w http.ResponseWriter, r *http.Request) {
        ctx, span := pprofio.StartSpan(r.Context(), "http_request",
            "method", r.Method,
            "endpoint", "/api/users",
        )
        defer span.End()
        
        // Handle request...
    })
    
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

### Kubernetes Deployment

```go
func main() {
    cfg := pprofio.Config{
        APIKey:      os.Getenv("PPROFIO_API_KEY"),
        IngestURL:   "https://api.pprofio.com",
        ServiceName: os.Getenv("SERVICE_NAME"),
        Tags: map[string]string{
            "env":       os.Getenv("ENVIRONMENT"),
            "version":   os.Getenv("VERSION"),
            "namespace": os.Getenv("NAMESPACE"),
            "pod":       os.Getenv("HOSTNAME"),
        },
        EnableCPU:    true,
        EnableMemory: true,
    }
    
    profiler, err := pprofio.New(cfg)
    if err != nil {
        log.Fatal(err)
    }
    
    // Graceful shutdown
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    go func() {
        c := make(chan os.Signal, 1)
        signal.Notify(c, os.Interrupt, syscall.SIGTERM)
        <-c
        log.Println("Shutting down...")
        cancel()
    }()
    
    if err := profiler.Start(ctx); err != nil {
        log.Fatal(err)
    }
    defer profiler.Stop()
    
    // Application logic...
    <-ctx.Done()
}
```

### Custom Storage Backend

```go
type S3Storage struct {
    bucket string
    client *s3.Client
}

func (s *S3Storage) Upload(ctx context.Context, filePath string) (string, error) {
    file, err := os.Open(filePath)
    if err != nil {
        return "", err
    }
    defer file.Close()
    
    key := fmt.Sprintf("profiles/%s/%s", 
        time.Now().Format("2006/01/02"), 
        filepath.Base(filePath))
    
    _, err = s.client.PutObject(ctx, &s3.PutObjectInput{
        Bucket: aws.String(s.bucket),
        Key:    aws.String(key),
        Body:   file,
    })
    
    if err != nil {
        return "", err
    }
    
    return fmt.Sprintf("s3://%s/%s", s.bucket, key), nil
}

// Usage
cfg := pprofio.Config{
    ServiceName: "my-service",
    Storage:     &S3Storage{bucket: "my-profiles", client: s3Client},
    EnableCPU:   true,
}
```

### Health Check Endpoint

```go
http.HandleFunc("/health/profiler", func(w http.ResponseWriter, r *http.Request) {
    status := profiler.HealthCheck()
    metrics := profiler.GetMetrics()
    
    response := map[string]interface{}{
        "status":  string(status),
        "metrics": metrics,
        "build":   pprofio.GetBuildInfo(),
    }
    
    w.Header().Set("Content-Type", "application/json")
    if status != pprofio.HealthStatusHealthy {
        w.WriteHeader(http.StatusServiceUnavailable)
    }
    
    json.NewEncoder(w).Encode(response)
})
```

This API documentation provides comprehensive coverage of all public APIs and usage patterns for production integration.