package pprofio

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// MockJSONStorage implements Storage interface and returns proper JSON responses
type MockJSONStorage struct {
	ShouldFail   bool
	ResponseData map[string]string
}

func NewMockJSONStorage() *MockJSONStorage {
	return &MockJSONStorage{
		ShouldFail: false,
		ResponseData: map[string]string{
			"profile_id":  "mock-profile-id",
			"profile_url": "https://mock.pprofio.com/profiles/mock-profile-id.pprof",
			"type":        "cpu",
		},
	}
}

func (m *MockJSONStorage) Upload(ctx context.Context, filePath string) (string, error) {
	if m.ShouldFail {
		return "", fmt.Errorf("mock upload failed")
	}

	// Determine profile type from filename
	profileType := "unknown"
	if contains(filePath, "cpu") {
		profileType = "cpu"
	} else if contains(filePath, "memory") || contains(filePath, "heap") {
		profileType = "memory"
	} else if contains(filePath, "goroutine") {
		profileType = "goroutine"
	} else if contains(filePath, "mutex") {
		profileType = "mutex"
	} else if contains(filePath, "block") {
		profileType = "block"
	}

	// Create response with correct profile type
	response := map[string]string{
		"profile_id":  "mock-profile-id",
		"profile_url": "https://mock.pprofio.com/profiles/mock-profile-id.pprof",
		"type":        profileType,
	}

	// Override with custom data if provided
	for k, v := range m.ResponseData {
		response[k] = v
	}
	response["type"] = profileType // Always use detected type

	jsonData, err := json.Marshal(response)
	if err != nil {
		return "", fmt.Errorf("failed to marshal mock response: %w", err)
	}

	return string(jsonData), nil
}

// Helper function since strings.Contains isn't available in all contexts
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[len(s)-len(substr):] == substr ||
		(len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// MockFailingJSONStorage for testing error conditions
type MockFailingJSONStorage struct {
	ErrorMessage string
}

func NewMockFailingJSONStorage(errorMsg string) *MockFailingJSONStorage {
	return &MockFailingJSONStorage{
		ErrorMessage: errorMsg,
	}
}

func (m *MockFailingJSONStorage) Upload(ctx context.Context, filePath string) (string, error) {
	return "", errors.New(m.ErrorMessage)
}

// MockInvalidJSONStorage returns invalid JSON for testing JSON parsing errors
type MockInvalidJSONStorage struct{}

func NewMockInvalidJSONStorage() *MockInvalidJSONStorage {
	return &MockInvalidJSONStorage{}
}

func (m *MockInvalidJSONStorage) Upload(ctx context.Context, filePath string) (string, error) {
	return "invalid json response", nil
}
