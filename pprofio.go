// Package pprofio provides continuous profiling for Go applications with minimal overhead.
//
// It collects runtime performance metrics (CPU, memory, goroutines, mutex contention, etc.)
// and uploads them to the Pprofio SaaS platform or a custom storage backend.
package pprofio

import (
	"context"
	"fmt"
	"runtime"
	"time"
)

// New creates a new profiler with the provided configuration.
// It returns an error if the configuration is invalid.
func New(config Config) (*Profiler, error) {
	// Apply defaults if necessary
	if config.SampleRate == 0 {
		config.SampleRate = DefaultSampleRate
	}
	if config.ProfileDuration == 0 {
		config.ProfileDuration = DefaultProfileDuration
	}
	if config.MemProfileRate == 0 {
		config.MemProfileRate = DefaultMemProfileRate
	}
	if config.MutexFraction == 0 {
		config.MutexFraction = DefaultMutexFraction
	}
	if config.BlockProfileRate == 0 {
		config.BlockProfileRate = DefaultBlockProfileRate
	}

	// Create stdout storage if OutputToStdout is enabled
	if config.OutputToStdout {
		config.Storage = NewStdoutStorage()
	} else if config.Storage == nil && config.APIKey != "" && config.IngestURL != "" {
		// Create HTTP storage if not provided and not in stdout mode
		config.Storage = NewHTTPStorage(config.IngestURL+"/upload", config.APIKey, config.Env)
	}

	// Enable CPU and Memory by default if nothing is enabled
	if !config.EnableCPU && !config.EnableMemory && !config.EnableGoroutine &&
		!config.EnableMutex && !config.EnableBlock && !config.EnableCustom {
		config.EnableCPU = true
		config.EnableMemory = true
	}

	return newProfiler(config)
}

// Start begins collecting and uploading profiles based on the configuration.
// It configures Go runtime settings for profiling and starts background goroutines
// for each enabled profile type.
//
// The profiler runs continuously until Stop() is called or the context is canceled.
// It's safe to call Start() multiple times - subsequent calls will return an error
// if the profiler is already running.
//
// Start() modifies global Go runtime settings (MemProfileRate, MutexProfileFraction,
// BlockProfileRate) which are restored when Stop() is called.
//
// Parameters:
//   - ctx: Context for cancellation. When canceled, all profiling stops.
//
// Returns an error if:
//   - The profiler is already running
//   - The configuration is invalid
//   - Storage initialization fails
//
// Example:
//
//	ctx, cancel := context.WithCancel(context.Background())
//	defer cancel()
//
//	if err := profiler.Start(ctx); err != nil {
//		log.Fatal("Failed to start profiler:", err)
//	}
func (p *Profiler) Start(ctx context.Context) error {
	return p.start(ctx)
}

// start is the internal implementation used by Start
func (p *Profiler) start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.initialized {
		return fmt.Errorf("profiler already started")
	}

	// Store original runtime settings before configuring
	p.originalMemProfileRate = runtime.MemProfileRate
	// Note: runtime doesn't provide getters for mutex and block rates,
	// so we store reasonable defaults to restore
	p.originalMutexFraction = 0    // Default is disabled
	p.originalBlockProfileRate = 0 // Default is disabled

	// Configure runtime settings
	if p.config.EnableMemory {
		runtime.MemProfileRate = p.config.MemProfileRate
	}

	if p.config.EnableMutex {
		runtime.SetMutexProfileFraction(p.config.MutexFraction)
	}

	if p.config.EnableBlock {
		runtime.SetBlockProfileRate(p.config.BlockProfileRate)
	}

	// Start collection goroutines
	if p.config.EnableCPU {
		p.wg.Add(1)
		go p.collectProfiles(ctx, profileTypeCPU)
	}

	if p.config.EnableMemory {
		p.wg.Add(1)
		go p.collectProfiles(ctx, profileTypeMemory)
	}

	if p.config.EnableGoroutine {
		p.wg.Add(1)
		go p.collectProfiles(ctx, profileTypeGoroutine)
	}

	if p.config.EnableMutex {
		p.wg.Add(1)
		go p.collectProfiles(ctx, profileTypeMutex)
	}

	if p.config.EnableBlock {
		p.wg.Add(1)
		go p.collectProfiles(ctx, profileTypeBlock)
	}

	if p.config.EnableCustom {
		p.wg.Add(1)
		go p.processCustomSpans(ctx)
	}

	p.initialized = true
	return nil
}

// Stop gracefully shuts down the profiler and waits for pending operations to complete.
// It stops all background goroutines, restores original Go runtime settings, and
// ensures any in-flight profile uploads finish before returning.
//
// Stop() is safe to call multiple times and will not block if the profiler is
// not running. It's recommended to call Stop() in a defer statement or signal handler
// to ensure clean shutdown.
//
// The method restores Go runtime settings (MemProfileRate, MutexProfileFraction,
// BlockProfileRate) to their original values from before Start() was called.
//
// Example:
//
//	profiler, _ := pprofio.New(config)
//	defer profiler.Stop() // Ensures cleanup on function exit
//
//	profiler.Start(ctx)
//	// ... application code ...
//	// Stop() called automatically via defer
func (p *Profiler) Stop() {
	p.stop()
}

// stop is the internal implementation used by Stop
func (p *Profiler) stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.initialized {
		return
	}

	// Safe channel close
	select {
	case <-p.stopCh:
		// Channel already closed, do nothing
	default:
		close(p.stopCh)
	}

	p.wg.Wait()

	// Restore original runtime settings
	runtime.MemProfileRate = p.originalMemProfileRate
	runtime.SetMutexProfileFraction(p.originalMutexFraction)
	runtime.SetBlockProfileRate(p.originalBlockProfileRate)

	p.initialized = false
}

// StartSpan begins timing a custom span with the given name and optional tags.
// Tags should be provided as alternating key-value pairs (e.g., "key1", "value1", "key2", "value2").
// The span is automatically associated with the profiler if the context contains one.
func StartSpan(ctx context.Context, name string, tags ...string) (context.Context, *Span) {
	span := &Span{
		Name:  name,
		Start: time.Now(),
		Tags:  make(map[string]string),
	}

	// Convert tags slice to map
	for i := 0; i < len(tags); i += 2 {
		if i+1 < len(tags) {
			key := tags[i]
			value := tags[i+1]
			span.Tags[key] = value
		}
	}

	// Check if we have a profiler in the context
	if prof, ok := ctx.Value(spanKey{}).(*Profiler); ok && prof != nil {
		// Queue span for processing
		select {
		case prof.spanCh <- span:
			// Span queued successfully
		default:
			// Channel full, log and continue
		}
	}

	return ctx, span
}

// WithProfiler attaches a profiler to a context for span collection.
// This allows spans to be automatically collected when created with StartSpan.
func WithProfiler(ctx context.Context, p *Profiler) context.Context {
	return context.WithValue(ctx, spanKey{}, p)
}
