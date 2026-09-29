package tests

import (
	"strings"
	"testing"

	"github.com/x-name15/gorrent/pkg/config"
	"github.com/x-name15/gorrent/pkg/search"
)

func TestManager_DedupeAndMerge(t *testing.T) {
	mgr := search.NewManager(&config.ScraperConfig{Filters: map[string]string{}})

	hash := "0123456789abcdef0123456789abcdef01234567"
	otherHash := "ffffffffffffffffffffffffffffffffffffffff"

	results := []search.TorrentResult{
		{
			InfoHash: hash,
			Name:     "Release 1080p TPB",
			Seeders:  20,
			Score:    20,
			Source:   "piratebay",
			Magnet:   "magnet:?xt=urn:btih:" + hash + "&dn=Release+1080p+TPB&tr=udp%3A%2F%2Ftracker.tpb.org%3A6969",
		},
		{
			InfoHash: strings.ToUpper(hash), // tests case normalization
			Name:     "Release 1080p 1337x",
			Seeders:  90,
			Score:    90,
			Source:   "1337x",
			Added:    1700000000,
			Magnet:   "magnet:?xt=urn:btih:" + hash + "&dn=Release+1080p+1337x&tr=udp%3A%2F%2Ftracker.1337x.org%3A1337",
		},
		{
			InfoHash: otherHash,
			Name:     "Completely Different Movie",
			Seeders:  5,
			Score:    5,
			Source:   "yts",
			Magnet:   "magnet:?xt=urn:btih:" + otherHash + "&dn=Different+Movie",
		},
	}

	deduped := mgr.DedupeAndMerge(results)

	if len(deduped) != 2 {
		t.Fatalf("expected exactly 2 deduplicated results, got %d", len(deduped))
	}

	winner := deduped[0]
	if winner.Source != "1337x" {
		t.Errorf("expected 1337x to win with score 90, got %s", winner.Source)
	}
	if winner.Seeders != 90 {
		t.Errorf("expected 90 seeders, got %d", winner.Seeders)
	}
	if winner.Added != 1700000000 {
		t.Errorf("expected Added timestamp 1700000000 to be preserved, got %d", winner.Added)
	}

	// Verify both trackers survived in the winner's magnet
	trackers := search.ExtractTrackers(winner.Magnet)
	hasTPB := false
	has1337x := false
	for _, tr := range trackers {
		if tr == "udp://tracker.tpb.org:6969" {
			hasTPB = true
		}
		if tr == "udp://tracker.1337x.org:1337" {
			has1337x = true
		}
	}
	if !hasTPB || !has1337x {
		t.Errorf("expected both trackers in magnet, got: %v (magnet: %s)", trackers, winner.Magnet)
	}

	// Verify second item is the untouched unique torrent
	if deduped[1].InfoHash != otherHash {
		t.Errorf("expected distinct infohash %s, got %s", otherHash, deduped[1].InfoHash)
	}
}
