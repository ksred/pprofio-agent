package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pprofio/pprofio"
)

func main() {
	// Configure the profiler
	cfg := pprofio.Config{
		APIKey:          "test-api-key",                 // Not used with file storage but required
		IngestURL:       "http://localhost:8080/api/v1", // Not used with file storage but required
		SampleRate:      10 * time.Second,               // More frequent for demonstration
		ProfileDuration: 5 * time.Second,
		ServiceName:     "example-service",
		Tags:            map[string]string{"env": "local", "version": "1.0.0"},
		EnableCPU:       true,
		EnableMemory:    true,
		EnableGoroutine: true,
		EnableMutex:     true,
		EnableBlock:     true,
		MemProfileRate:  4096,
		Env:             "local",
	}

	// Create the profiler
	p, err := pprofio.New(cfg)
	if err != nil {
		fmt.Printf("Failed to create profiler: %v\n", err)
		os.Exit(1)
	}

	// Create context that can be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the profiler
	if err := p.Start(ctx); err != nil {
		fmt.Printf("Failed to start profiler: %v\n", err)
		os.Exit(1)
	}
	defer p.Stop()

	fmt.Println("Profiler started! Collecting CPU, memory, goroutine, mutex, and block profiles every 10 seconds.")
	fmt.Println("Press Ctrl+C to stop...")

	// Create a workload to profile
	go func() {
		for {
			// CPU-intensive work
			for i := 0; i < 1000000; i++ {
				_ = i * i
			}

			// Memory-intensive work
			data := make([][]byte, 100)
			for i := 0; i < 100; i++ {
				data[i] = make([]byte, 1024*1024) // Allocate 1MB
				for j := 0; j < len(data[i]); j++ {
					data[i][j] = byte(j)
				}
			}
			time.Sleep(time.Second)
		}
	}()

	// Handle graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	fmt.Println("\nShutting down profiler...")
}
