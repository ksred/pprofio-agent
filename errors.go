package pprofio

import (
	"errors"
	"fmt"
)

// Common errors returned by the pprofio package.
// These provide structured error handling for common failure scenarios.
var (
	// ErrAlreadyStarted is returned when Start() is called on an already running profiler.
	ErrAlreadyStarted = errors.New("profiler is already started")

	// ErrNotStarted is returned when operations are attempted on a non-running profiler.
	ErrNotStarted = errors.New("profiler is not started")

	// ErrInvalidConfig is returned when required configuration fields are missing or invalid.
	ErrInvalidConfig = errors.New("invalid configuration")

	// ErrUploadFailed is returned when profile upload fails after all retry attempts.
	ErrUploadFailed = errors.New("profile upload failed")

	// ErrAuthenticationFailed is returned when API authentication fails.
	ErrAuthenticationFailed = errors.New("authentication failed")

	// ErrUnsecureConnection is returned when HTTPS is required but not used.
	ErrUnsecureConnection = errors.New("HTTPS is required for secure uploads")
)

// ConfigError represents a configuration validation error with details about
// which field caused the issue.
type ConfigError struct {
	Field   string // The configuration field that caused the error
	Message string // Human-readable description of the problem
}

// Error implements the error interface for ConfigError.
func (e *ConfigError) Error() string {
	return fmt.Sprintf("configuration error in field '%s': %s", e.Field, e.Message)
}

// Is allows ConfigError to be checked with errors.Is().
func (e *ConfigError) Is(target error) bool {
	return target == ErrInvalidConfig
}

// UploadError represents an upload failure with retry information and status codes.
type UploadError struct {
	URL        string // The upload URL that failed
	StatusCode int    // HTTP status code (if applicable)
	Attempts   int    // Number of retry attempts made
	Underlying error  // The underlying error that caused the failure
}

// Error implements the error interface for UploadError.
func (e *UploadError) Error() string {
	if e.StatusCode > 0 {
		return fmt.Sprintf("upload failed to %s (status %d) after %d attempts: %v",
			e.URL, e.StatusCode, e.Attempts, e.Underlying)
	}
	return fmt.Sprintf("upload failed to %s after %d attempts: %v",
		e.URL, e.Attempts, e.Underlying)
}

// Is allows UploadError to be checked with errors.Is().
func (e *UploadError) Is(target error) bool {
	return target == ErrUploadFailed
}

// Unwrap allows UploadError to be unwrapped to access the underlying error.
func (e *UploadError) Unwrap() error {
	return e.Underlying
}

// StorageError represents a storage backend error with context about the operation.
type StorageError struct {
	Operation string // The operation that failed (e.g., "compress", "write", "upload")
	Backend   string // The storage backend type (e.g., "http", "file", "stdout")
	Underlying error // The underlying error that caused the failure
}

// Error implements the error interface for StorageError.
func (e *StorageError) Error() string {
	return fmt.Sprintf("storage error during %s operation with %s backend: %v",
		e.Operation, e.Backend, e.Underlying)
}

// Unwrap allows StorageError to be unwrapped to access the underlying error.
func (e *StorageError) Unwrap() error {
	return e.Underlying
}

// ProfileError represents an error during profile collection with details
// about which profile type failed.
type ProfileError struct {
	Type       string // The profile type that failed (e.g., "cpu", "memory", "goroutine")
	Operation  string // The operation that failed (e.g., "collect", "write", "upload")
	Underlying error  // The underlying error that caused the failure
}

// Error implements the error interface for ProfileError.
func (e *ProfileError) Error() string {
	return fmt.Sprintf("profile error during %s operation for %s profile: %v",
		e.Operation, e.Type, e.Underlying)
}

// Unwrap allows ProfileError to be unwrapped to access the underlying error.
func (e *ProfileError) Unwrap() error {
	return e.Underlying
}