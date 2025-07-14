package pprofio

import (
	"errors"
	"os"
	"time"
)

const (
	DefaultSampleRate       = 60 * time.Second
	DefaultProfileDuration  = 10 * time.Second
	DefaultMemProfileRate   = 4096
	DefaultMutexFraction    = 5
	DefaultBlockProfileRate = 100
	UnknownHostname         = "unknown"
	RequiredFieldPrefix     = " is required"
	APIKeyRequired          = "APIKey" + RequiredFieldPrefix
	IngestURLRequired       = "IngestURL" + RequiredFieldPrefix
	StorageRequired         = "Storage" + RequiredFieldPrefix
	ServiceNameRequired     = "ServiceName" + RequiredFieldPrefix
	UploadPath              = "/upload"
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
}

func (c *Config) validate() error {
	if err := c.validateRequiredFields(); err != nil {
		return err
	}

	c.setDefaults()
	c.ensureAtLeastOneProfileEnabled()
	c.setHostnameIfEmpty()

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
		APIKey:           apiKey,
		IngestURL:        ingestURL,
		SampleRate:       DefaultSampleRate,
		ProfileDuration:  DefaultProfileDuration,
		Storage:          &HTTPStorage{URL: ingestURL + UploadPath, APIKey: apiKey},
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

// ComprehensiveConfig creates a configuration with all profile types enabled
func ComprehensiveConfig(apiKey, ingestURL, serviceName string) Config {
	return Config{
		APIKey:           apiKey,
		IngestURL:        ingestURL,
		SampleRate:       DefaultSampleRate,
		ProfileDuration:  DefaultProfileDuration,
		Storage:          &HTTPStorage{URL: ingestURL + UploadPath, APIKey: apiKey},
		ServiceName:      serviceName,
		Tags:             make(map[string]string),
		MemProfileRate:   DefaultMemProfileRate,
		MutexFraction:    DefaultMutexFraction,
		BlockProfileRate: DefaultBlockProfileRate,
		EnableCPU:        true,
		EnableMemory:     true,
		EnableGoroutine:  true,
		EnableMutex:      true,
		EnableBlock:      true,
		EnableCustom:     true,
		OutputToStdout:   false,
	}
}
