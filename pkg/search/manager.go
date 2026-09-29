package search

import (
	"context"
	"github.com/x-name15/gorrent/pkg/config"
	"github.com/x-name15/gorrent/pkg/logger"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// Manager handles executing multiple scrapers and scoring the results.
type Manager struct {
	sources []Source
	config  *config.ScraperConfig
}

// NewManager creates a new scraper manager.
func NewManager(cfg *config.ScraperConfig) *Manager {
	return &Manager{
		sources: []Source{},
		config:  cfg,
	}
}

// Register adds a source to the manager.
func (m *Manager) Register(s Source) {
	m.sources = append(m.sources, s)
}

// Search executes all registered scrapers concurrently.
func (m *Manager) Search(ctx context.Context, query string, targetSource string) []TorrentResult {
	var wg sync.WaitGroup
	resultsChan := make(chan []TorrentResult, len(m.sources))

	// Build a fast lookup map for enabled sources
	enabled := make(map[string]bool)
	for _, s := range m.config.Sources {
		enabled[s] = true
	}

	for _, source := range m.sources {
		if !enabled[source.ID()] {
			continue // Skip deactivated modules
		}

		if targetSource != "" && !strings.EqualFold(source.ID(), targetSource) {
			continue // Skip if specific source requested and doesn't match
		}

		wg.Add(1)
		go func(s Source) {
			defer wg.Done()
			res, err := s.Search(ctx, query)
			if err == nil {
				logger.Debugf("Scraper [%s] returned %d results for '%s'", s.ID(), len(res), query)
				resultsChan <- res
			} else {
				logger.Debugf("Scraper [%s] error for '%s': %v", s.ID(), query, err)
			}
		}(source)
	}

	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	var allResults []TorrentResult
	for res := range resultsChan {
		allResults = append(allResults, res...)
	}

	return m.ScoreAndFilter(allResults)
}

// ScoreAndFilter applies heuristics to rank torrents based on config filters.
func (m *Manager) ScoreAndFilter(results []TorrentResult) []TorrentResult {
	logger.Debugf("ScoreAndFilter: Processing %d total results", len(results))
	var filtered []TorrentResult

	minSeedersStr := m.config.Filters["min_seeders"]
	minSeeders := 0
	if minSeedersStr != "" {
		minSeeders, _ = strconv.Atoi(minSeedersStr)
	}

	// Pre-compile regexes for agnostic term matching (word boundaries)
	termRegexes := make(map[string]*regexp.Regexp)
	for key, val := range m.config.Filters {
		if key == "min_seeders" || val == "" {
			continue
		}
		terms := strings.Split(strings.ToLower(val), ",")
		for _, term := range terms {
			term = strings.TrimSpace(term)
			if term != "" && termRegexes[term] == nil {
				pattern := `(?:^|[^a-z0-9])` + regexp.QuoteMeta(term) + `(?:[^a-z0-9]|$)`
				termRegexes[term] = regexp.MustCompile(pattern)
			}
		}
	}

	for _, r := range results {
		// Hard filter
		if r.Seeders < minSeeders {
			continue
		}

		score := r.Seeders
		nameLower := strings.ToLower(r.Name)

		// Agnostic Scoring Logic: Apply +1000 for each matched term from config
		for key, val := range m.config.Filters {
			if key == "min_seeders" || val == "" {
				continue
			}

			terms := strings.Split(strings.ToLower(val), ",")
			for _, term := range terms {
				term = strings.TrimSpace(term)
				if term == "" {
					continue
				}

				if re, ok := termRegexes[term]; ok && re.MatchString(nameLower) {
					if key == "resolution" {
						score += 2000 // Give resolution priority
					} else {
						score += 1000
					}
				}
			}
		}

		r.Score = score
		filtered = append(filtered, r)
	}

	// Sort by score descending
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Score > filtered[j].Score
	})

	deduped := m.DedupeAndMerge(filtered)
	logger.Debugf("ScoreAndFilter: Returning %d deduplicated results (from %d)", len(deduped), len(filtered))
	return deduped
}

// DedupeAndMerge folds rows for the same infohash across multiple sources into one.
// The winning row (higher score, or higher seeders on tie) keeps its identity,
// while missing metadata (such as Added timestamp) and unique trackers (&tr=...)
// from losing rows are merged into the winning magnet URI.
func (m *Manager) DedupeAndMerge(results []TorrentResult) []TorrentResult {
	if len(results) <= 1 {
		return results
	}

	byHash := make(map[string]int) // hash -> index in deduplicated list
	var deduped []TorrentResult

	for _, r := range results {
		h := NormalizeInfoHash(r.InfoHash)
		if h == "" {
			h = ExtractInfoHash(r.Magnet)
		}
		if h == "" {
			deduped = append(deduped, r)
			continue
		}
		r.InfoHash = h

		if idx, exists := byHash[h]; exists {
			existing := deduped[idx]
			win, lose := existing, r
			if r.Score > existing.Score || (r.Score == existing.Score && r.Seeders > existing.Seeders) {
				win, lose = r, existing
			}

			if win.Added == 0 && lose.Added > 0 {
				win.Added = lose.Added
			}

			win.Magnet = MergeMagnetTrackers(win.Magnet, lose.Magnet)
			deduped[idx] = win
		} else {
			byHash[h] = len(deduped)
			deduped = append(deduped, r)
		}
	}

	sort.Slice(deduped, func(i, j int) bool {
		if deduped[i].Score == deduped[j].Score {
			return deduped[i].Seeders > deduped[j].Seeders
		}
		return deduped[i].Score > deduped[j].Score
	})

	return deduped
}
