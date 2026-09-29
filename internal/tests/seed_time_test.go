package tests

import (
	"strings"
	"testing"
	"time"

	"github.com/x-name15/gorrent/pkg/config"
	"github.com/x-name15/gorrent/pkg/torrent"
)

func TestParseSeedDuration(t *testing.T) {
	tests := []struct {
		input       string
		expectedDur time.Duration
		isInfinite  bool
		expectError bool
	}{
		{"0", 0, true, false},
		{"", 0, false, false},
		{"30d", 30 * 24 * time.Hour, false, false},
		{"1d", 24 * time.Hour, false, false},
		{"2h", 2 * time.Hour, false, false},
		{"90m", 90 * time.Minute, false, false},
		{"300s", 300 * time.Second, false, false},
		{"3600", 3600 * time.Second, false, false}, // bare seconds
		{"60", 60 * time.Second, false, false},
		{"-5h", 0, false, true},
		{"-10", 0, false, true},
		{"invalid", 0, false, true},
	}

	for _, tc := range tests {
		dur, isInf, err := torrent.ParseSeedDuration(tc.input)
		if tc.expectError {
			if err == nil {
				t.Errorf("ParseSeedDuration(%q) expected error, got nil", tc.input)
			}
			continue
		}

		if err != nil {
			t.Errorf("ParseSeedDuration(%q) unexpected error: %v", tc.input, err)
			continue
		}

		if isInf != tc.isInfinite {
			t.Errorf("ParseSeedDuration(%q) isInfinite = %v, expected %v", tc.input, isInf, tc.isInfinite)
		}

		if dur != tc.expectedDur {
			t.Errorf("ParseSeedDuration(%q) duration = %v, expected %v", tc.input, dur, tc.expectedDur)
		}
	}
}

func TestShouldDropTorrent_Logic(t *testing.T) {
	now := time.Now()
	completedOneHourAgo := now.Add(-1 * time.Hour)
	completedThreeHoursAgo := now.Add(-3 * time.Hour)
	completedFortyDaysAgo := now.Add(-40 * 24 * time.Hour)

	totalLen := int64(100 * 1024 * 1024)

	cfg := &config.TorrentConfig{
		AutoCleanup: false,
		SeedRatio:   2.0,
		MaxSeedDays: 30,
	}

	// Case 1: Incomplete download should never drop
	drop, _ := torrent.ShouldDropTorrent(totalLen, totalLen/2, totalLen*5, torrent.PersistedTorrent{SeedTime: "1m"}, cfg, completedThreeHoursAgo, now)
	if drop {
		t.Errorf("incomplete download should never drop")
	}

	// Case 2: SeedTime = "0" (infinite) should never drop
	drop, _ = torrent.ShouldDropTorrent(totalLen, totalLen, totalLen*10, torrent.PersistedTorrent{SeedTime: "0"}, cfg, completedFortyDaysAgo, now)
	if drop {
		t.Errorf("SeedTime '0' should never drop")
	}

	// Case 3: Per-torrent seed limit "2h"
	// Completed 1 hour ago (not expired yet)
	drop, _ = torrent.ShouldDropTorrent(totalLen, totalLen, 0, torrent.PersistedTorrent{SeedTime: "2h"}, cfg, completedOneHourAgo, now)
	if drop {
		t.Errorf("SeedTime '2h' with 1h elapsed should not drop")
	}

	// Completed 3 hours ago (expired)
	drop, reason := torrent.ShouldDropTorrent(totalLen, totalLen, 0, torrent.PersistedTorrent{SeedTime: "2h"}, cfg, completedThreeHoursAgo, now)
	if !drop {
		t.Errorf("SeedTime '2h' with 3h elapsed should drop")
	}
	if !strings.Contains(reason, "2h reached") {
		t.Errorf("unexpected reason: %s", reason)
	}

	// Case 4: No per-torrent override, AutoCleanup = false -> should NOT drop
	drop, _ = torrent.ShouldDropTorrent(totalLen, totalLen, totalLen*5, torrent.PersistedTorrent{}, cfg, completedFortyDaysAgo, now)
	if drop {
		t.Errorf("AutoCleanup = false with no per-torrent limit should not drop")
	}

	// Case 5: AutoCleanup = true
	cfgAuto := &config.TorrentConfig{
		AutoCleanup: true,
		SeedRatio:   2.0,
		MaxSeedDays: 30,
	}

	// Ratio reached (wrote 2.5x length)
	drop, reason = torrent.ShouldDropTorrent(totalLen, totalLen, int64(float64(totalLen)*2.5), torrent.PersistedTorrent{}, cfgAuto, completedOneHourAgo, now)
	if !drop {
		t.Errorf("AutoCleanup should drop when ratio target is reached")
	}
	if !strings.Contains(reason, "ratio") {
		t.Errorf("expected ratio reason, got: %s", reason)
	}

	// MaxSeedDays reached (completed 40 days ago, max 30)
	drop, reason = torrent.ShouldDropTorrent(totalLen, totalLen, 0, torrent.PersistedTorrent{}, cfgAuto, completedFortyDaysAgo, now)
	if !drop {
		t.Errorf("AutoCleanup should drop when MaxSeedDays is reached")
	}
	if !strings.Contains(reason, "seeded for") {
		t.Errorf("expected seeded days reason, got: %s", reason)
	}
}
