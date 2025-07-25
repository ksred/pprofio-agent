package pprofio

import (
	"errors"
	"os"
	"time"
)

const (
	DefaultIngestURL                = "https://api.pprofio.com/api/v1"
	DefaultSampleRate               = 60 * time.Second
	DefaultProfileDuration          = 10 * time.Second
	DefaultMemProfileRate           = 4096
	DefaultMutexFraction            = 5
	DefaultBlockProfileRate         = 100
	DefaultHTTPMetricsSampleRate    = 1.0
	DefaultHTTPMetricsBatchSize     = 50
	DefaultHTTPMetricsFlushInterval = 30 * time.Second
	UnknownHostname                 = "unknown"
	RequiredFieldPrefix             = " is required"
	APIKeyRequired                  = "APIKey" + RequiredFieldPrefix
	IngestURLRequired               = "IngestURL" + RequiredFieldPrefix
	StorageRequired                 = "Storage" + RequiredFieldPrefix
	ServiceNameRequired             = "ServiceName" + RequiredFieldPrefix
	UploadPath                      = "/upload"
)

type Config struct {
	APIKey           string
	IngestURL        string
	SampleRate       time.Duration
	ProfileDuration  time.Duration
	Storage          Storage
	ServiceName      string
	Tags             map[string]string
	MemProfileRate   int
	MutexFraction    int
	BlockProfileRate int
	EnableCPU        bool
	EnableMemory     bool
	EnableGoroutine  bool
	EnableMutex      bool
	EnableBlock      bool
	EnableCustom     bool
	OutputToStdout   bool
	Env              string
	Hostname         string

	// HTTP Middleware Configuration
	EnableHTTPMetrics           bool          `json:"enable_http_metrics"`
	HTTPMetricsSampleRate       float64       `json:"http_metrics_sample_rate"`      // 0.0 to 1.0
	HTTPMetricsExcludePaths     []string      `json:"http_metrics_exclude_paths"`    // Paths to exclude from monitoring
	HTTPMetricsIncludeHeaders   []string      `json:"http_metrics_include_headers"`  // Headers to capture
	HTTPMetricsMaxPayloadSize   int64         `json:"http_metrics_max_payload_size"` // Maximum payload size to capture
	HTTPMetricsCollectUserAgent bool          `json:"http_metrics_collect_user_agent"`
	HTTPMetricsHashIPs          bool          `json:"http_metrics_hash_ips"`       // Whether to hash IP addresses
	HTTPMetricsBatchSize        int           `json:"http_metrics_batch_size"`     // Batch size for sending metrics
	HTTPMetricsFlushInterval    time.Duration `json:"http_metrics_flush_interval"` // How often to flush batched metrics
}

func (c *Config) validate() error {
	// Apply defaults first before validation
	c.setDefaults()
	c.ensureAtLeastOneProfileEnabled()
	c.setHostnameIfEmpty()

	// Then validate required fields
	if err := c.validateRequiredFields(); err != nil {
		return err
	}

	return nil
}

func (c *Config) validateRequiredFields() error {
	if !c.OutputToStdout {
		if c.APIKey == "" {
			return errors.New(APIKeyRequired)
		}

		if c.IngestURL == "" {
			return errors.New(IngestURLRequired)
		}

		if c.Storage == nil {
			return errors.New(StorageRequired)
		}
	}

	if c.ServiceName == "" {
		return errors.New(ServiceNameRequired)
	}

	return nil
}

func (c *Config) setDefaults() {
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

	// Set HTTP metrics defaults
	if c.HTTPMetricsSampleRate <= 0 {
		c.HTTPMetricsSampleRate = DefaultHTTPMetricsSampleRate
	}

	if c.HTTPMetricsBatchSize <= 0 {
		c.HTTPMetricsBatchSize = DefaultHTTPMetricsBatchSize
	}

	if c.HTTPMetricsFlushInterval <= 0 {
		c.HTTPMetricsFlushInterval = DefaultHTTPMetricsFlushInterval
	}

	// Set default excluded paths if none provided
	if c.EnableHTTPMetrics && len(c.HTTPMetricsExcludePaths) == 0 {
		c.HTTPMetricsExcludePaths = []string{"/health", "/metrics", "/ping"}
	}

	// Set default URL
	if c.IngestURL == "" {
		c.IngestURL = DefaultIngestURL
	}
}

func (c *Config) ensureAtLeastOneProfileEnabled() {
	if !c.EnableCPU && !c.EnableMemory && !c.EnableGoroutine && !c.EnableMutex && !c.EnableBlock && !c.EnableCustom {
		c.EnableCPU = true
		c.EnableMemory = true
	}
}

func (c *Config) setHostnameIfEmpty() {
	if c.Hostname == "" {
		hostname, err := os.Hostname()
		if err != nil {
			// If we can't get the hostname, use "unknown"
			c.Hostname = UnknownHostname
		} else {
			c.Hostname = hostname
		}
	}
}

func DefaultConfig(apiKey, ingestURL, serviceName string) Config {
	return Config{
		APIKey:            apiKey,
		IngestURL:         ingestURL,
		SampleRate:        DefaultSampleRate,
		ProfileDuration:   DefaultProfileDuration,
		Storage:           &HTTPStorage{URL: ingestURL + UploadPath, APIKey: apiKey, Env: "production"},
		ServiceName:       serviceName,
		Tags:              make(map[string]string),
		MemProfileRate:    DefaultMemProfileRate,
		MutexFraction:     DefaultMutexFraction,
		BlockProfileRate:  DefaultBlockProfileRate,
		EnableCPU:         true,
		EnableMemory:      true,
		EnableGoroutine:   false,
		EnableMutex:       false,
		EnableBlock:       false,
		EnableCustom:      false,
		OutputToStdout:    false,
		EnableHTTPMetrics: false,
	}
}

// ComprehensiveConfig creates a configuration with all profile types enabled
func ComprehensiveConfig(apiKey, ingestURL, serviceName string) Config {
	return Config{
		APIKey:            apiKey,
		IngestURL:         ingestURL,
		SampleRate:        DefaultSampleRate,
		ProfileDuration:   DefaultProfileDuration,
		Storage:           &HTTPStorage{URL: ingestURL + UploadPath, APIKey: apiKey, Env: "production"},
		ServiceName:       serviceName,
		Tags:              make(map[string]string),
		MemProfileRate:    DefaultMemProfileRate,
		MutexFraction:     DefaultMutexFraction,
		BlockProfileRate:  DefaultBlockProfileRate,
		EnableCPU:         true,
		EnableMemory:      true,
		EnableGoroutine:   true,
		EnableMutex:       true,
		EnableBlock:       true,
		EnableCustom:      true,
		EnableHTTPMetrics: true,
		OutputToStdout:    false,
	}
}
