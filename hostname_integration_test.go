package pprofio

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestHostnameInProfileMetadata(t *testing.T) {
	// Create a channel to capture metadata
	metadataCh := make(chan map[string]string, 1)

	// Create a test server to receive metadata
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/metadata":
			// Capture the metadata
			var metadata map[string]string
			if err := json.NewDecoder(r.Body).Decode(&metadata); err != nil {
				t.Errorf("Failed to decode metadata: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			// Send metadata to channel for verification
			select {
			case metadataCh <- metadata:
			default:
			}

			w.WriteHeader(http.StatusOK)
		case "/upload":
			// Mock profile upload response
			response := map[string]string{
				"profile_id":  "test-profile-123",
				"profile_url": "https://storage.pprofio.com/test-profile-123.pprof",
				"type":        "cpu",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	tests := []struct {
		name             string
		providedHostname string
		expectHostname   string
	}{
		{
			name:             "Auto-populated hostname",
			providedHostname: "", // Empty hostname should be auto-populated
			expectHostname:   "", // Will be checked against actual hostname
		},
		{
			name:             "Custom hostname",
			providedHostname: "custom-test-hostname",
			expectHostname:   "custom-test-hostname",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create storage
			storage := &HTTPStorage{
				URL:     server.URL + "/upload",
				APIKey:  "test-key",
				Client:  server.Client(),
				Retries: 1,
				Env:     "local", // Set to local to bypass HTTPS check in tests
			}

			// Create config
			config := Config{
				APIKey:          "test-key",
				IngestURL:       server.URL,
				SampleRate:      1 * time.Hour, // Long interval to avoid multiple collections
				ProfileDuration: 10 * time.Millisecond,
				Storage:         storage,
				ServiceName:     "test-service",
				Tags:            map[string]string{"env": "test"},
				EnableCPU:       true,
				EnableMemory:    false,
				Hostname:        tt.providedHostname,
			}

			// Create profiler
			p, err := New(config)
			if err != nil {
				t.Fatalf("New() error = %v", err)
			}

			// Start profiler
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			if err := p.Start(ctx); err != nil {
				t.Fatalf("Start() error = %v", err)
			}

			// Wait for initial profile collection
			select {
			case metadata := <-metadataCh:
				// Verify hostname is present
				hostname, ok := metadata["hostname"]
				if !ok {
					t.Error("Hostname not found in metadata")
				}

				// Check expected hostname
				if tt.expectHostname != "" {
					if hostname != tt.expectHostname {
						t.Errorf("Expected hostname %q, got %q", tt.expectHostname, hostname)
					}
				} else {
					// For auto-populated hostname, verify it's either the system hostname or "unknown"
					expectedHostname, err := os.Hostname()
					if err != nil {
						if hostname != "unknown" {
							t.Errorf("Expected hostname to be 'unknown' when os.Hostname() fails, got %q", hostname)
						}
					} else if hostname != expectedHostname && hostname != "unknown" {
						t.Errorf("Expected hostname to be %q or 'unknown', got %q", expectedHostname, hostname)
					}
				}

				// Verify other metadata fields are present
				requiredFields := []string{"profile_id", "profile_url", "service", "type", "timestamp"}
				for _, field := range requiredFields {
					if _, ok := metadata[field]; !ok {
						t.Errorf("Required field %q not found in metadata", field)
					}
				}

				// Verify custom tags are included
				if metadata["env"] != "test" {
					t.Errorf("Custom tag 'env' not found or incorrect in metadata")
				}

			case <-time.After(2 * time.Second):
				t.Error("Timeout waiting for metadata")
			}

			// Stop profiler
			p.Stop()
		})
	}
}
