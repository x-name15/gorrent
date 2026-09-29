package tests

import (
	"os"
	"testing"

	"github.com/x-name15/gorrent/pkg/config"
	"github.com/x-name15/gorrent/pkg/torrent"
)

func TestUTP_DisabledByConfig(t *testing.T) {
	// Ensure env var is clean
	os.Unsetenv("GORRENT_NO_UTP")

	cfgEnabled := &config.TorrentConfig{
		DisableUTP: false,
	}
	if torrent.IsUTPDisabled(cfgEnabled) {
		t.Errorf("expected uTP to be enabled when DisableUTP is false and env unset")
	}

	cfgDisabled := &config.TorrentConfig{
		DisableUTP: true,
	}
	if !torrent.IsUTPDisabled(cfgDisabled) {
		t.Errorf("expected uTP to be disabled when DisableUTP is true")
	}
}

func TestUTP_DisabledByEnv(t *testing.T) {
	cfg := &config.TorrentConfig{
		DisableUTP: false,
	}

	// Test with GORRENT_NO_UTP=1
	t.Setenv("GORRENT_NO_UTP", "1")
	if !torrent.IsUTPDisabled(cfg) {
		t.Errorf("expected uTP to be disabled when GORRENT_NO_UTP=1")
	}

	// Test with GORRENT_NO_UTP=true
	t.Setenv("GORRENT_NO_UTP", "true")
	if !torrent.IsUTPDisabled(cfg) {
		t.Errorf("expected uTP to be disabled when GORRENT_NO_UTP=true")
	}

	// Test with nil config but env set
	if !torrent.IsUTPDisabled(nil) {
		t.Errorf("expected uTP to be disabled when GORRENT_NO_UTP is set even if config is nil")
	}
}
