package search

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type healthState struct {
	fails     int
	skipUntil time.Time
}

var (
	sourceHealthMap = sync.Map{}
	failThreshold   = 3
	cooldownDur     = 10 * time.Minute
)

// CircuitBreakerSource wraps a Source to stop querying it if it's down.
type CircuitBreakerSource struct {
	source Source
}

func NewCircuitBreakerSource(s Source) *CircuitBreakerSource {
	return &CircuitBreakerSource{
		source: s,
	}
}

func (c *CircuitBreakerSource) ID() string {
	return c.source.ID()
}

func (c *CircuitBreakerSource) Name() string {
	return c.source.Name()
}

func (c *CircuitBreakerSource) Search(ctx context.Context, query string) ([]TorrentResult, error) {
	id := c.source.ID()

	// Check if skipped
	if val, ok := sourceHealthMap.Load(id); ok {
		h := val.(healthState)
		if time.Now().Before(h.skipUntil) {
			return nil, fmt.Errorf("source %s is currently benched (circuit broken)", id)
		}
	}

	res, err := c.source.Search(ctx, query)

	if err != nil {
		// Record failure
		val, _ := sourceHealthMap.LoadOrStore(id, healthState{})
		h := val.(healthState)
		h.fails++
		if h.fails >= failThreshold {
			h.skipUntil = time.Now().Add(cooldownDur)
		}
		sourceHealthMap.Store(id, h)
	} else {
		// Record success (reset)
		sourceHealthMap.Delete(id)
	}

	return res, err
}
