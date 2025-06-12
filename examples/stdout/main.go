package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pprofio/pprofio"
)

func main() {
	// Add demo mode flag
	demoMode := flag.Bool("demo", false, "Run in demo mode for 6 seconds then exit")
	flag.Parse()

	// Configure the profiler with stdout output for testing
	cfg := pprofio.Config{
		SampleRate:      5 * time.Second, // More frequent for demonstration
		ProfileDuration: 2 * time.Second, // Shorter duration for quick output
		ServiceName:     "stdout-example-service",
		Tags:            map[string]string{"env": "testing", "version": "1.0.0"},
		EnableCPU:       true,
		EnableMemory:    true,
		OutputToStdout:  true, // Enable stdout output instead of server upload
	}

	// Create the profiler
	p, err := pprofio.New(cfg)
	if err != nil {
		fmt.Printf("Failed to create profiler: %v\n", err)
		os.Exit(1)
	}

	// Create context
	var ctx context.Context
	var cancel context.CancelFunc

	if *demoMode {
		// In demo mode, run for 6 seconds then automatically exit
		ctx, cancel = context.WithTimeout(context.Background(), 6*time.Second)
		fmt.Println("Profiler started in demo mode (will auto-exit in 6 seconds)!")
	} else {
		// In normal mode, run until interrupted
		ctx, cancel = context.WithCancel(context.Background())
		fmt.Println("Profiler started with stdout output!")
	}
	defer cancel()

	fmt.Println("All profile data and metadata will be output to stdout for testing.")
	if !*demoMode {
		fmt.Println("Press Ctrl+C to stop...")
	}
	fmt.Println("----------------------------------------")

	// Start the profiler
	if err := p.Start(ctx); err != nil {
		fmt.Printf("Failed to start profiler: %v\n", err)
		os.Exit(1)
	}

	// Create a workload to profile
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				// CPU-intensive work
				for i := 0; i < 500000; i++ {
					_ = i * i
				}

				// Memory-intensive work
				data := make([][]byte, 50)
				for i := 0; i < 50; i++ {
					data[i] = make([]byte, 512*1024) // Allocate 512KB
					for j := 0; j < len(data[i]); j++ {
						data[i][j] = byte(j)
					}
				}
				time.Sleep(500 * time.Millisecond)
			}
		}
	}()

	// Set up signal handling for graceful shutdown (only in normal mode)
	if !*demoMode {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

		select {
		case <-sigCh:
			fmt.Println("\nReceived interrupt signal, stopping profiler...")
		case <-ctx.Done():
			// Context cancelled
		}
	} else {
		// In demo mode, just wait for context to timeout
		<-ctx.Done()
		fmt.Println("Demo completed!")
	}

	// Stop the profiler
	p.Stop()
	fmt.Println("Profiler stopped.")
}
