package pprofio

import (
	"bytes"
	"compress/gzip"
	"context"
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

// Storage defines the interface for uploading profile data.
// Implementations can send profiles to remote APIs, local storage, or other destinations.
//
// The Upload method should handle compression, retries, and error handling as appropriate
// for the storage backend. It returns a response string that may contain metadata
// about the uploaded profile (e.g., profile ID, URL).
type Storage interface {
	// Upload uploads a profile file to the storage backend.
	// The filePath points to a temporary pprof file that should be read and uploaded.
	// Returns a response string (often JSON) containing upload metadata.
	Upload(ctx context.Context, filePath string) (string, error)
}

// HTTPStorage uploads profiles to a remote HTTP API with authentication and retry logic.
// It automatically compresses profiles using gzip and includes authentication headers.
//
// HTTPStorage is the recommended storage for production use with the Pprofio service.
type HTTPStorage struct {
	// URL is the endpoint for uploading profiles (e.g., "https://api.pprofio.com/upload").
	URL string

	// APIKey is the authentication token for the remote API.
	APIKey string

	// Client is the HTTP client used for requests. If nil, a default client is used.
	Client *http.Client

	// Retries is the number of retry attempts for failed uploads. Default: 3.
	Retries int

	// Env specifies the environment for security validation.
	// In non-"local" environments, HTTPS is required.
	Env string
}

// NewHTTPStorage creates a new HTTP storage instance with default settings.
// It configures automatic retries and appropriate timeouts for production use.
//
// Parameters:
//   - url: The upload endpoint URL (must be HTTPS in production)
//   - apiKey: Authentication token for the API
//   - env: Environment identifier ("local", "production", etc.)
//
// Example:
//
//	storage := pprofio.NewHTTPStorage("https://api.pprofio.com/upload", "your-api-key", "production")
func NewHTTPStorage(url, apiKey, env string) *HTTPStorage {
	return &HTTPStorage{
		URL:     url,
		APIKey:  apiKey,
		Client:  &http.Client{Timeout: 30 * time.Second},
		Retries: 3,
		Env:     env,
	}
}

// Upload compresses and uploads a profile file to the configured HTTP endpoint.
// It includes automatic retry logic with exponential backoff for transient failures.
//
// The method validates HTTPS usage in production environments and handles
// authentication, compression, and error responses appropriately.
func (s *HTTPStorage) Upload(ctx context.Context, filePath string) (string, error) {
	if s.URL == "" || s.APIKey == "" {
		return "", errors.New("URL and APIKey are required")
	}

	// Validate URL format and ensure HTTPS
	parsedURL, err := url.Parse(s.URL)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}
	if parsedURL.Scheme != "https" && s.Env != "local" {
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
			backoff := math.Pow(2, float64(attempt-1)) * 100
			time.Sleep(time.Duration(backoff) * time.Millisecond)
		}

		// Create the request
		req, err := http.NewRequestWithContext(ctx, "POST", s.URL, bytes.NewReader(data))
		if err != nil {
			lastErr = fmt.Errorf("failed to create request: %w", err)
			continue
		}

		req.Header.Set("Content-Type", "application/octet-stream")
		req.Header.Set("Content-Encoding", "gzip")
		req.Header.Set("Authorization", "Bearer "+s.APIKey)
		req.Header.Set("User-Agent", GetBuildInfo().UserAgent())

		// Send the request
		resp, err := s.Client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			continue
		}

		defer resp.Body.Close()

		// Handle HTTP errors
		if resp.StatusCode == 401 || resp.StatusCode == 403 {
			return "", fmt.Errorf("authentication failed: %d", resp.StatusCode)
		}

		if resp.StatusCode == 429 || (resp.StatusCode >= 500 && resp.StatusCode < 600) {
			lastErr = fmt.Errorf("server error: %d", resp.StatusCode)
			continue
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
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

// FileStorage saves profiles to local files in a specified directory.
// This storage type is useful for development, testing, or when profiles
// need to be processed by external tools.
//
// Files are saved with their original names and can be analyzed using
// standard Go profiling tools like `go tool pprof`.
type FileStorage struct {
	// Directory is the path where profile files will be saved.
	// The directory will be created if it doesn't exist.
	Directory string
}

// NewFileStorage creates a new file storage instance for the specified directory.
// It automatically creates the directory if it doesn't exist with appropriate permissions.
//
// Parameters:
//   - directory: Path where profile files should be saved
//
// Example:
//
//	storage, err := pprofio.NewFileStorage("/var/log/profiles")
//	if err != nil {
//		log.Fatal(err)
//	}
func NewFileStorage(directory string) (*FileStorage, error) {
	if directory == "" {
		return nil, errors.New("directory is required")
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(directory, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	return &FileStorage{Directory: directory}, nil
}

func (s *FileStorage) Upload(ctx context.Context, filePath string) (string, error) {
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

	// Return JSON response matching the expected format
	response := map[string]string{
		"profile_id":  "file-" + fmt.Sprintf("%d", time.Now().Unix()),
		"profile_url": targetPath,
		"type":        s.detectProfileType(filePath),
	}
	
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		return "", fmt.Errorf("failed to marshal response: %w", err)
	}
	
	return string(jsonResponse), nil
}

// detectProfileType determines the profile type from the filename for FileStorage
func (s *FileStorage) detectProfileType(filePath string) string {
	if strings.Contains(filePath, "cpu") {
		return "cpu"
	} else if strings.Contains(filePath, "memory") || strings.Contains(filePath, "heap") {
		return "memory"
	} else if strings.Contains(filePath, "goroutine") {
		return "goroutine"
	} else if strings.Contains(filePath, "mutex") {
		return "mutex"
	} else if strings.Contains(filePath, "block") {
		return "block"
	}
	return "unknown"
}

// StdoutStorage outputs profile data and metadata to stdout for testing and debugging.
// This storage type is primarily used for development, testing, and CI/CD pipelines
// where profiles need to be captured in logs or processed by external tools.
//
// The output format includes profile metadata and guidance for analysis with
// standard Go tooling.
type StdoutStorage struct{}

// NewStdoutStorage creates a new stdout storage instance.
// This storage type requires no configuration and is safe for concurrent use.
//
// Example:
//
//	cfg := pprofio.Config{
//		ServiceName:    "my-service",
//		OutputToStdout: true,
//		Storage:        pprofio.NewStdoutStorage(),
//	}
func NewStdoutStorage() *StdoutStorage {
	return &StdoutStorage{}
}

// Upload reads the profile file and outputs its contents to stdout in a structured format.
// The output includes profile type detection, file metadata, and instructions for
// further analysis using go tool pprof.
func (s *StdoutStorage) Upload(ctx context.Context, filePath string) (string, error) {
	// Read the profile file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read profile file: %w", err)
	}

	// Determine profile type from filename
	profileType := s.detectProfileType(filePath)

	// Output profile data header
	fmt.Printf("PROFILE_DATA (size: %d bytes):\n", len(data))

	// Try to parse and display the pprof data using go tool pprof
	if err := s.displayPprofData(filePath); err != nil {
		// If parsing fails, show basic info
		fmt.Printf("  Binary pprof data (%d bytes) - use 'go tool pprof %s' to analyze\n", len(data), filePath)
	}

	fmt.Println() // Add separator line

	// Return JSON response matching the expected format
	response := map[string]string{
		"profile_id":  "stdout-" + fmt.Sprintf("%d", time.Now().Unix()),
		"profile_url": "stdout://local",
		"type":        profileType,
	}
	
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		return "", fmt.Errorf("failed to marshal response: %w", err)
	}
	
	return string(jsonResponse), nil
}

// detectProfileType determines the profile type from the filename
func (s *StdoutStorage) detectProfileType(filePath string) string {
	if strings.Contains(filePath, "cpu") {
		return "cpu"
	} else if strings.Contains(filePath, "memory") || strings.Contains(filePath, "heap") {
		return "memory"
	} else if strings.Contains(filePath, "goroutine") {
		return "goroutine"
	} else if strings.Contains(filePath, "mutex") {
		return "mutex"
	} else if strings.Contains(filePath, "block") {
		return "block"
	}
	return "unknown"
}

// displayPprofData uses go tool pprof to show readable profile information
func (s *StdoutStorage) displayPprofData(filePath string) error {
	// For now, just show basic file information
	info, err := os.Stat(filePath)
	if err != nil {
		return err
	}

	// Determine profile type from filename
	profileType := "unknown"
	if strings.Contains(filePath, "cpu") {
		profileType = "CPU Profile"
	} else if strings.Contains(filePath, "memory") || strings.Contains(filePath, "heap") {
		profileType = "Memory/Heap Profile"
	} else if strings.Contains(filePath, "goroutine") {
		profileType = "Goroutine Profile"
	} else if strings.Contains(filePath, "mutex") {
		profileType = "Mutex Profile"
	} else if strings.Contains(filePath, "block") {
		profileType = "Block Profile"
	}

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
