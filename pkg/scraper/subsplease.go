package scraper

import (
	"github.com/x-name15/gorrent/pkg/search"

	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type subsplease struct {
	client *http.Client
}

func NewSubsPlease() search.Source {
	return &subsplease{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (s *subsplease) InjectClient(c *http.Client) {
	s.client = c
}

func (s *subsplease) ID() string {
	return "subsplease"
}

func (s *subsplease) Name() string {
	return "SubsPlease"
}

type spDownload struct {
	Res    string `json:"res"`
	Magnet string `json:"magnet"`
}

type spEntry struct {
	Show        string       `json:"show"`
	Episode     string       `json:"episode"`
	ReleaseDate string       `json:"release_date"`
	Downloads   []spDownload `json:"downloads"`
}

func (s *subsplease) Search(ctx context.Context, query string) ([]search.TorrentResult, error) {
	q := strings.TrimSpace(query)
	params := url.Values{}
	params.Set("tz", "UTC")
	if q != "" {
		params.Set("f", "search")
		params.Set("s", q)
	} else {
		params.Set("f", "latest")
	}

	u := fmt.Sprintf("https://subsplease.org/api/?%s", params.Encode())
	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "gorrent/1.0")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("SubsPlease returned %d", resp.StatusCode)
	}

	var jsonRaw map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&jsonRaw); err != nil {
		return nil, err // Might be an array if empty
	}

	var out []search.TorrentResult
	for _, entryRaw := range jsonRaw {
		entryMap, ok := entryRaw.(map[string]interface{})
		if !ok {
			continue
		}

		b, _ := json.Marshal(entryMap)
		var entry spEntry
		json.Unmarshal(b, &entry)

		var bestDl *spDownload
		for _, res := range []string{"1080", "720", "480"} {
			for _, d := range entry.Downloads {
				if d.Res == res && d.Magnet != "" {
					dCopy := d
					bestDl = &dCopy
					break
				}
			}
			if bestDl != nil {
				break
			}
		}
		if bestDl == nil {
			for _, d := range entry.Downloads {
				if d.Magnet != "" {
					dCopy := d
					bestDl = &dCopy
					break
				}
			}
		}

		if bestDl == nil || bestDl.Magnet == "" {
			continue
		}

		m := regexp.MustCompile(`urn:btih:([a-zA-Z0-9]+)`).FindStringSubmatch(bestDl.Magnet)
		if len(m) < 2 {
			continue
		}
		infoHash := strings.ToLower(m[1])

		show := entry.Show
		if show == "" {
			show = "Unknown"
		}
		ep := ""
		if entry.Episode != "" {
			ep = " - " + entry.Episode
		}

		sizeBytes := int64(0)
		sizeM := regexp.MustCompile(`[?&]xl=(\d+)`).FindStringSubmatch(bestDl.Magnet)
		if len(sizeM) > 1 {
			sizeBytes, _ = strconv.ParseInt(sizeM[1], 10, 64)
		}

		out = append(out, search.TorrentResult{
			InfoHash:  infoHash,
			Name:      fmt.Sprintf("%s%s [%sp]", show, ep, bestDl.Res),
			SizeBytes: sizeBytes,
			Seeders:   0,
			Leechers:  0,
			Source:    "subsplease",
			Magnet:    bestDl.Magnet,
		})
	}
	return out, nil
}
