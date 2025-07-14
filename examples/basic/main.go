package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/pprofio/pprofio"
)

func main() {
	// Configure the profiler
	cfg := pprofio.Config{
		APIKey:          "test-api-key",                 // Not used with file storage but required
		IngestURL:       "http://localhost:8085/api/v1", // Not used with file storage but required
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

	// Create multiple workloads to demonstrate different profiling capabilities
	// CPU-intensive workload
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				cpuIntensiveWork()
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()

	// Memory-intensive workload
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				memoryIntensiveWork()
				time.Sleep(2 * time.Second)
			}
		}
	}()

	// Mutex contention workload
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				var wg sync.WaitGroup
				for i := 0; i < 3; i++ {
					wg.Add(1)
					go func(id int) {
						defer wg.Done()
						mutexContentionWork(id)
					}(i)
				}
				wg.Wait()
				time.Sleep(3 * time.Second)
			}
		}
	}()

	// Blocking operations workload
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				blockingWork(1)
				time.Sleep(2 * time.Second)
			}
		}
	}()

	// Goroutine creation workload
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				goroutineContentionWork()
				time.Sleep(5 * time.Second)
			}
		}
	}()

	// Handle graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	fmt.Println("\nShutting down profiler...")
}
