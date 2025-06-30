//go:build debug || dev
// +build debug dev

package pprofio

import (
	"fmt"
	"log"
	"os"
)

// Development build configuration
// This file is included when building with -tags=debug or -tags=dev

const (
	// IsDebugBuild indicates whether this is a debug build
	IsDebugBuild = true

	// IsDevBuild indicates whether this is a development build
	IsDevBuild = true

	// IsProdBuild indicates whether this is a production build
	IsProdBuild = false
)

// debugLog outputs debug information in development builds
func debugLog(format string, args ...interface{}) {
	if os.Getenv("PPROFIO_DEBUG") != "" {
		log.Printf("[PPROFIO DEBUG] "+format, args...)
	}
}

// devModeEnabled returns true in development builds
func devModeEnabled() bool {
	return true
}

// validateDevConfig performs additional validation in development builds
func validateDevConfig(cfg *Config) error {
	if cfg.ServiceName == "" {
		return fmt.Errorf("ServiceName is required")
	}

	// Additional development-time checks
	if cfg.SampleRate < DefaultSampleRate/10 {
		debugLog("Warning: Very short SampleRate (%v) may cause high overhead", cfg.SampleRate)
	}

	if cfg.ProfileDuration > DefaultProfileDuration*3 {
		debugLog("Warning: Long ProfileDuration (%v) may cause high overhead", cfg.ProfileDuration)
	}

	return nil
}