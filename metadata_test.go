package pprofio

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestMetadataClient(t *testing.T) {
	// Create a test server to receive metadata
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request
		if r.URL.Path != "/metadata" {
			t.Errorf("Expected URL path '/metadata', got '%s'", r.URL.Path)
		}

		if r.Method != "POST" {
			t.Errorf("Expected method 'POST', got '%s'", r.Method)
		}

		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("Expected Authorization header 'Bearer test-key', got '%s'", r.Header.Get("Authorization"))
		}

		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type header 'application/json', got '%s'", r.Header.Get("Content-Type"))
		}

		// Success response
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create metadata client
	client := newMetadataClient(server.URL, "test-key")
	client.client = server.Client()

	// Test sending metadata
	metadata := map[string]string{
		"profile_id":   "test-profile-id",
		"service_name": "test-service",
		"profile_type": "cpu",
		"timestamp":    "1609459200", // 2021-01-01 00:00:00
		"env":          "test",
	}

	err := client.sendMetadata(context.Background(), metadata)
	if err != nil {
		t.Fatalf("sendMetadata() error = %v", err)
	}

	// Test with error responses
	errorCases := []struct {
		name       string
		statusCode int
		expectErr  bool
	}{
		{
			name:       "Unauthorized",
			statusCode: http.StatusUnauthorized,
			expectErr:  true,
		},
		{
			name:       "Server Error",
			statusCode: http.StatusInternalServerError,
			expectErr:  true,
		},
		{
			name:       "Success",
			statusCode: http.StatusOK,
			expectErr:  false,
		},
	}

	for _, tc := range errorCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a test server with the specific status code
			errServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
			}))
			defer errServer.Close()

			// Create client
			client := newMetadataClient(errServer.URL, "test-key")
			client.client = errServer.Client()
			client.retries = 1 // Reduce retries for faster tests

			// Test sending metadata
			err := client.sendMetadata(context.Background(), metadata)

			if tc.expectErr && err == nil {
				t.Errorf("sendMetadata() expected error, got nil")
			}

			if !tc.expectErr && err != nil {
				t.Errorf("sendMetadata() unexpected error: %v", err)
			}
		})
	}
}

func TestProfilerSendMetadata(t *testing.T) {
	// Create a test server to receive metadata
	metadataReceived := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/metadata" {
			metadataReceived = true
			w.WriteHeader(http.StatusOK)
		} else {
			// For profile uploads
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("https://storage.pprofio.com/profile123"))
		}
	}))
	defer server.Close()

	// Create a profiler that uses our test server
	storage := &HTTPStorage{
		URL:     server.URL,
		APIKey:  "test-key",
		Client:  server.Client(),
		Retries: 1,
	}

	config := Config{
		APIKey:          "test-key",
		IngestURL:       server.URL,
		SampleRate:      100 * time.Millisecond,
		ProfileDuration: 50 * time.Millisecond,
		Storage:         storage,
		ServiceName:     "test-service",
		Tags:            map[string]string{"env": "test"},
		EnableCPU:       true,
	}

	p, err := newProfiler(config)
	if err != nil {
		t.Fatalf("newProfiler() error = %v", err)
	}

	// Directly test sending metadata
	metadata := map[string]string{
		"profile_id":   "test-profile-id",
		"service_name": "test-service",
		"profile_type": "cpu",
		"timestamp":    "1609459200",
	}

	err = p.sendMetadata(context.Background(), metadata)
	if err != nil {
		t.Fatalf("sendMetadata() error = %v", err)
	}

	if !metadataReceived {
		t.Error("Metadata was not received by the server")
	}
}
