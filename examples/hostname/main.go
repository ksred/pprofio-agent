package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/pprofio/pprofio"
)

func main() {
	// Example 1: Auto-populated hostname
	cfg1 := pprofio.Config{
		APIKey:          "test-api-key",
		IngestURL:       "http://localhost:8085/api/v1",
		SampleRate:      60 * time.Second,
		ProfileDuration: 10 * time.Second,
		Storage:         &pprofio.FileStorage{BaseDir: "./profiles"},
		ServiceName:     "hostname-example-auto",
		Tags:            map[string]string{"env": "demo"},
		EnableCPU:       true,
		EnableMemory:    true,
		// Hostname field is not set - will be auto-populated
	}

	p1, err := pprofio.New(cfg1)
	if err != nil {
		log.Fatalf("Failed to create profiler with auto hostname: %v", err)
	}

	// Get the actual system hostname to show what was auto-populated
	actualHostname, _ := os.Hostname()
	fmt.Printf("System hostname (auto-populated): %s\n", actualHostname)

	// Example 2: Custom hostname
	cfg2 := pprofio.Config{
		APIKey:          "test-api-key",
		IngestURL:       "http://localhost:8085/api/v1",
		SampleRate:      60 * time.Second,
		ProfileDuration: 10 * time.Second,
		Storage:         &pprofio.FileStorage{BaseDir: "./profiles"},
		ServiceName:     "hostname-example-custom",
		Tags:            map[string]string{"env": "demo"},
		EnableCPU:       true,
		EnableMemory:    true,
		Hostname:        "my-custom-hostname", // Explicitly set hostname
	}

	p2, err := pprofio.New(cfg2)
	if err != nil {
		log.Fatalf("Failed to create profiler with custom hostname: %v", err)
	}

	fmt.Printf("Custom hostname: %s\n", cfg2.Hostname)

	// Start profiling with auto-populated hostname
	ctx := context.Background()
	if err := p1.Start(ctx); err != nil {
		log.Fatalf("Failed to start profiler: %v", err)
	}

	fmt.Println("\nProfiler started with auto-populated hostname.")
	fmt.Println("The hostname will be included in all profile metadata.")
	fmt.Println("Check the profiles directory to see the uploaded profiles.")
	
	// You could also start p2 with custom hostname
	_ = p2 // p2 is available if you want to use custom hostname

	fmt.Println("\nPress Ctrl+C to stop...")

	// Keep the program running
	select {}
}