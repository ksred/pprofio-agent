package pprofio

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	DefaultMetadataTimeout = 10 * time.Second
	DefaultMetadataRetries = 3
	BackoffBaseMs          = 100
)

// metadataClient handles sending profile metadata to the ingest API
type metadataClient struct {
	ingestURL string
	apiKey    string
	client    *http.Client
	retries   int
}

func newMetadataClient(ingestURL, apiKey string) *metadataClient {
	return &metadataClient{
		ingestURL: ingestURL,
		apiKey:    apiKey,
		client:    &http.Client{Timeout: DefaultMetadataTimeout},
		retries:   DefaultMetadataRetries,
	}
}

func (m *metadataClient) sendMetadata(ctx context.Context, metadata map[string]string) error {
	// Validate URL
	parsedURL, err := url.Parse(m.ingestURL)
	if err != nil {
		metadataErr := fmt.Errorf("invalid ingest URL: %w", err)
		fmt.Fprintf(os.Stderr, "pprofio: %v\n", metadataErr)

		return metadataErr
	}
	// Skip HTTPS check in tests if running a localhost/127.0.0.1 URL
	if parsedURL.Scheme != "https" && !strings.Contains(parsedURL.Host, "localhost") &&
		!strings.Contains(parsedURL.Host, "127.0.0.1") {
		httpsErr := fmt.Errorf("HTTPS is required for ingest URL")
		fmt.Fprintf(os.Stderr, "pprofio: metadata endpoint requires HTTPS: %s\n", m.ingestURL)

		return httpsErr
	}

	// Marshal metadata to JSON
	payload, err := json.Marshal(metadata)
	if err != nil {
		marshalErr := fmt.Errorf("failed to marshal metadata: %w", err)
		fmt.Fprintf(os.Stderr, "pprofio: %v\n", marshalErr)

		return marshalErr
	}

	// Send with retries
	var lastErr error

	for attempt := 0; attempt < m.retries; attempt++ {
		if err := m.sendRequest(ctx, payload); err != nil {
			lastErr = err
			fmt.Fprintf(os.Stderr, "pprofio: error sending metadata to %s/metadata: %v (attempt %d/%d)\n", m.ingestURL, err, attempt+1, m.retries)
			// Exponential backoff
			backoffMs := (1 << attempt) * BackoffBaseMs
			time.Sleep(time.Duration(backoffMs) * time.Millisecond)

			continue
		}

		return nil
	}

	finalErr := fmt.Errorf("failed to send metadata after %d attempts: %w", m.retries, lastErr)
	fmt.Fprintf(os.Stderr, "pprofio: %v\n", finalErr)

	return finalErr
}

func (m *metadataClient) sendRequest(ctx context.Context, payload []byte) error {
	req, err := http.NewRequestWithContext(ctx, "POST", m.ingestURL+"/metadata", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+m.apiKey)

	resp, err := m.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// Update the Profiler to use the metadata client
func (p *Profiler) sendMetadata(ctx context.Context, metadata map[string]string) error {
	client := newMetadataClient(p.config.IngestURL, p.config.APIKey)
	return client.sendMetadata(ctx, metadata)
}
