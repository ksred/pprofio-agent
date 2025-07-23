package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/pprofio/pprofio"
)

func main() {
	// Configure the profiler with HTTP metrics enabled
	cfg := pprofio.Config{
		APIKey:          "test-api-key",                 // Not used with file storage but required
		IngestURL:       "http://localhost:8085/api/v1", // Not used with file storage but required
		SampleRate:      10 * time.Second,               // More frequent for demonstration
		ProfileDuration: 5 * time.Second,
		ServiceName:     "example-service",
		Tags:            map[string]string{"env": "local", "version": "1.0.0", "team": "backend"},
		EnableCPU:       true,
		EnableMemory:    true,
		EnableGoroutine: true,
		EnableMutex:     true,
		EnableBlock:     true,
		MemProfileRate:  4096,
		Env:             "local",

		// 🆕 HTTP Metrics Configuration - now integrated!
		EnableHTTPMetrics:           true,
		HTTPMetricsSampleRate:       1.0, // 100% sampling for demo
		HTTPMetricsExcludePaths:     []string{"/health", "/metrics"},
		HTTPMetricsIncludeHeaders:   []string{"User-Agent", "Content-Type"},
		HTTPMetricsCollectUserAgent: true,
		HTTPMetricsHashIPs:          false, // Don't hash IPs in local demo
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
	fmt.Println("HTTP metrics collection enabled!")

	// 🆕 Start HTTP server with integrated middleware - ONE LINE!
	go startHTTPServer(p)

	fmt.Println("HTTP server running on :9090 with integrated profiling middleware")
	fmt.Println("Test endpoints:")
	fmt.Println("  curl http://localhost:9090/")
	fmt.Println("  curl http://localhost:9090/api/users")
	fmt.Println("  curl http://localhost:9090/api/orders")
	fmt.Println("  curl http://localhost:9090/health  # (excluded from metrics)")
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

// startHTTPServer demonstrates HTTP middleware integration
func startHTTPServer(profiler *pprofio.Profiler) {
	mux := http.NewServeMux()

	// Define sample endpoints
	mux.HandleFunc("/", handleHome)
	mux.HandleFunc("/api/users", handleUsers)
	mux.HandleFunc("/api/orders", handleOrders)
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/metrics", handleMetrics)

	// 🎯 Apply pprofio HTTP middleware - ONE LINE INTEGRATION!
	wrappedHandler := profiler.HTTPMiddleware()(mux)

	// Alternative router-specific methods:
	// wrappedHandler := profiler.ForStandardHTTP()(mux)  // Same as above
	// wrappedHandler := profiler.ForChi()(mux)           // For Chi router
	// wrappedHandler := profiler.ForGorillaMux()(mux)    // For Gorilla Mux

	server := &http.Server{
		Addr:    ":9090",
		Handler: wrappedHandler,
	}

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Printf("HTTP server error: %v\n", err)
	}
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	// Simulate some processing time
	time.Sleep(50 * time.Millisecond)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Welcome to pprofio example!"))
}

func handleUsers(w http.ResponseWriter, r *http.Request) {
	// Simulate database work
	time.Sleep(100 * time.Millisecond)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"users": ["alice", "bob", "charlie"]}`))
}

func handleOrders(w http.ResponseWriter, r *http.Request) {
	// Simulate longer processing
	time.Sleep(200 * time.Millisecond)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"orders": [{"id": 1, "amount": 99.99}]}`))
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	// This endpoint is excluded from metrics collection
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func handleMetrics(w http.ResponseWriter, r *http.Request) {
	// This endpoint is also excluded from metrics collection
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("# Metrics would go here"))
}
