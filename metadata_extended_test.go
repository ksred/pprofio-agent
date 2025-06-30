package pprofio

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProfiler_sendMetadata_Success(t *testing.T) {
	// Create a test server that accepts metadata
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/metadata" {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Create a profiler with proper configuration
	config := Config{
		APIKey:      "test-key",
		IngestURL:   server.URL,
		ServiceName: "test-service",
	}

	profiler, err := newProfiler(config)
	if err != nil {
		t.Fatalf("newProfiler() error = %v", err)
	}

	// Test successful metadata send
	metadata := map[string]string{
		"profile_id":  "test-123",
		"profile_url": "https://test.com/profile.pprof",
		"service":     "test-service",
		"type":        "cpu",
	}

	ctx := context.Background()
	err = profiler.sendMetadata(ctx, metadata)
	if err != nil {
		t.Errorf("sendMetadata() should succeed, got error: %v", err)
	}
}

func TestNewMetadataClient_Coverage(t *testing.T) {
	// Test creating a metadata client
	client := newMetadataClient("https://api.pprofio.com", "test-key")
	
	if client == nil {
		t.Error("newMetadataClient() should return a valid client")
	}
	
	if client.client == nil {
		t.Error("newMetadataClient() should set up HTTP client")
	}
}

func TestMetadata_sendMetadata_InternalFunction(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create metadata client
	client := newMetadataClient(server.URL, "test-key")
	
	// Test the internal sendMetadata function
	metadata := map[string]string{
		"service": "test-service",
		"type":    "cpu",
	}
	
	ctx := context.Background()
	err := client.sendMetadata(ctx, metadata)
	if err != nil {
		t.Errorf("sendMetadata() error = %v", err)
	}
}

