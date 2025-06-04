/*
Package pprofio provides continuous profiling for Go applications with minimal overhead.

# Overview

Pprofio is a lightweight, cloud-agnostic Go package that integrates with one line of code to
collect runtime performance metrics in production. It sends these metrics to the Pprofio SaaS
platform for analysis, allowing you to visualize performance with flame graphs and receive
regression alerts.

# Features

  - Simple Integration: Import with minimal configuration
  - Low Overhead: <1% CPU/memory impact on your application
  - Multiple Metrics:
    - CPU profiles (stack traces, CPU time)
    - Memory profiles (allocations, heap size)
    - Goroutine profiles (count, stack traces)
    - Mutex contention profiles (wait time)
    - Block profiles (I/O, syscall delays)
    - Custom instrumentation (user-defined spans)
  - Flexible Storage: Upload to Pprofio SaaS or store locally
  - Secure: HTTPS and API key authentication

# Getting Started

Install the package:

	go get github.com/pprofio/pprofio

Basic usage:

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

# Configuration Options

The Config struct allows you to customize the profiler's behavior:

  - APIKey: Your Pprofio API key for authentication
  - IngestURL: The Pprofio API endpoint (usually https://api.pprofio.com)
  - SampleRate: How often to collect profiles (default: 60s)
  - ProfileDuration: Length of each sample (default: 10s for CPU/mutex/block)
  - Storage: Choose HTTPStorage, FileStorage, or custom implementation
  - ServiceName: Identifier for your application
  - Tags: Additional metadata (e.g., "env=prod", "version=1.2.3")
  - MemProfileRate: Controls memory profiling detail (default: 4096)
  - MutexFraction: Controls mutex profiling frequency (default: 5)
  - BlockProfileRate: Controls block profiling frequency (default: 100)
  - EnableCPU, EnableMemory, etc.: Toggle specific profile types

# Custom Instrumentation

You can add custom spans to track specific operations:

	ctx, span := pprofio.StartSpan(r.Context(), "handle_request", "endpoint", "/api/v1")
	defer span.End()

To ensure spans are collected, attach the profiler to your context:

	ctx = pprofio.WithProfiler(ctx, p)

# Custom Storage

Implement the Storage interface to create your own storage backend:

	type MyStorage struct {
		// Your fields here
	}

	func (s *MyStorage) Upload(ctx context.Context, filePath string) (string, error) {
		// Your implementation here
	}

# Performance Considerations

The profiler is designed to have minimal impact (<1% CPU) on your application:

  - CPU profiles: Collected for 10s every 60s (configurable)
  - Memory profiles: Snapshots taken every 60s
  - Goroutine profiles: Snapshots taken every 60s
  - Mutex profiles: Collected for 10s every 60s
  - Block profiles: Collected for 10s every 60s
  - Custom spans: Continuously collected with minimal overhead

Configure sampling rates and profile durations to balance detail with performance impact.
*/
package pprofio