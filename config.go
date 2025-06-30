package pprofio

import (
	"errors"
	"time"
)

// Default configuration values for the profiler.
// These values provide a good balance between performance insight and overhead.
const (
	// DefaultSampleRate is the default interval between profile collections (60 seconds).
	// This provides regular insight into application performance without significant overhead.
	DefaultSampleRate = 60 * time.Second

	// DefaultProfileDuration is the default duration for CPU, mutex, and block profiles (10 seconds).
	// This captures enough data for meaningful analysis while minimizing impact.
	DefaultProfileDuration = 10 * time.Second

	// DefaultMemProfileRate controls the fraction of memory allocations recorded.
	// Setting it to 4096 means ~1 in 4096 allocations are recorded.
	DefaultMemProfileRate = 4096

	// DefaultMutexFraction controls the fraction of mutex contention events recorded.
	// Setting it to 5 means ~1 in 5 mutex events are recorded.
	DefaultMutexFraction = 5

	// DefaultBlockProfileRate controls the fraction of blocking events recorded.
	// Setting it to 100 means ~1 in 100 blocking operations are recorded.
	DefaultBlockProfileRate = 100
)

// Config holds all configuration options for the profiler.
// It provides fine-grained control over profiling behavior, performance impact,
// and data destination.
//
// Example usage:
//
//	cfg := pprofio.Config{
//		APIKey:      "your-api-key",
//		IngestURL:   "https://api.pprofio.com",
//		ServiceName: "my-service",
//		Tags:        map[string]string{"env": "production", "version": "1.0.0"},
//		EnableCPU:   true,
//		EnableMemory: true,
//	}
type Config struct {
	// APIKey is your Pprofio API key for authentication.
	// Required unless OutputToStdout is true.
	APIKey string

	// IngestURL is the Pprofio API endpoint for uploading profiles.
	// Typically "https://api.pprofio.com" for the hosted service.
	// Required unless OutputToStdout is true.
	IngestURL string

	// SampleRate controls how often profiles are collected.
	// Default: 60 seconds. Lower values increase overhead but provide more data.
	SampleRate time.Duration

	// ProfileDuration controls how long CPU, mutex, and block profiles run.
	// Default: 10 seconds. Longer durations provide more data but increase overhead.
	ProfileDuration time.Duration

	// Storage defines where profiles are uploaded. If nil, HTTPStorage will be created
	// automatically using APIKey and IngestURL.
	Storage Storage

	// ServiceName identifies your application in the Pprofio dashboard.
	// This should be a consistent identifier across deployments.
	// Required.
	ServiceName string

	// Tags provide additional metadata attached to all profiles.
	// Common tags include environment, version, region, etc.
	// Example: map[string]string{"env": "prod", "version": "1.2.3"}
	Tags map[string]string

	// MemProfileRate controls memory profiling detail (allocations recorded).
	// Higher values = less detail but lower overhead.
	// Default: 4096 (approximately 1 in 4096 allocations recorded).
	MemProfileRate int

	// MutexFraction controls mutex profiling frequency.
	// Higher values = less detail but lower overhead.
	// Default: 5 (approximately 1 in 5 mutex events recorded).
	MutexFraction int

	// BlockProfileRate controls block profiling frequency.
	// Higher values = less detail but lower overhead.
	// Default: 100 (approximately 1 in 100 blocking operations recorded).
	BlockProfileRate int

	// Profile type enablement flags. At least one must be true.

	// EnableCPU enables CPU profiling (stack traces and CPU time).
	// Recommended for most applications.
	EnableCPU bool

	// EnableMemory enables memory/heap profiling (allocations and heap size).
	// Recommended for most applications.
	EnableMemory bool

	// EnableGoroutine enables goroutine profiling (count and stack traces).
	// Useful for debugging concurrency issues.
	EnableGoroutine bool

	// EnableMutex enables mutex contention profiling (wait times).
	// Useful for identifying lock contention issues.
	EnableMutex bool

	// EnableBlock enables block profiling (I/O and syscall delays).
	// Useful for identifying blocking I/O issues.
	EnableBlock bool

	// EnableCustom enables custom span collection for user-defined instrumentation.
	// Use with StartSpan() for custom performance tracking.
	EnableCustom bool

	// OutputToStdout enables stdout output mode for testing and debugging.
	// When true, profiles are written to stdout instead of uploaded.
	// APIKey and IngestURL are not required in this mode.
	OutputToStdout bool

	// Env specifies the environment (e.g., "production", "staging", "development").
	// Used for security validation and storage configuration.
	Env string
}

// validate checks the configuration for required fields and sets defaults.
// It returns an error if required fields are missing or invalid.
func (c *Config) validate() error {
	if !c.OutputToStdout {
		if c.APIKey == "" {
			return errors.New("APIKey is required")
		}

		if c.IngestURL == "" {
			return errors.New("IngestURL is required")
		}
	}

	if !c.OutputToStdout && c.Storage == nil {
		return errors.New("Storage is required")
	}

	if c.ServiceName == "" {
		return errors.New("ServiceName is required")
	}

	if c.SampleRate <= 0 {
		c.SampleRate = DefaultSampleRate
	}

	if c.ProfileDuration <= 0 {
		c.ProfileDuration = DefaultProfileDuration
	}

	if c.MemProfileRate <= 0 {
		c.MemProfileRate = DefaultMemProfileRate
	}

	if c.MutexFraction <= 0 {
		c.MutexFraction = DefaultMutexFraction
	}

	if c.BlockProfileRate <= 0 {
		c.BlockProfileRate = DefaultBlockProfileRate
	}

	if !c.EnableCPU && !c.EnableMemory && !c.EnableGoroutine && !c.EnableMutex && !c.EnableBlock && !c.EnableCustom {
		c.EnableCPU = true
		c.EnableMemory = true
	}

	return nil
}

// DefaultConfig creates a production-ready configuration with sensible defaults.
// It enables CPU and memory profiling with standard settings optimized for low overhead.
//
// Parameters:
//   - apiKey: Your Pprofio API key for authentication
//   - ingestURL: The Pprofio API endpoint (typically "https://api.pprofio.com")
//   - serviceName: Identifier for your application
//
// Example:
//
//	cfg := pprofio.DefaultConfig("your-api-key", "https://api.pprofio.com", "my-service")
//	cfg.Tags["env"] = "production"
//	cfg.Tags["version"] = "1.0.0"
func DefaultConfig(apiKey, ingestURL, serviceName string) Config {
	return Config{
		APIKey:           apiKey,
		IngestURL:        ingestURL,
		SampleRate:       DefaultSampleRate,
		ProfileDuration:  DefaultProfileDuration,
		Storage:          &HTTPStorage{URL: ingestURL + "/upload", APIKey: apiKey},
		ServiceName:      serviceName,
		Tags:             make(map[string]string),
		MemProfileRate:   DefaultMemProfileRate,
		MutexFraction:    DefaultMutexFraction,
		BlockProfileRate: DefaultBlockProfileRate,
		EnableCPU:        true,
		EnableMemory:     true,
		EnableGoroutine:  false,
		EnableMutex:      false,
		EnableBlock:      false,
		EnableCustom:     false,
		OutputToStdout:   false,
	}
}
