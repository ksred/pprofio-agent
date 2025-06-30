package pprofio

import (
	"context"
	"encoding/json"
	"fmt"
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

// Profiler manages continuous collection and upload of runtime performance profiles.
// It coordinates multiple profile types (CPU, memory, goroutines, etc.) and handles
// their scheduling, collection, and upload to configured storage backends.
//
// The Profiler is designed for production use with minimal overhead (<1% CPU impact)
// and automatic error handling. It can be safely started and stopped multiple times.
//
// Example usage:
//
//	cfg := pprofio.DefaultConfig("api-key", "https://api.pprofio.com", "my-service")
//	profiler, err := pprofio.New(cfg)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	ctx := context.Background()
//	if err := profiler.Start(ctx); err != nil {
//		log.Fatal(err)
//	}
//	defer profiler.Stop()
type Profiler struct {
	config      Config
	mu          sync.Mutex
	stopCh      chan struct{}
	wg          sync.WaitGroup
	initialized bool
	spanCh      chan *Span
	metrics     Metrics

	// Store original runtime values for restoration
	originalMemProfileRate   int
	originalMutexFraction    int
	originalBlockProfileRate int
}

// newProfiler is the internal constructor used by New
func newProfiler(config Config) (*Profiler, error) {
	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	p := &Profiler{
		config: config,
		stopCh: make(chan struct{}),
		spanCh: make(chan *Span, 1000), // Buffer for custom spans
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
		// Profiler is stopping
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

func (p *Profiler) uploadProfile(ctx context.Context, filePath, profileType string) error {
	// Upload the profile and parse the returned response
	uploadResp, err := p.config.Storage.Upload(ctx, filePath)
	if err != nil {
		return fmt.Errorf("failed to upload profile: %w", err)
	}

	// Try to parse as JSON first, fall back to plain string for backward compatibility
	var response struct {
		ProfileID  string `json:"profile_id"`
		ProfileURL string `json:"profile_url"`
		Type       string `json:"type"`
	}
	
	var profileURL, profileID, profileTypeFromResp string
	
	if err := json.Unmarshal([]byte(uploadResp), &response); err != nil {
		// Not JSON, treat as plain string (legacy HTTP endpoints)
		profileURL = uploadResp
		profileID = fmt.Sprintf("legacy-%d", time.Now().Unix())
		profileTypeFromResp = profileType
	} else {
		// JSON response
		profileURL = response.ProfileURL
		profileID = response.ProfileID
		profileTypeFromResp = response.Type
	}

	// Send metadata with the returned profile_url
	metadata := map[string]string{
		"profile_id":  profileID,
		"profile_url": profileURL, // Use the returned URL instead of generated UUID
		"service":     p.config.ServiceName,
		"type":        profileTypeFromResp,
		"timestamp":   fmt.Sprintf("%d", time.Now().Unix()),
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
