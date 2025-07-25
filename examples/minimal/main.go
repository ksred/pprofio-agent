package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/pprofio/pprofio"
)

func main() {
	// Create a minimal configuration - no IngestURL needed!
	// It will automatically use https://api.pprofio.com/api/v1
	cfg := pprofio.Config{
		APIKey:      "your-api-key-here",
		ServiceName: "minimal-example",
		// Optional: specify profile types (defaults to CPU and Memory)
		EnableCPU:       true,
		EnableMemory:    true,
		EnableGoroutine: true,
	}

	// Create the profiler
	profiler, err := pprofio.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create profiler: %v", err)
	}

	// Start profiling
	ctx := context.Background()
	if err := profiler.Start(ctx); err != nil {
		log.Fatalf("Failed to start profiler: %v", err)
	}

	fmt.Println("Profiler started with default URL:", cfg.IngestURL)
	fmt.Println("Profiling will upload to:", cfg.IngestURL+"/upload")

	// Run for 2 minutes to collect some profiles
	fmt.Println("Running for 2 minutes...")
	time.Sleep(2 * time.Minute)

	// Stop profiling
	profiler.Stop()
	fmt.Println("Profiler stopped")
}
