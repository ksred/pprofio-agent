package pprofio

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"
)

const (
	// DefaultHTTPClientTimeout is the default timeout for HTTP requests
	DefaultHTTPClientTimeout = 30 * time.Second
)

// HTTPMetricsStorage handles sending HTTP metrics to the Pprofio API endpoints
type HTTPMetricsStorage struct {
	baseURL string
	apiKey  string
	client  *http.Client

	// Batching configuration
	batchSize     int
	flushInterval time.Duration

	// Batching state
	mu        sync.Mutex
	batch     []*RequestMetrics
	lastFlush time.Time
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
}

// BatchMetrics represents a batch of HTTP metrics for submission
type BatchMetrics struct {
	Metrics  []*RequestMetrics `json:"metrics"`
	Metadata BatchMetadata     `json:"metadata"`
}

// BatchMetadata contains metadata about the batch submission
type BatchMetadata struct {
	BatchID   string    `json:"batch_id"`
	Source    string    `json:"source"`
	Version   string    `json:"version"`
	SentAt    time.Time `json:"sent_at"`
	BatchSize int       `json:"batch_size"`
	Service   string    `json:"service"`
}

// NewHTTPMetricsStorage creates a new HTTP metrics storage backend
func NewHTTPMetricsStorage(baseURL, apiKey string, batchSize int, flushInterval time.Duration) *HTTPMetricsStorage {
	ctx, cancel := context.WithCancel(context.Background())

	storage := &HTTPMetricsStorage{
		baseURL:       baseURL,
		apiKey:        apiKey,
		client:        &http.Client{Timeout: DefaultHTTPClientTimeout},
		batchSize:     batchSize,
		flushInterval: flushInterval,
		batch:         make([]*RequestMetrics, 0, batchSize),
		lastFlush:     time.Now(),
		ctx:           ctx,
		cancel:        cancel,
	}

	// Start background flusher
	storage.wg.Add(1)
	go storage.backgroundFlusher()

	return storage
}

// SubmitMetric adds a metric to the batch and potentially triggers a flush
func (h *HTTPMetricsStorage) SubmitMetric(metric *RequestMetrics) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.batch = append(h.batch, metric)

	// Check if we should flush due to batch size
	if len(h.batch) >= h.batchSize {
		return h.flushBatch()
	}

	return nil
}

// SubmitSingle sends a single HTTP metric immediately
func (h *HTTPMetricsStorage) SubmitSingle(metric *RequestMetrics) error {
	url := fmt.Sprintf("%s/api/v1/http-metrics", h.baseURL)

	jsonData, err := json.Marshal(metric)
	if err != nil {
		return fmt.Errorf("failed to marshal metric: %w", err)
	}

	req, err := http.NewRequestWithContext(h.ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+h.apiKey)

	resp, err := h.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send HTTP metric: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("HTTP metrics API returned status %d", resp.StatusCode)
	}

	return nil
}

// flushBatch sends the current batch to the API (must be called with mutex held)
func (h *HTTPMetricsStorage) flushBatch() error {
	if len(h.batch) == 0 {
		return nil
	}

	url := fmt.Sprintf("%s/api/v1/http-metrics/batch", h.baseURL)

	// Create batch with metadata
	batchMetrics := BatchMetrics{
		Metrics: h.batch,
		Metadata: BatchMetadata{
			BatchID:   fmt.Sprintf("batch_%d", time.Now().UnixNano()),
			Source:    "pprofio-agent",
			Version:   Version,
			SentAt:    time.Now(),
			BatchSize: len(h.batch),
			Service:   h.getServiceFromBatch(),
		},
	}

	jsonData, err := json.Marshal(batchMetrics)
	if err != nil {
		return fmt.Errorf("failed to marshal batch: %w", err)
	}

	req, err := http.NewRequestWithContext(h.ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create batch request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+h.apiKey)

	resp, err := h.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send HTTP metrics batch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("HTTP metrics batch API returned status %d", resp.StatusCode)
	}

	// Clear the batch after successful submission
	h.batch = h.batch[:0]
	h.lastFlush = time.Now()

	return nil
}

// getServiceFromBatch extracts service name from the first metric in the batch
func (h *HTTPMetricsStorage) getServiceFromBatch() string {
	if len(h.batch) > 0 && h.batch[0].Service != "" {
		return h.batch[0].Service
	}

	return ProfileTypeUnknown
}

// backgroundFlusher runs in the background and flushes batches based on time intervals
func (h *HTTPMetricsStorage) backgroundFlusher() {
	defer h.wg.Done()

	ticker := time.NewTicker(h.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			h.mu.Lock()
			if len(h.batch) > 0 && time.Since(h.lastFlush) >= h.flushInterval {
				if err := h.flushBatch(); err != nil {
					// Log error but don't stop the flusher
					fmt.Fprintf(os.Stderr, "Error flushing HTTP metrics batch: %v\n", err)
				}
			}
			h.mu.Unlock()

		case <-h.ctx.Done():
			// Final flush before shutdown
			h.mu.Lock()
			if len(h.batch) > 0 {
				if err := h.flushBatch(); err != nil {
					fmt.Fprintf(os.Stderr, "Error in final HTTP metrics batch flush: %v\n", err)
				}
			}
			h.mu.Unlock()

			return
		}
	}
}

// Flush immediately flushes any pending metrics
func (h *HTTPMetricsStorage) Flush() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	return h.flushBatch()
}

// Close stops the background flusher and flushes any remaining metrics
func (h *HTTPMetricsStorage) Close() error {
	h.cancel()
	h.wg.Wait()

	// Final flush
	h.mu.Lock()
	defer h.mu.Unlock()

	return h.flushBatch()
}
