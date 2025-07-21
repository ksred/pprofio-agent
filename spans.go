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

	var spansLock sync.Mutex

	// Ticker for periodic flushing
	flushTicker := time.NewTicker(p.config.SampleRate)
	defer flushTicker.Stop()

	// Track processing goroutines
	var processingWg sync.WaitGroup
	defer processingWg.Wait()

	// Track unique span names to prevent unbounded growth
	spanNameCount := 0

	for {
		select {
		case span := <-p.spanCh:
			spansLock.Lock()
			if p.addSpan(spans, span, &spanNameCount) {
				spansLock.Unlock()
				continue
			}
			spansLock.Unlock()

		case <-flushTicker.C:
			p.flushSpans(ctx, &spansLock, spans, &spanNameCount, &processingWg)

		case <-p.stopCh:
			p.drainAndProcessSpans(ctx, &spansLock, spans, &processingWg)
			return

		case <-ctx.Done():
			return
		}
	}
}

// addSpan adds a span to the collection, returns true if the span was dropped
func (p *Profiler) addSpan(spans map[string][]*Span, span *Span, spanNameCount *int) bool {
	if _, exists := spans[span.Name]; !exists {
		*spanNameCount++
		if *spanNameCount > MaxUniqueSpanNames {
			fmt.Fprintf(os.Stderr, "Warning: Exceeded maximum unique span names (%d), dropping span: %s\n", MaxUniqueSpanNames, span.Name)
			return true
		}
	}

	spans[span.Name] = append(spans[span.Name], span)

	return false
}

// flushSpans flushes collected spans for processing
func (p *Profiler) flushSpans(ctx context.Context, spansLock *sync.Mutex, spans map[string][]*Span,
	spanNameCount *int, processingWg *sync.WaitGroup) {
	spansLock.Lock()
	defer spansLock.Unlock()

	if len(spans) == 0 {
		return
	}

	snapshotSpans := spans

	for k := range spans {
		delete(spans, k)
	}

	*spanNameCount = 0

	processingWg.Add(1)

	processFn := func(snapshot map[string][]*Span) {
		defer processingWg.Done()

		if err := p.processSpans(ctx, snapshot); err != nil {
			fmt.Fprintf(os.Stderr, "Error processing spans: %v\n", err)
		}
	}

	go processFn(snapshotSpans)
}

// drainAndProcessSpans drains the channel and processes any remaining spans
func (p *Profiler) drainAndProcessSpans(ctx context.Context, spansLock *sync.Mutex,
	spans map[string][]*Span, processingWg *sync.WaitGroup) {
	spansLock.Lock()
	defer spansLock.Unlock()

	// Drain any remaining spans in the channel
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
}

func (p *Profiler) processSpans(_ context.Context, _ map[string][]*Span) error {
	// This would convert spans to a pprof-compatible format
	// and upload them as a custom profile
	// Placeholder implementation
	return nil
}
