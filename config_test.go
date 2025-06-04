package pprofio

import (
	"testing"
)

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "Valid config",
			config: Config{
				APIKey:      "test-key",
				IngestURL:   "https://api.pprofio.com",
				Storage:     &HTTPStorage{URL: "https://api.pprofio.com/upload", APIKey: "test-key"},
				ServiceName: "test-service",
			},
			wantErr: false,
		},
		{
			name: "Missing APIKey",
			config: Config{
				IngestURL:   "https://api.pprofio.com",
				Storage:     &HTTPStorage{URL: "https://api.pprofio.com/upload", APIKey: "test-key"},
				ServiceName: "test-service",
			},
			wantErr: true,
		},
		{
			name: "Missing IngestURL",
			config: Config{
				APIKey:      "test-key",
				Storage:     &HTTPStorage{URL: "https://api.pprofio.com/upload", APIKey: "test-key"},
				ServiceName: "test-service",
			},
			wantErr: true,
		},
		{
			name: "Missing Storage",
			config: Config{
				APIKey:      "test-key",
				IngestURL:   "https://api.pprofio.com",
				ServiceName: "test-service",
			},
			wantErr: true,
		},
		{
			name: "Missing ServiceName",
			config: Config{
				APIKey:    "test-key",
				IngestURL: "https://api.pprofio.com",
				Storage:   &HTTPStorage{URL: "https://api.pprofio.com/upload", APIKey: "test-key"},
			},
			wantErr: true,
		},
		{
			name: "Default SampleRate",
			config: Config{
				APIKey:      "test-key",
				IngestURL:   "https://api.pprofio.com",
				Storage:     &HTTPStorage{URL: "https://api.pprofio.com/upload", APIKey: "test-key"},
				ServiceName: "test-service",
				SampleRate:  0, // Will be set to default
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.validate() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Check defaults are applied
			if !tt.wantErr && tt.config.SampleRate <= 0 {
				if tt.config.SampleRate != DefaultSampleRate {
					t.Errorf("Default SampleRate not applied, got %v, want %v", tt.config.SampleRate, DefaultSampleRate)
				}
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	apiKey := "test-key"
	ingestURL := "https://api.pprofio.com"
	serviceName := "test-service"

	cfg := DefaultConfig(apiKey, ingestURL, serviceName)

	if cfg.APIKey != apiKey {
		t.Errorf("APIKey not set correctly, got %v, want %v", cfg.APIKey, apiKey)
	}

	if cfg.IngestURL != ingestURL {
		t.Errorf("IngestURL not set correctly, got %v, want %v", cfg.IngestURL, ingestURL)
	}

	if cfg.ServiceName != serviceName {
		t.Errorf("ServiceName not set correctly, got %v, want %v", cfg.ServiceName, serviceName)
	}

	if cfg.SampleRate != DefaultSampleRate {
		t.Errorf("SampleRate not set correctly, got %v, want %v", cfg.SampleRate, DefaultSampleRate)
	}

	if cfg.ProfileDuration != DefaultProfileDuration {
		t.Errorf("ProfileDuration not set correctly, got %v, want %v", cfg.ProfileDuration, DefaultProfileDuration)
	}

	if cfg.MemProfileRate != DefaultMemProfileRate {
		t.Errorf("MemProfileRate not set correctly, got %v, want %v", cfg.MemProfileRate, DefaultMemProfileRate)
	}

	if cfg.MutexFraction != DefaultMutexFraction {
		t.Errorf("MutexFraction not set correctly, got %v, want %v", cfg.MutexFraction, DefaultMutexFraction)
	}

	if cfg.BlockProfileRate != DefaultBlockProfileRate {
		t.Errorf("BlockProfileRate not set correctly, got %v, want %v", cfg.BlockProfileRate, DefaultBlockProfileRate)
	}

	if !cfg.EnableCPU {
		t.Error("EnableCPU should be true by default")
	}

	if !cfg.EnableMemory {
		t.Error("EnableMemory should be true by default")
	}

	if cfg.EnableGoroutine {
		t.Error("EnableGoroutine should be false by default")
	}

	if cfg.EnableMutex {
		t.Error("EnableMutex should be false by default")
	}

	if cfg.EnableBlock {
		t.Error("EnableBlock should be false by default")
	}

	if cfg.EnableCustom {
		t.Error("EnableCustom should be false by default")
	}
}