package search

import (
	"context"
	"strings"
	"sync"
	"time"
)

// DefaultMaxCacheEntries is the maximum number of search queries cached per source.
const DefaultMaxCacheEntries = 100

type cacheEntry struct {
	results []TorrentResult
	expires time.Time
}

// CachingSource wraps any Source and adds bounded TTL caching with LRU eviction.
type CachingSource struct {
	source     Source
	ttl        time.Duration
	maxEntries int
	mu         sync.Mutex
	entries    map[string]cacheEntry
	order      []string
}

func NewCachingSource(s Source, ttl time.Duration) *CachingSource {
	return NewCachingSourceWithLimit(s, ttl, DefaultMaxCacheEntries)
}

func NewCachingSourceWithLimit(s Source, ttl time.Duration, maxEntries int) *CachingSource {
	if maxEntries <= 0 {
		maxEntries = DefaultMaxCacheEntries
	}
	return &CachingSource{
		source:     s,
		ttl:        ttl,
		maxEntries: maxEntries,
		entries:    make(map[string]cacheEntry),
		order:      make([]string, 0, maxEntries),
	}
}

func (c *CachingSource) ID() string {
	return c.source.ID()
}

func (c *CachingSource) Name() string {
	return c.source.Name()
}

func (c *CachingSource) promote(key string) {
	newOrder := c.order[:0]
	for _, k := range c.order {
		if k != key {
			newOrder = append(newOrder, k)
		}
	}
	c.order = append(newOrder, key)
}

func (c *CachingSource) Search(ctx context.Context, query string) ([]TorrentResult, error) {
	key := strings.ToLower(strings.TrimSpace(query))

	c.mu.Lock()
	if entry, ok := c.entries[key]; ok {
		if time.Now().Before(entry.expires) {
			c.promote(key)
			results := entry.results
			c.mu.Unlock()
			return results, nil
		}
		// Evict expired entry
		delete(c.entries, key)
		newOrder := c.order[:0]
		for _, k := range c.order {
			if k != key {
				newOrder = append(newOrder, k)
			}
		}
		c.order = newOrder
	}
	c.mu.Unlock()

	results, err := c.source.Search(ctx, query)
	if err == nil {
		c.mu.Lock()
		c.promote(key)
		c.entries[key] = cacheEntry{
			results: results,
			expires: time.Now().Add(c.ttl),
		}

		// Evict oldest entries exceeding maxEntries ceiling
		for len(c.order) > c.maxEntries {
			oldest := c.order[0]
			c.order = c.order[1:]
			delete(c.entries, oldest)
		}
		c.mu.Unlock()
	}

	return results, err
}
