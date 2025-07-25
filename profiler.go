package pprofio

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"runtime/pprof"
	"sync"
	"time"
)

type profileType string

const (
	profileTypeCPU       profileType = "cpu"
	profileTypeMemory    profileType = "memory"
	profileTypeGoroutine profileType = "goroutine"
	profileTypeMutex     profileType = "mutex"
	profileTypeBlock     profileType = "block"
	profileTypeCustom    profileType = "custom"
)

const (
	// DefaultSpanBufferSize is the default buffer size for custom spans
	DefaultSpanBufferSize = 1000
	// MaxUniqueSpanNames limits the number of unique span names to prevent unbounded memory growth
	MaxUniqueSpanNames = 10000
)

type Profiler struct {
	config      Config
	mu          sync.Mutex
	stopCh      chan struct{}
	wg          sync.WaitGroup
	initialized bool
	spanCh      chan *Span

	// Store original runtime values for restoration
	originalMemProfileRate   int
	originalMutexFraction    int
	originalBlockProfileRate int

	// HTTP Metrics
	metricsCollector *MetricsCollector
	httpStorage      *HTTPMetricsStorage
}

// newProfiler is the internal constructor used by New
func newProfiler(config Config) (*Profiler, error) {
	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	p := &Profiler{
		config: config,
		stopCh: make(chan struct{}),
		spanCh: make(chan *Span, DefaultSpanBufferSize), // Buffer for custom spans
	}

	// Initialize HTTP metrics collector if enabled
	if config.EnableHTTPMetrics {
		// Create HTTP metrics storage backend
		if !config.OutputToStdout {
			p.httpStorage = NewHTTPMetricsStorage(
				config.IngestURL,
				config.APIKey,
				config.HTTPMetricsBatchSize,
				config.HTTPMetricsFlushInterval,
			)
		}

		middlewareConfig := MiddlewareConfig{
			Enabled:          true,
			SampleRate:       config.HTTPMetricsSampleRate,
			ExcludedPaths:    config.HTTPMetricsExcludePaths,
			IncludeHeaders:   config.HTTPMetricsIncludeHeaders,
			MaxPayloadSize:   config.HTTPMetricsMaxPayloadSize,
			CollectUserAgent: config.HTTPMetricsCollectUserAgent,
			HashIPs:          config.HTTPMetricsHashIPs,
		}

		p.metricsCollector = NewMetricsCollector(middlewareConfig)

		// Set up the metrics handler to send HTTP metrics through the profiler's storage
		p.metricsCollector.SetMetricsHandler(p.handleHTTPMetrics)
	}

	return p, nil
}

func (p *Profiler) collectProfiles(ctx context.Context, profileType profileType) {
	defer p.wg.Done()

	ticker := time.NewTicker(p.config.SampleRate)
	defer ticker.Stop()

	// Collect one profile immediately at startup
	if err := p.collectProfile(ctx, profileType); err != nil {
		fmt.Fprintf(os.Stderr, "Error collecting %s profile: %v\n", profileType, err)
	}

	for {
		select {
		case <-ticker.C:
			if err := p.collectProfile(ctx, profileType); err != nil {
				fmt.Fprintf(os.Stderr, "Error collecting %s profile: %v\n", profileType, err)
			}
		case <-p.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

func (p *Profiler) collectProfile(ctx context.Context, profileType profileType) error {
	switch profileType {
	case profileTypeCPU:
		return p.collectCPU(ctx)
	case profileTypeMemory:
		return p.collectMemory(ctx)
	case profileTypeGoroutine:
		return p.collectGoroutine(ctx)
	case profileTypeMutex:
		return p.collectMutex(ctx)
	case profileTypeBlock:
		return p.collectBlock(ctx)
	case profileTypeCustom:
		// Custom profiles are handled separately
		return nil
	default:
		return fmt.Errorf("unknown profile type: %s", profileType)
	}
}

func (p *Profiler) collectCPU(ctx context.Context) error {
	f, err := os.CreateTemp("", "cpu.pprof")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(f.Name())

	if err := pprof.StartCPUProfile(f); err != nil {
		f.Close()
		return fmt.Errorf("failed to start CPU profile: %w", err)
	}

	// Profile for the configured duration
	profileCtx, cancel := context.WithTimeout(ctx, p.config.ProfileDuration)
	defer cancel()

	select {
	case <-profileCtx.Done():
		// Profile duration completed
	case <-p.stopCh:
	}

	pprof.StopCPUProfile()
	f.Close()

	return p.uploadProfile(ctx, f.Name(), string(profileTypeCPU))
}

func (p *Profiler) collectMemory(ctx context.Context) error {
	f, err := os.CreateTemp("", "memory.pprof")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(f.Name())

	// Force garbage collection to get accurate memory profile
	runtime.GC()

	if err := pprof.WriteHeapProfile(f); err != nil {
		f.Close()
		return fmt.Errorf("failed to write memory profile: %w", err)
	}

	f.Close()

	return p.uploadProfile(ctx, f.Name(), string(profileTypeMemory))
}

func (p *Profiler) collectGoroutine(ctx context.Context) error {
	f, err := os.CreateTemp("", "goroutine.pprof")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(f.Name())

	if err := pprof.Lookup("goroutine").WriteTo(f, 0); err != nil {
		f.Close()
		return fmt.Errorf("failed to write goroutine profile: %w", err)
	}

	f.Close()

	return p.uploadProfile(ctx, f.Name(), string(profileTypeGoroutine))
}

func (p *Profiler) collectMutex(ctx context.Context) error {
	f, err := os.CreateTemp("", "mutex.pprof")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(f.Name())

	if err := pprof.Lookup("mutex").WriteTo(f, 0); err != nil {
		f.Close()
		return fmt.Errorf("failed to write mutex profile: %w", err)
	}

	f.Close()

	return p.uploadProfile(ctx, f.Name(), string(profileTypeMutex))
}

func (p *Profiler) collectBlock(ctx context.Context) error {
	f, err := os.CreateTemp("", "block.pprof")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(f.Name())

	if err := pprof.Lookup("block").WriteTo(f, 0); err != nil {
		f.Close()
		return fmt.Errorf("failed to write block profile: %w", err)
	}

	f.Close()

	return p.uploadProfile(ctx, f.Name(), string(profileTypeBlock))
}

func (p *Profiler) uploadProfile(ctx context.Context, filePath, _ string) error {
	// Upload the profile and parse the returned JSON response
	uploadResp, err := p.config.Storage.Upload(ctx, filePath)
	if err != nil {
		// Log additional details about the storage configuration
		if httpStorage, ok := p.config.Storage.(*HTTPStorage); ok {
			fmt.Fprintf(os.Stderr, "pprofio: upload failed for URL=%s, Env=%s\n", httpStorage.URL, httpStorage.Env)
		}

		return fmt.Errorf("failed to upload profile: %w", err)
	}

	// Parse the JSON response
	var response struct {
		ProfileID  string `json:"profile_id"`
		ProfileURL string `json:"profile_url"`
		Type       string `json:"type"`
	}

	if err := json.Unmarshal([]byte(uploadResp), &response); err != nil {
		return fmt.Errorf("failed to parse upload response: %w", err)
	}

	profileURL := response.ProfileURL
	profileID := response.ProfileID
	profileTypeFromResp := response.Type

	// Send metadata with the returned profile_url
	metadata := map[string]string{
		"profile_id":  profileID,
		"profile_url": profileURL, // Use the returned URL instead of generated UUID
		"service":     p.config.ServiceName,
		"type":        profileTypeFromResp,
		"timestamp":   fmt.Sprintf("%d", time.Now().Unix()),
		"hostname":    p.config.Hostname,
	}

	// Add user-provided tags
	for k, v := range p.config.Tags {
		metadata[k] = v
	}

	// If using stdout mode, output metadata to stdout as well
	if p.config.OutputToStdout {
		if stdoutStorage, ok := p.config.Storage.(*StdoutStorage); ok {
			if err := stdoutStorage.OutputMetadata(metadata); err != nil {
				return fmt.Errorf("failed to output metadata to stdout: %w", err)
			}
		}
	} else {
		// Send metadata to server in normal mode
		if err := p.sendMetadata(ctx, metadata); err != nil {
			return fmt.Errorf("failed to send metadata: %w", err)
		}
	}

	return nil
}

// handleHTTPMetrics processes HTTP metrics and sends them to the configured storage
func (p *Profiler) handleHTTPMetrics(metrics *RequestMetrics) {
	// Add service metadata from config
	if p.config.ServiceName != "" {
		metrics.Service = p.config.ServiceName
	}

	if env, ok := p.config.Tags["environment"]; ok {
		metrics.Environment = env
	}

	if version, ok := p.config.Tags["version"]; ok {
		metrics.Version = version
	}

	if region, ok := p.config.Tags["region"]; ok {
		metrics.Region = region
	}

	// Add remaining tags
	if metrics.Tags == nil {
		metrics.Tags = make(map[string]string)
	}

	for k, v := range p.config.Tags {
		// Skip the ones we already moved to top-level fields
		if k != "environment" && k != "version" && k != "region" {
			metrics.Tags[k] = v
		}
	}

	// Send to appropriate storage
	if p.config.OutputToStdout {
		// Output to stderr in development mode
		fmt.Fprintf(os.Stderr, "HTTP Metric: %s %s %d %v\n",
			metrics.Method, metrics.Path, metrics.StatusCode, metrics.Duration)
	} else if p.httpStorage != nil {
		// Send to Pprofio API endpoints
		if err := p.httpStorage.SubmitMetric(metrics); err != nil {
			fmt.Fprintf(os.Stderr, "Error submitting HTTP metric: %v\n", err)
		}
	}
}

// HTTPMiddleware returns HTTP middleware that collects request metrics
// This integrates HTTP metrics collection directly into the profiler
func (p *Profiler) HTTPMiddleware() func(http.Handler) http.Handler {
	if !p.config.EnableHTTPMetrics || p.metricsCollector == nil {
		// Return a no-op middleware if HTTP metrics are disabled
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	return p.metricsCollector.HTTPMiddleware()
}
