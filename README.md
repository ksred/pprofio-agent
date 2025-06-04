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
- **Flexible Storage**: Upload to Pprofio SaaS or store locally
- **Secure**: HTTPS and API key authentication

## Installation

```bash
go get github.com/pprofio/pprofio
```

### Requirements

- Go 1.20 or later
- Linux, macOS, or Windows

## Quick Start

```go
package main

import (
    "context"
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

### Configuration Options

```go
type Config struct {
    APIKey          string            // Required: Your Pprofio API key
    IngestURL       string            // Required: Pprofio ingest endpoint
    ServiceName     string            // Required: Name of your service
    SampleRate      time.Duration     // Profile collection interval (default: 60s)
    Tags            map[string]string // Custom tags for profiles
    EnableCPU       bool              // Enable CPU profiling (default: true)
    EnableMemory    bool              // Enable memory profiling (default: true)
    ProfileDuration time.Duration     // Profile collection duration (default: 10s)
}
```

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

# Run tests
go test -v -race ./...

# Run linting
golangci-lint run --timeout=10m
```

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
| Phase 2 | Goroutine  | Goroutine count, stack traces      | Snapshot/60s    | Area charts, leak alerts     | <0.2%    |
| Phase 2 | Mutex      | Contention events, wait time       | 10s every 60s   | Bar charts                   | <0.2%    |
| Phase 3 | Block      | Blocking events, duration          | 10s every 60s   | Pie/timeline charts          | <0.2%    |
| Phase 3 | Custom       | User-defined spans, tags, stacks   | Continuous      | Flame graphs, timelines      | <0.1%    |

## License

MIT