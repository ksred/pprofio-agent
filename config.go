package pprofio

import (
	"errors"
	"time"
)

const (
	DefaultSampleRate       = 60 * time.Second
	DefaultProfileDuration  = 10 * time.Second
	DefaultMemProfileRate   = 4096
	DefaultMutexFraction    = 5
	DefaultBlockProfileRate = 100
)

type Config struct {
	APIKey          string
	IngestURL       string
	SampleRate      time.Duration
	ProfileDuration time.Duration
	Storage         Storage
	ServiceName     string
	Tags            map[string]string
	MemProfileRate  int
	MutexFraction   int
	BlockProfileRate int
	EnableCPU       bool
	EnableMemory    bool
	EnableGoroutine bool
	EnableMutex     bool
	EnableBlock     bool
	EnableCustom    bool
}

func (c *Config) validate() error {
	if c.APIKey == "" {
		return errors.New("APIKey is required")
	}

	if c.IngestURL == "" {
		return errors.New("IngestURL is required")
	}

	if c.Storage == nil {
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

func DefaultConfig(apiKey, ingestURL, serviceName string) Config {
	return Config{
		APIKey:          apiKey,
		IngestURL:       ingestURL,
		SampleRate:      DefaultSampleRate,
		ProfileDuration: DefaultProfileDuration,
		Storage:         &HTTPStorage{URL: ingestURL + "/upload", APIKey: apiKey},
		ServiceName:     serviceName,
		Tags:            make(map[string]string),
		MemProfileRate:  DefaultMemProfileRate,
		MutexFraction:   DefaultMutexFraction,
		BlockProfileRate: DefaultBlockProfileRate,
		EnableCPU:       true,
		EnableMemory:    true,
		EnableGoroutine: false,
		EnableMutex:     false,
		EnableBlock:     false,
		EnableCustom:    false,
	}
}