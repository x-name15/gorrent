package search

import (
	"context"
	"strings"
	"sync"
	"time"
)

type cacheEntry struct {
	results []TorrentResult
	expires time.Time
}

// CachingSource wraps any Source and adds TTL caching
type CachingSource struct {
	source Source
	ttl    time.Duration
	cache  sync.Map
}

func NewCachingSource(s Source, ttl time.Duration) *CachingSource {
	return &CachingSource{
		source: s,
		ttl:    ttl,
	}
}

func (c *CachingSource) ID() string {
	return c.source.ID()
}

func (c *CachingSource) Name() string {
	return c.source.Name()
}

func (c *CachingSource) Search(ctx context.Context, query string) ([]TorrentResult, error) {
	key := strings.ToLower(strings.TrimSpace(query))

	if val, ok := c.cache.Load(key); ok {
		entry := val.(cacheEntry)
		if time.Now().Before(entry.expires) {
			return entry.results, nil
		}
		c.cache.Delete(key) // Evict expired
	}

	results, err := c.source.Search(ctx, query)
	if err == nil {
		c.cache.Store(key, cacheEntry{
			results: results,
			expires: time.Now().Add(c.ttl),
		})
	}
	return results, err
}
