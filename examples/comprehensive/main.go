package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/pprofio/pprofio"
)

func main() {
	// Configure the profiler with all profile types enabled
	cfg := pprofio.ComprehensiveConfig("test-api-key", "http://localhost:8080/api/v1", "comprehensive-example")
	cfg.SampleRate = 5 * time.Second // Collect profiles every 5 seconds
	cfg.ProfileDuration = 2 * time.Second
	cfg.Tags = map[string]string{
		"env":     "demo",
		"version": "1.0.0",
		"feature": "comprehensive-profiling",
	}
	// Use stdout for this example to see the profiles
	cfg.OutputToStdout = true

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

	fmt.Println("🚀 Comprehensive profiler started!")
	fmt.Println("📊 Collecting all profile types: CPU, Memory, Goroutine, Mutex, Block, and Custom Spans")
	fmt.Println("⏱️  Profiles collected every 5 seconds")
	fmt.Println("Press Ctrl+C to stop...")
	fmt.Println()

	// Create various workloads to generate profile data

	// 1. CPU-intensive workload
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				// CPU-intensive mathematical operations
				for i := 0; i < 1000000; i++ {
					_ = i * i * i
				}
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()

	// 2. Memory-intensive workload
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				// Allocate and release memory
				data := make([][]byte, 100)
				for i := 0; i < 100; i++ {
					data[i] = make([]byte, 1024*1024) // 1MB per allocation
					// Fill with data to ensure it's not optimized away
					for j := 0; j < len(data[i]); j++ {
						data[i][j] = byte(j % 256)
					}
				}
				time.Sleep(200 * time.Millisecond)
				// Let data go out of scope to trigger GC
				runtime.GC()
			}
		}
	}()

	// 3. Goroutine workload (creates many short-lived goroutines)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				var wg sync.WaitGroup
				for i := 0; i < 50; i++ {
					wg.Add(1)
					go func(id int) {
						defer wg.Done()
						// Simulate some work
						time.Sleep(10 * time.Millisecond)
						_ = id * 2
					}(i)
				}
				wg.Wait()
				time.Sleep(300 * time.Millisecond)
			}
		}
	}()

	// 4. Mutex contention workload
	var contentionMutex sync.Mutex
	sharedCounter := 0

	for i := 0; i < 5; i++ {
		go func(worker int) {
			for {
				select {
				case <-ctx.Done():
					return
				default:
					contentionMutex.Lock()
					// Hold the lock for a bit to create contention
					time.Sleep(5 * time.Millisecond)
					sharedCounter++
					contentionMutex.Unlock()
					time.Sleep(10 * time.Millisecond)
				}
			}
		}(i)
	}

	// 5. Channel blocking workload
	blockingChan := make(chan int, 1) // Small buffer to create blocking

	// Producer
	go func() {
		counter := 0
		for {
			select {
			case <-ctx.Done():
				return
			case blockingChan <- counter: // This will block when buffer is full
				counter++
			}
		}
	}()

	// Slow consumer (creates blocking)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case val := <-blockingChan:
				// Slow processing to create blocking
				time.Sleep(50 * time.Millisecond)
				_ = val
			}
		}
	}()

	// 6. Custom span tracing workload
	ctx = pprofio.WithProfiler(ctx, p)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				// Simulate API request processing
				spanCtx, span := pprofio.StartSpan(ctx, "api-request",
					"endpoint", "/users",
					"method", "GET",
					"user_id", "12345")

				// Simulate processing time
				time.Sleep(25 * time.Millisecond)

				// Nested span for database operation
				_, dbSpan := pprofio.StartSpan(spanCtx, "database-query",
					"table", "users",
					"operation", "SELECT")
				time.Sleep(15 * time.Millisecond)
				dbSpan.End()

				span.End()
				time.Sleep(500 * time.Millisecond)
			}
		}
	}()

	// Print periodic status updates
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				fmt.Printf("🔄 System stats - Goroutines: %d, Counter: %d\n",
					runtime.NumGoroutine(), sharedCounter)
			}
		}
	}()

	// Handle graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	fmt.Println("\n🛑 Received shutdown signal, stopping profiler...")
	cancel() // Cancel context to stop all goroutines

	// Give a moment for final profile collection
	time.Sleep(1 * time.Second)

	fmt.Println("✅ Profiler stopped successfully!")
	fmt.Printf("📈 Final stats - Goroutines: %d, Counter: %d\n",
		runtime.NumGoroutine(), sharedCounter)
}
