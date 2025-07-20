package pprofio

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"
)

type spanKey struct{}

type Span struct {
	Name     string
	Start    time.Time
	Duration time.Duration
	Tags     map[string]string
}

func (s *Span) End() {
	s.Duration = time.Since(s.Start)
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

	// Track processing goroutines
	var processingWg sync.WaitGroup
	defer processingWg.Wait() // Wait for all processing goroutines to complete

	// Track unique span names to prevent unbounded growth
	spanNameCount := 0

	for {
		select {
		case span := <-p.spanCh:
			spansLock.Lock()
			// Check if this is a new span name
			if _, exists := spans[span.Name]; !exists {
				spanNameCount++
				// If we've exceeded the limit, drop the span and log warning
				if spanNameCount > MaxUniqueSpanNames {
					spansLock.Unlock()
					fmt.Fprintf(os.Stderr, "Warning: Exceeded maximum unique span names (%d), dropping span: %s\n", MaxUniqueSpanNames, span.Name)
					continue
				}
			}
			spans[span.Name] = append(spans[span.Name], span)
			spansLock.Unlock()

		case <-flushTicker.C:
			// Take a snapshot of current spans and reset
			spansLock.Lock()
			if len(spans) > 0 {
				snapshotSpans := spans
				spans = make(map[string][]*Span)
				spanNameCount = 0 // Reset the count
				spansLock.Unlock()

				// Process spans in a separate goroutine to avoid blocking
				processingWg.Add(1)
				go func(snapshot map[string][]*Span) {
					defer processingWg.Done()
					if err := p.processSpans(ctx, snapshot); err != nil {
						fmt.Fprintf(os.Stderr, "Error processing spans: %v\n", err)
					}
				}(snapshotSpans)
			} else {
				spansLock.Unlock()
			}

		case <-p.stopCh:
			// Drain any remaining spans in the channel
			spansLock.Lock()
			drainLoop:
			for {
				select {
				case span := <-p.spanCh:
					spans[span.Name] = append(spans[span.Name], span)
				default:
					break drainLoop
				}
			}
			// Process any remaining spans
			if len(spans) > 0 {
				processingWg.Add(1)
				finalSpans := spans
				go func(snapshot map[string][]*Span) {
					defer processingWg.Done()
					if err := p.processSpans(ctx, snapshot); err != nil {
						fmt.Fprintf(os.Stderr, "Error processing final spans: %v\n", err)
					}
				}(finalSpans)
			}
			spansLock.Unlock()
			return

		case <-ctx.Done():
			return
		}
	}
}

func (p *Profiler) processSpans(_ context.Context, _ map[string][]*Span) error {
	// This would convert spans to a pprof-compatible format
	// and upload them as a custom profile
	// Placeholder implementation
	return nil
}
