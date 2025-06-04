# Pprofio

Pprofio is a lightweight, continuous profiling agent for Go applications. It collects runtime performance metrics with minimal overhead (<1% CPU) and uploads them to the Pprofio SaaS platform for analysis.

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
    p := pprofio.New(cfg)
    ctx := context.Background()
    p.Start(ctx)
    
    // Your application code here...
}
```

## Documentation

For complete documentation, visit [docs.pprofio.com](https://docs.pprofio.com).

## License

MIT