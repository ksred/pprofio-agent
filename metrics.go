package pprofio

import (
	"sync/atomic"
	"time"
)

// Metrics contains internal performance and health metrics for the profiler.
// These metrics can be used for monitoring profiler health and debugging issues.
type Metrics struct {
	// Profile collection metrics
	ProfilesCollected   int64 `json:"profiles_collected"`
	ProfilesUploaded    int64 `json:"profiles_uploaded"`
	ProfilesFailed      int64 `json:"profiles_failed"`
	UploadRetries       int64 `json:"upload_retries"`
	
	// Timing metrics
	LastCollectionTime  int64 `json:"last_collection_time"`  // Unix timestamp
	TotalCollectionTime int64 `json:"total_collection_time"` // Milliseconds
	AverageUploadTime   int64 `json:"average_upload_time"`   // Milliseconds
	
	// Error metrics
	ConfigErrors        int64 `json:"config_errors"`
	NetworkErrors       int64 `json:"network_errors"`
	StorageErrors       int64 `json:"storage_errors"`
	
	// Custom span metrics
	SpansCreated        int64 `json:"spans_created"`
	SpansProcessed      int64 `json:"spans_processed"`
	SpansDropped        int64 `json:"spans_dropped"`
	
	// Resource usage
	GoroutinesRunning   int64 `json:"goroutines_running"`
	MemoryAllocated     int64 `json:"memory_allocated"` // Bytes
}

// GetMetrics returns a snapshot of current profiler metrics.
// This method is safe for concurrent use and provides insights into
// profiler performance and health.
//
// Example:
//
//	metrics := profiler.GetMetrics()
//	log.Printf("Profiles collected: %d, uploaded: %d, failed: %d",
//		metrics.ProfilesCollected, metrics.ProfilesUploaded, metrics.ProfilesFailed)
func (p *Profiler) GetMetrics() Metrics {
	return Metrics{
		ProfilesCollected:   atomic.LoadInt64(&p.metrics.ProfilesCollected),
		ProfilesUploaded:    atomic.LoadInt64(&p.metrics.ProfilesUploaded),
		ProfilesFailed:      atomic.LoadInt64(&p.metrics.ProfilesFailed),
		UploadRetries:       atomic.LoadInt64(&p.metrics.UploadRetries),
		LastCollectionTime:  atomic.LoadInt64(&p.metrics.LastCollectionTime),
		TotalCollectionTime: atomic.LoadInt64(&p.metrics.TotalCollectionTime),
		AverageUploadTime:   atomic.LoadInt64(&p.metrics.AverageUploadTime),
		ConfigErrors:        atomic.LoadInt64(&p.metrics.ConfigErrors),
		NetworkErrors:       atomic.LoadInt64(&p.metrics.NetworkErrors),
		StorageErrors:       atomic.LoadInt64(&p.metrics.StorageErrors),
		SpansCreated:        atomic.LoadInt64(&p.metrics.SpansCreated),
		SpansProcessed:      atomic.LoadInt64(&p.metrics.SpansProcessed),
		SpansDropped:        atomic.LoadInt64(&p.metrics.SpansDropped),
		GoroutinesRunning:   atomic.LoadInt64(&p.metrics.GoroutinesRunning),
		MemoryAllocated:     atomic.LoadInt64(&p.metrics.MemoryAllocated),
	}
}

// ResetMetrics resets all metrics counters to zero.
// This method is useful for testing or periodic metric collection.
func (p *Profiler) ResetMetrics() {
	atomic.StoreInt64(&p.metrics.ProfilesCollected, 0)
	atomic.StoreInt64(&p.metrics.ProfilesUploaded, 0)
	atomic.StoreInt64(&p.metrics.ProfilesFailed, 0)
	atomic.StoreInt64(&p.metrics.UploadRetries, 0)
	atomic.StoreInt64(&p.metrics.LastCollectionTime, 0)
	atomic.StoreInt64(&p.metrics.TotalCollectionTime, 0)
	atomic.StoreInt64(&p.metrics.AverageUploadTime, 0)
	atomic.StoreInt64(&p.metrics.ConfigErrors, 0)
	atomic.StoreInt64(&p.metrics.NetworkErrors, 0)
	atomic.StoreInt64(&p.metrics.StorageErrors, 0)
	atomic.StoreInt64(&p.metrics.SpansCreated, 0)
	atomic.StoreInt64(&p.metrics.SpansProcessed, 0)
	atomic.StoreInt64(&p.metrics.SpansDropped, 0)
	atomic.StoreInt64(&p.metrics.GoroutinesRunning, 0)
	atomic.StoreInt64(&p.metrics.MemoryAllocated, 0)
}

// HealthStatus represents the overall health of the profiler
type HealthStatus string

const (
	// HealthStatusHealthy indicates the profiler is operating normally
	HealthStatusHealthy HealthStatus = "healthy"
	
	// HealthStatusDegraded indicates some issues but profiler is still functional
	HealthStatusDegraded HealthStatus = "degraded"
	
	// HealthStatusUnhealthy indicates significant issues affecting profiler operation
	HealthStatusUnhealthy HealthStatus = "unhealthy"
)

// HealthCheck returns the current health status of the profiler based on metrics.
// This provides a quick way to assess profiler health for monitoring systems.
//
// Health criteria:
//   - Healthy: Recent successful uploads, low error rate
//   - Degraded: Some failures but still functional
//   - Unhealthy: High error rate or no recent successful uploads
func (p *Profiler) HealthCheck() HealthStatus {
	metrics := p.GetMetrics()
	
	// Check if profiler is running
	if !p.initialized {
		return HealthStatusUnhealthy
	}
	
	// Calculate error rates
	totalAttempts := metrics.ProfilesUploaded + metrics.ProfilesFailed
	if totalAttempts == 0 {
		// No attempts yet, consider healthy if recently started
		return HealthStatusHealthy
	}
	
	errorRate := float64(metrics.ProfilesFailed) / float64(totalAttempts)
	
	// Health thresholds
	switch {
	case errorRate > 0.5: // More than 50% failure rate
		return HealthStatusUnhealthy
	case errorRate > 0.2: // More than 20% failure rate
		return HealthStatusDegraded
	default:
		return HealthStatusHealthy
	}
}

// Internal metric tracking methods
func (p *Profiler) incrementProfilesCollected() {
	atomic.AddInt64(&p.metrics.ProfilesCollected, 1)
	atomic.StoreInt64(&p.metrics.LastCollectionTime, time.Now().Unix())
}

func (p *Profiler) incrementProfilesUploaded() {
	atomic.AddInt64(&p.metrics.ProfilesUploaded, 1)
}

func (p *Profiler) incrementProfilesFailed() {
	atomic.AddInt64(&p.metrics.ProfilesFailed, 1)
}

func (p *Profiler) incrementUploadRetries() {
	atomic.AddInt64(&p.metrics.UploadRetries, 1)
}

func (p *Profiler) recordUploadTime(duration time.Duration) {
	ms := duration.Milliseconds()
	atomic.AddInt64(&p.metrics.TotalCollectionTime, ms)
	
	// Calculate simple moving average
	uploaded := atomic.LoadInt64(&p.metrics.ProfilesUploaded)
	if uploaded > 0 {
		total := atomic.LoadInt64(&p.metrics.TotalCollectionTime)
		avg := total / uploaded
		atomic.StoreInt64(&p.metrics.AverageUploadTime, avg)
	}
}

func (p *Profiler) incrementSpansCreated() {
	atomic.AddInt64(&p.metrics.SpansCreated, 1)
}

func (p *Profiler) incrementSpansProcessed() {
	atomic.AddInt64(&p.metrics.SpansProcessed, 1)
}

func (p *Profiler) incrementSpansDropped() {
	atomic.AddInt64(&p.metrics.SpansDropped, 1)
}