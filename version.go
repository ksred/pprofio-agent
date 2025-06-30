package pprofio

import (
	"fmt"
	"runtime"
)

// Version information for the pprofio package.
// These values can be overridden at build time using ldflags.
var (
	// Version is the current package version.
	// Can be overridden at build time: go build -ldflags "-X github.com/pprofio/pprofio.Version=1.0.0"
	Version = "0.1.0"

	// Commit is the git commit hash of the build.
	// Can be overridden at build time: go build -ldflags "-X github.com/pprofio/pprofio.Commit=abc123"
	Commit = "dev"

	// BuildDate is the timestamp when the package was built.
	// Can be overridden at build time: go build -ldflags "-X github.com/pprofio/pprofio.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
	BuildDate = "unknown"

	// GoVersion is the Go version used to build the package.
	GoVersion = runtime.Version()
)

// BuildInfo contains comprehensive build and runtime information.
type BuildInfo struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"build_date"`
	GoVersion string `json:"go_version"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
}

// GetBuildInfo returns comprehensive build and runtime information.
// This information can be useful for debugging and support purposes.
//
// Example:
//
//	info := pprofio.GetBuildInfo()
//	log.Printf("pprofio version: %s (commit: %s)", info.Version, info.Commit)
func GetBuildInfo() BuildInfo {
	return BuildInfo{
		Version:   Version,
		Commit:    Commit,
		BuildDate: BuildDate,
		GoVersion: GoVersion,
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	}
}

// String returns a human-readable string representation of the build information.
func (b BuildInfo) String() string {
	return fmt.Sprintf("pprofio %s (commit: %s, built: %s, go: %s, %s/%s)",
		b.Version, b.Commit, b.BuildDate, b.GoVersion, b.OS, b.Arch)
}

// UserAgent returns a user agent string for HTTP requests that includes version information.
// This helps with debugging and analytics on the server side.
func (b BuildInfo) UserAgent() string {
	return fmt.Sprintf("pprofio/%s (%s; %s/%s) go/%s",
		b.Version, b.Commit, b.OS, b.Arch, b.GoVersion)
}