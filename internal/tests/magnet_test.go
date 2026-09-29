package tests

import (
	"encoding/base32"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/x-name15/gorrent/pkg/search"
)

func TestNormalizeInfoHash(t *testing.T) {
	// 20 raw bytes:
	rawBytes := []byte("12345678901234567890")
	expectedHex := hex.EncodeToString(rawBytes)
	b32 := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(rawBytes)

	if len(b32) != 32 {
		t.Fatalf("expected 32 chars base32, got %d", len(b32))
	}

	norm := search.NormalizeInfoHash(b32)
	if norm != expectedHex {
		t.Fatalf("expected exact hex %s, got %s", expectedHex, norm)
	}

	// 40-char Hex should remain unchanged (lowercased)
	hexInput := strings.ToUpper(expectedHex)
	if search.NormalizeInfoHash(hexInput) != expectedHex {
		t.Fatalf("expected lowercased hex, got %s", search.NormalizeInfoHash(hexInput))
	}
}

func TestExtractInfoHash(t *testing.T) {
	rawBytes := []byte("12345678901234567890")
	expectedHex := hex.EncodeToString(rawBytes)
	b32 := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(rawBytes)

	// 1. Extract and normalize Base32 infohash from magnet
	magnetB32 := "magnet:?xt=urn:btih:" + b32 + "&dn=Test"
	h := search.ExtractInfoHash(magnetB32)
	if h != expectedHex {
		t.Fatalf("expected exact normalized hex %s, got %s", expectedHex, h)
	}

	// 2. Extract standard 40-char hex infohash from magnet
	magnetHex := "magnet:?xt=urn:btih:" + expectedHex + "&dn=Test"
	if search.ExtractInfoHash(magnetHex) != expectedHex {
		t.Fatalf("expected hex preserved, got %s", search.ExtractInfoHash(magnetHex))
	}

	// 3. Extract bare hash input
	if search.ExtractInfoHash(expectedHex) != expectedHex {
		t.Fatalf("expected bare hash extracted")
	}
}

func TestMergeMagnetTrackers(t *testing.T) {
	primary := "magnet:?xt=urn:btih:0123456789abcdef0123456789abcdef01234567&dn=Movie&tr=udp%3A%2F%2Ftracker1.org%3A1337%2Fannounce"
	other := "magnet:?xt=urn:btih:0123456789abcdef0123456789abcdef01234567&dn=Movie&tr=udp%3A%2F%2Ftracker2.org%3A6969%2Fannounce&tr=udp%3A%2F%2Ftracker1.org%3A1337%2Fannounce"

	merged := search.MergeMagnetTrackers(primary, other)
	trackers := search.ExtractTrackers(merged)

	if len(trackers) != 2 {
		t.Fatalf("expected 2 unique trackers, got %d: %v", len(trackers), trackers)
	}

	hasT1 := false
	hasT2 := false
	for _, tr := range trackers {
		if tr == "udp://tracker1.org:1337/announce" {
			hasT1 = true
		}
		if tr == "udp://tracker2.org:6969/announce" {
			hasT2 = true
		}
	}
	if !hasT1 || !hasT2 {
		t.Fatalf("expected both tracker1 and tracker2 in merged list, got %v", trackers)
	}
}
