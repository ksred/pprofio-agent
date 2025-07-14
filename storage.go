package pprofio

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	DefaultHTTPTimeout            = 30 * time.Second
	DefaultRetries                = 3
	DefaultFileMode               = 0o755
	BackoffMultiplier             = 2
	BackoffBase                   = 100
	HTTPStatusOK                  = 200
	HTTPStatusMultipleChoices     = 300
	HTTPStatusUnauthorized        = 401
	HTTPStatusForbidden           = 403
	HTTPStatusTooManyRequests     = 429
	HTTPStatusInternalServerError = 500
	HTTPStatusServiceUnavailable  = 600
	ProfileTypeUnknown            = "unknown"
	ProfileTypeCPU                = "cpu"
	ProfileTypeMemory             = "memory"
	ProfileTypeHeap               = "heap"
	ProfileTypeGoroutine          = "goroutine"
	ProfileTypeMutex              = "mutex"
	ProfileTypeBlock              = "block"
	ProfileTypeCustom             = "custom"
	ContentTypeOctetStream        = "application/octet-stream"
	ContentEncoding               = "gzip"
	AuthorizationPrefix           = "Bearer "
	UserAgentHeader               = "User-Agent"
	StdoutProfileID               = "stdout-profile"
	StdoutProfileURL              = "stdout"
	LocalEnv                      = "local"
	HTTPSScheme                   = "https"
	UUIDLength                    = 32
	UUIDBytesLength               = 16
	UUIDVersionMask               = 0x0f
	UUIDVersionValue              = 0x40
	UUIDVariantMask               = 0x3f
	UUIDVariantValue              = 0x80
)

type Storage interface {
	Upload(ctx context.Context, filePath string) (string, error)
}

type HTTPStorage struct {
	URL     string
	APIKey  string
	Client  *http.Client
	Retries int
	Env     string
}

func NewHTTPStorage(url, apiKey, env string) *HTTPStorage {
	return &HTTPStorage{
		URL:     url,
		APIKey:  apiKey,
		Client:  &http.Client{Timeout: DefaultHTTPTimeout},
		Retries: DefaultRetries,
		Env:     env,
	}
}

func (s *HTTPStorage) Upload(ctx context.Context, filePath string) (string, error) {
	if s.URL == "" || s.APIKey == "" {
		return "", errors.New("URL and APIKey are required")
	}

	// Validate URL format and ensure HTTPS
	parsedURL, err := url.Parse(s.URL)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	if parsedURL.Scheme != HTTPSScheme && s.Env != LocalEnv {
		return "", errors.New("HTTPS is required for secure uploads")
	}

	// Open and compress the file
	data, err := s.readAndCompressFile(filePath)
	if err != nil {
		return "", err
	}

	// Upload with retries
	return s.uploadWithRetries(ctx, data)
}

func (s *HTTPStorage) readAndCompressFile(filePath string) ([]byte, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Read file data
	fileData, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Compress with gzip
	var buf bytes.Buffer
	gzipWriter := gzip.NewWriter(&buf)

	_, err = gzipWriter.Write(fileData)
	if err != nil {
		return nil, fmt.Errorf("failed to compress data: %w", err)
	}

	if err := gzipWriter.Close(); err != nil {
		return nil, fmt.Errorf("failed to finalize compression: %w", err)
	}

	return buf.Bytes(), nil
}

func (s *HTTPStorage) uploadWithRetries(ctx context.Context, data []byte) (string, error) {
	var lastErr error

	for attempt := 0; attempt < s.Retries; attempt++ {
		// Exponential backoff
		if attempt > 0 {
			backoff := math.Pow(BackoffMultiplier, float64(attempt-1)) * BackoffBase
			time.Sleep(time.Duration(backoff) * time.Millisecond)
		}

		// Create the request
		req, err := http.NewRequestWithContext(ctx, "POST", s.URL, bytes.NewReader(data))
		if err != nil {
			lastErr = fmt.Errorf("failed to create request: %w", err)
			continue
		}

		req.Header.Set("Content-Type", ContentTypeOctetStream)
		req.Header.Set("Content-Encoding", ContentEncoding)
		req.Header.Set("Authorization", AuthorizationPrefix+s.APIKey)

		// Send the request
		resp, err := s.Client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			continue
		}

		defer resp.Body.Close()

		// Handle HTTP errors
		if resp.StatusCode == HTTPStatusUnauthorized || resp.StatusCode == HTTPStatusForbidden {
			return "", fmt.Errorf("authentication failed: %d", resp.StatusCode)
		}

		if shouldRetry(resp.StatusCode) {
			lastErr = fmt.Errorf("server error: %d", resp.StatusCode)
			continue
		}

		if resp.StatusCode < HTTPStatusOK || resp.StatusCode >= HTTPStatusMultipleChoices {
			return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		// Read response
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = fmt.Errorf("failed to read response: %w", err)
			continue
		}

		return string(body), nil
	}

	return "", fmt.Errorf("upload failed after %d attempts: %w", s.Retries, lastErr)
}

// extractProfileType extracts the profile type from the filename
func extractProfileType(filename string) string {
	// Remove extension
	name := strings.TrimSuffix(filename, filepath.Ext(filename))

	// Common profile type patterns
	switch {
	case strings.Contains(name, ProfileTypeCPU):
		return ProfileTypeCPU
	case strings.Contains(name, ProfileTypeMemory), strings.Contains(name, ProfileTypeHeap):
		return ProfileTypeMemory
	case strings.Contains(name, ProfileTypeGoroutine):
		return ProfileTypeGoroutine
	case strings.Contains(name, ProfileTypeMutex):
		return ProfileTypeMutex
	case strings.Contains(name, ProfileTypeBlock):
		return ProfileTypeBlock
	case strings.Contains(name, ProfileTypeCustom):
		return ProfileTypeCustom
	default:
		return ProfileTypeUnknown
	}
}

type FileStorage struct {
	Directory string
}

func NewFileStorage(directory string) (*FileStorage, error) {
	if directory == "" {
		return nil, errors.New("directory is required")
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(directory, DefaultFileMode); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	return &FileStorage{Directory: directory}, nil
}

func (s *FileStorage) Upload(_ context.Context, filePath string) (string, error) {
	if s.Directory == "" {
		return "", errors.New("directory is required")
	}

	fileName := filepath.Base(filePath)
	targetPath := filepath.Join(s.Directory, fileName)

	// Copy the file
	source, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open source file: %w", err)
	}
	defer source.Close()

	dest, err := os.Create(targetPath)
	if err != nil {
		return "", fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dest.Close()

	_, err = io.Copy(dest, source)
	if err != nil {
		return "", fmt.Errorf("failed to copy file: %w", err)
	}

	// Extract profile type from filename
	profileType := extractProfileType(fileName)

	// Generate a profile ID
	profileID := generateUUID()

	// Return JSON response like HTTPStorage does
	response := map[string]string{
		"profile_id":  profileID,
		"profile_url": targetPath,
		"type":        profileType,
	}

	responseJSON, err := json.Marshal(response)
	if err != nil {
		return "", fmt.Errorf("failed to marshal response: %w", err)
	}

	return string(responseJSON), nil
}

// StdoutStorage outputs profile data and metadata to stdout for testing purposes
type StdoutStorage struct{}

// NewStdoutStorage creates a new stdout storage instance
func NewStdoutStorage() *StdoutStorage {
	return &StdoutStorage{}
}

// Upload reads the profile file and outputs its contents to stdout in a structured format
func (s *StdoutStorage) Upload(_ context.Context, filePath string) (string, error) {
	// Read the profile file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read profile file: %w", err)
	}

	// Output profile data header
	fmt.Printf("PROFILE_DATA (size: %d bytes):\n", len(data))

	// Try to parse and display the pprof data using go tool pprof
	if displayErr := s.displayPprofData(filePath); displayErr != nil {
		// If parsing fails, show basic info
		fmt.Printf("  Binary pprof data (%d bytes) - use 'go tool pprof %s' to analyze\n", len(data), filePath)
	}

	fmt.Println() // Add separator line

	// Determine profile type from filename for JSON response
	profileType := extractProfileType(filePath)

	// Return JSON response like other storage implementations
	response := map[string]string{
		"profile_id":  StdoutProfileID,
		"profile_url": StdoutProfileURL,
		"type":        profileType,
	}

	jsonData, err := json.Marshal(response)
	if err != nil {
		return "", fmt.Errorf("failed to marshal stdout response: %w", err)
	}

	return string(jsonData), nil
}

// displayPprofData uses go tool pprof to show readable profile information
func (s *StdoutStorage) displayPprofData(filePath string) error {
	// For now, just show basic file information
	info, err := os.Stat(filePath)
	if err != nil {
		return err
	}

	// Determine profile type from filename
	profileType := getDisplayProfileType(filePath)

	fmt.Printf("  Type: %s\n", profileType)
	fmt.Printf("  File: %s\n", filepath.Base(filePath))
	fmt.Printf("  Size: %d bytes\n", info.Size())
	fmt.Printf("  Created: %s\n", info.ModTime().Format("2006-01-02 15:04:05"))
	fmt.Printf("  Analysis: Use 'go tool pprof %s' for detailed analysis\n", filePath)

	return nil
}

// OutputMetadata outputs metadata to stdout in JSON format
func (s *StdoutStorage) OutputMetadata(metadata map[string]string) error {
	jsonData, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	fmt.Printf("METADATA: %s\n", string(jsonData))

	return nil
}

// generateUUID generates a UUID without external dependencies
func generateUUID() string {
	b := make([]byte, UUIDBytesLength)
	_, err := rand.Read(b)

	if err != nil {
		// Fallback to a simple timestamp-based ID if random fails
		return fmt.Sprintf("fallback-%d", time.Now().UnixNano())
	}

	// Set version (4) and variant bits
	b[6] = (b[6] & UUIDVersionMask) | UUIDVersionValue
	b[8] = (b[8] & UUIDVariantMask) | UUIDVariantValue

	return hex.EncodeToString(b)
}

// shouldRetry determines if an HTTP status code should trigger a retry
func shouldRetry(statusCode int) bool {
	return statusCode == HTTPStatusTooManyRequests ||
		(statusCode >= HTTPStatusInternalServerError && statusCode < HTTPStatusServiceUnavailable)
}

// getDisplayProfileType returns a human-readable profile type for display
func getDisplayProfileType(filePath string) string {
	switch {
	case strings.Contains(filePath, ProfileTypeCPU):
		return "CPU Profile"
	case strings.Contains(filePath, ProfileTypeMemory), strings.Contains(filePath, ProfileTypeHeap):
		return "Memory/Heap Profile"
	case strings.Contains(filePath, ProfileTypeGoroutine):
		return "Goroutine Profile"
	case strings.Contains(filePath, ProfileTypeMutex):
		return "Mutex Profile"
	case strings.Contains(filePath, ProfileTypeBlock):
		return "Block Profile"
	default:
		return ProfileTypeUnknown
	}
}
