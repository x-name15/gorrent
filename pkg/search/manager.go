package search

import (
	"context"
	"github.com/x-name15/gorrent/pkg/config"
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
func (m *Manager) Search(ctx context.Context, query string) []TorrentResult {
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

		wg.Add(1)
		go func(s Source) {
			defer wg.Done()
			res, err := s.Search(ctx, query)
			if err == nil {
				resultsChan <- res
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

	return filtered
}
