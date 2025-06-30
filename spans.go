package pprofio

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"
)

// spanKey is used as a context key for storing profiler instances.
// This unexported type prevents collisions with other context keys.
type spanKey struct{}

// Span represents a custom performance measurement with timing and metadata.
// Spans are used to track user-defined operations and are collected along
// with system profiles to provide comprehensive performance insights.
//
// Spans should be created using StartSpan() and must call End() when the
// operation completes to record the duration.
//
// Example usage:
//
//	ctx, span := pprofio.StartSpan(ctx, "database_query", "table", "users")
//	defer span.End()
//	// ... perform database operation ...
type Span struct {
	// Name identifies the operation being measured (e.g., "http_request", "database_query").
	Name string

	// Start is the timestamp when the span began.
	Start time.Time

	// Duration is how long the operation took. Set automatically by End().
	Duration time.Duration

	// Tags provide additional metadata about the operation.
	// Common tags include endpoint, method, table, user_id, etc.
	Tags map[string]string
}

// End marks the completion of the span and calculates its duration.
// This method should be called when the measured operation completes,
// typically using defer immediately after creating the span.
//
// Example:
//
//	ctx, span := pprofio.StartSpan(ctx, "api_call")
//	defer span.End() // Automatically records duration when function returns
func (s *Span) End() {
	s.Duration = time.Since(s.Start)
	// Queue for upload - actual implementation would send to profiler
}

func (p *Profiler) processCustomSpans(ctx context.Context) {
	defer p.wg.Done()

	// Map to collect spans by name
	spans := make(map[string][]*Span)

	// Lock for spans map
	var spansLock sync.Mutex

	// Ticker for periodic flushing
	flushTicker := time.NewTicker(p.config.SampleRate)
	defer flushTicker.Stop()

	for {
		select {
		case span := <-p.spanCh:
			spansLock.Lock()
			spans[span.Name] = append(spans[span.Name], span)
			spansLock.Unlock()

		case <-flushTicker.C:
			// Take a snapshot of current spans and reset
			spansLock.Lock()
			if len(spans) > 0 {
				snapshotSpans := spans
				spans = make(map[string][]*Span)
				spansLock.Unlock()

				// Process spans in a separate goroutine to avoid blocking
				go func() {
					if err := p.processSpans(ctx, snapshotSpans); err != nil {
						fmt.Fprintf(os.Stderr, "Error processing spans: %v\n", err)
					}
				}()
			} else {
				spansLock.Unlock()
			}

		case <-p.stopCh:
			return

		case <-ctx.Done():
			return
		}
	}
}

func (p *Profiler) processSpans(ctx context.Context, spans map[string][]*Span) error {
	// This would convert spans to a pprof-compatible format
	// and upload them as a custom profile

	// Placeholder implementation
	return nil
}
