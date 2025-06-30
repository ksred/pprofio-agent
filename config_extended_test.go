package pprofio

import (
	"testing"
	"time"
)

func TestDefaultConfig_Complete(t *testing.T) {
	apiKey := "test-api-key"
	ingestURL := "https://api.pprofio.com"
	serviceName := "test-service"
	
	config := DefaultConfig(apiKey, ingestURL, serviceName)
	
	// Verify all fields are set correctly
	if config.APIKey != apiKey {
		t.Errorf("Expected APIKey %q, got %q", apiKey, config.APIKey)
	}
	
	if config.IngestURL != ingestURL {
		t.Errorf("Expected IngestURL %q, got %q", ingestURL, config.IngestURL)
	}
	
	if config.ServiceName != serviceName {
		t.Errorf("Expected ServiceName %q, got %q", serviceName, config.ServiceName)
	}
	
	// Verify defaults
	if config.SampleRate != DefaultSampleRate {
		t.Errorf("Expected SampleRate %v, got %v", DefaultSampleRate, config.SampleRate)
	}
	
	if config.ProfileDuration != DefaultProfileDuration {
		t.Errorf("Expected ProfileDuration %v, got %v", DefaultProfileDuration, config.ProfileDuration)
	}
	
	if config.MemProfileRate != DefaultMemProfileRate {
		t.Errorf("Expected MemProfileRate %d, got %d", DefaultMemProfileRate, config.MemProfileRate)
	}
	
	if config.MutexFraction != DefaultMutexFraction {
		t.Errorf("Expected MutexFraction %d, got %d", DefaultMutexFraction, config.MutexFraction)
	}
	
	if config.BlockProfileRate != DefaultBlockProfileRate {
		t.Errorf("Expected BlockProfileRate %d, got %d", DefaultBlockProfileRate, config.BlockProfileRate)
	}
	
	// Verify enabled profiles
	if !config.EnableCPU {
		t.Error("Expected EnableCPU to be true by default")
	}
	
	if !config.EnableMemory {
		t.Error("Expected EnableMemory to be true by default")
	}
	
	if config.EnableGoroutine {
		t.Error("Expected EnableGoroutine to be false by default")
	}
	
	if config.EnableMutex {
		t.Error("Expected EnableMutex to be false by default")
	}
	
	if config.EnableBlock {
		t.Error("Expected EnableBlock to be false by default")
	}
	
	if config.EnableCustom {
		t.Error("Expected EnableCustom to be false by default")
	}
	
	if config.OutputToStdout {
		t.Error("Expected OutputToStdout to be false by default")
	}
	
	// Verify storage is set up
	if config.Storage == nil {
		t.Error("Expected Storage to be configured by default")
	}
	
	// Verify it's an HTTPStorage
	if httpStorage, ok := config.Storage.(*HTTPStorage); ok {
		expectedURL := ingestURL + "/upload"
		if httpStorage.URL != expectedURL {
			t.Errorf("Expected storage URL %q, got %q", expectedURL, httpStorage.URL)
		}
		
		if httpStorage.APIKey != apiKey {
			t.Errorf("Expected storage APIKey %q, got %q", apiKey, httpStorage.APIKey)
		}
	} else {
		t.Error("Expected default storage to be HTTPStorage")
	}
	
	// Verify tags map is initialized
	if config.Tags == nil {
		t.Error("Expected Tags map to be initialized")
	}
}

func TestConfig_validate_EdgeCases(t *testing.T) {
	t.Run("AllRuntimeSettingsZero", func(t *testing.T) {
		config := &Config{
			APIKey:           "test-key",
			IngestURL:        "https://api.pprofio.com",
			ServiceName:      "test-service",
			SampleRate:       0,
			ProfileDuration:  0,
			MemProfileRate:   0,
			MutexFraction:    0,
			BlockProfileRate: 0,
			OutputToStdout:   false,
		}
		
		err := config.validate()
		if err != nil {
			t.Fatalf("validate() error = %v", err)
		}
		
		// Verify defaults are applied
		if config.SampleRate != DefaultSampleRate {
			t.Errorf("Expected default SampleRate %v, got %v", DefaultSampleRate, config.SampleRate)
		}
		
		if config.ProfileDuration != DefaultProfileDuration {
			t.Errorf("Expected default ProfileDuration %v, got %v", DefaultProfileDuration, config.ProfileDuration)
		}
		
		if config.MemProfileRate != DefaultMemProfileRate {
			t.Errorf("Expected default MemProfileRate %d, got %d", DefaultMemProfileRate, config.MemProfileRate)
		}
		
		if config.MutexFraction != DefaultMutexFraction {
			t.Errorf("Expected default MutexFraction %d, got %d", DefaultMutexFraction, config.MutexFraction)
		}
		
		if config.BlockProfileRate != DefaultBlockProfileRate {
			t.Errorf("Expected default BlockProfileRate %d, got %d", DefaultBlockProfileRate, config.BlockProfileRate)
		}
	})
	
	t.Run("NoProfileTypesEnabled", func(t *testing.T) {
		config := &Config{
			APIKey:          "test-key",
			IngestURL:       "https://api.pprofio.com",
			ServiceName:     "test-service",
			SampleRate:      60 * time.Second,
			ProfileDuration: 10 * time.Second,
			// All profile types disabled by default
			EnableCPU:       false,
			EnableMemory:    false,
			EnableGoroutine: false,
			EnableMutex:     false,
			EnableBlock:     false,
			EnableCustom:    false,
			OutputToStdout:  false,
		}
		
		err := config.validate()
		if err != nil {
			t.Fatalf("validate() error = %v", err)
		}
		
		// Should enable CPU and Memory by default
		if !config.EnableCPU {
			t.Error("Expected EnableCPU to be enabled by default when nothing is enabled")
		}
		
		if !config.EnableMemory {
			t.Error("Expected EnableMemory to be enabled by default when nothing is enabled")
		}
	})
	
	t.Run("StdoutModeSkipsValidation", func(t *testing.T) {
		config := &Config{
			ServiceName:    "test-service",
			OutputToStdout: true,
			// Missing APIKey and IngestURL, but should be OK in stdout mode
		}
		
		err := config.validate()
		if err != nil {
			t.Errorf("validate() with OutputToStdout should not require APIKey/IngestURL: %v", err)
		}
	})
}

func TestConstants(t *testing.T) {
	// Test that constants have reasonable values
	if DefaultSampleRate != 60*time.Second {
		t.Errorf("Expected DefaultSampleRate to be 60s, got %v", DefaultSampleRate)
	}
	
	if DefaultProfileDuration != 10*time.Second {
		t.Errorf("Expected DefaultProfileDuration to be 10s, got %v", DefaultProfileDuration)
	}
	
	if DefaultMemProfileRate != 4096 {
		t.Errorf("Expected DefaultMemProfileRate to be 4096, got %d", DefaultMemProfileRate)
	}
	
	if DefaultMutexFraction != 5 {
		t.Errorf("Expected DefaultMutexFraction to be 5, got %d", DefaultMutexFraction)
	}
	
	if DefaultBlockProfileRate != 100 {
		t.Errorf("Expected DefaultBlockProfileRate to be 100, got %d", DefaultBlockProfileRate)
	}
}