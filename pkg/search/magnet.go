package search

import (
	"encoding/base32"
	"encoding/hex"
	"net/url"
	"regexp"
	"strings"
)

var (
	magnetBTIHRegex = regexp.MustCompile(`(?i)xt=urn:btih:([a-f0-9]{40}|[a-z2-7]{32})`)
	bareHashRegex   = regexp.MustCompile(`(?i)^([a-f0-9]{40}|[a-z2-7]{32})$`)
)

// NormalizeInfoHash converts a 32-character Base32 infohash into a 40-character lowercase Hex infohash.
// If the input is already a 40-character hex string, it is returned in lowercase.
func NormalizeInfoHash(raw string) string {
	s := strings.TrimSpace(raw)
	if len(s) == 32 {
		// BitTorrent standard uses unpadded RFC 4648 Base32
		decoded, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(s))
		if err == nil && len(decoded) == 20 {
			return hex.EncodeToString(decoded)
		}
	}
	return strings.ToLower(s)
}

// ExtractInfoHash extracts and normalizes the infohash from a magnet URI or bare hash string.
func ExtractInfoHash(input string) string {
	s := strings.TrimSpace(input)
	if m := magnetBTIHRegex.FindStringSubmatch(s); len(m) > 1 {
		return NormalizeInfoHash(m[1])
	}
	if bareHashRegex.MatchString(s) {
		return NormalizeInfoHash(s)
	}
	return ""
}

// ExtractTrackers parses all "tr" query parameters from a magnet URI.
func ExtractTrackers(magnet string) []string {
	s := strings.TrimSpace(magnet)
	if !strings.HasPrefix(strings.ToLower(s), "magnet:?") {
		return nil
	}
	u, err := url.Parse(s)
	if err != nil {
		return nil
	}
	var trackers []string
	seen := make(map[string]bool)
	for _, tr := range u.Query()["tr"] {
		tr = strings.TrimSpace(tr)
		if tr != "" && !seen[tr] {
			seen[tr] = true
			trackers = append(trackers, tr)
		}
	}
	return trackers
}

// MergeMagnetTrackers preserves the primary magnet URI byte-for-byte and appends
// any unique trackers present in the other magnet links that are missing from primary.
func MergeMagnetTrackers(primary string, others ...string) string {
	primary = strings.TrimSpace(primary)
	if primary == "" {
		for _, o := range others {
			if strings.TrimSpace(o) != "" {
				return o
			}
		}
		return ""
	}

	ownTrackers := ExtractTrackers(primary)
	seen := make(map[string]bool)
	for _, t := range ownTrackers {
		seen[t] = true
	}

	var toAdd []string
	for _, other := range others {
		for _, tr := range ExtractTrackers(other) {
			if !seen[tr] {
				seen[tr] = true
				toAdd = append(toAdd, tr)
			}
		}
	}

	if len(toAdd) == 0 {
		return primary
	}

	var sb strings.Builder
	sb.WriteString(primary)
	for _, tr := range toAdd {
		sb.WriteString("&tr=")
		sb.WriteString(url.QueryEscape(tr))
	}
	return sb.String()
}
