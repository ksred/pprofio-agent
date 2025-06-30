//go:build !debug && !dev
// +build !debug,!dev

package pprofio

// Production build configuration
// This file is included in production builds where debug and dev tags are not set.

const (
	// IsDebugBuild indicates whether this is a debug build
	IsDebugBuild = false

	// IsDevBuild indicates whether this is a development build
	IsDevBuild = false

	// IsProdBuild indicates whether this is a production build
	IsProdBuild = true
)

// debugLog is a no-op in production builds for performance
func debugLog(format string, args ...interface{}) {
	// No-op in production
}

// devModeEnabled returns false in production builds
func devModeEnabled() bool {
	return false
}