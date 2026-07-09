package rss

import (
	"encoding/json"
	"encoding/xml"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/x-name15/gorrent/pkg/config"
	"github.com/x-name15/gorrent/pkg/torrent"
)

// RSS represents a standard RSS feed document.
// Handles both standard <link> and <enclosure url="magnet:..."> patterns,
// as well as the <torrent:magnetURI> extension used by some feeds.
type RSS struct {
	Channel struct {
		Items []struct {
			Title     string `xml:"title"`
			Link      string `xml:"link"`
			Enclosure struct {
				URL  string `xml:"url,attr"`
				Type string `xml:"type,attr"`
			} `xml:"enclosure"`
			MagnetURI string `xml:"magnetURI"` // torrent: namespace extension (e.g. Nyaa)
		} `xml:"item"`
	} `xml:"channel"`
}

type Manager struct {
	cfg     *config.RSSConfig
	dataDir string
	client  *torrent.Client
	history map[string]bool
}

func NewManager(cfg *config.Config, client *torrent.Client) *Manager {
	m := &Manager{
		cfg:     &cfg.RSS,
		dataDir: cfg.Daemon.DataDir,
		client:  client,
		history: make(map[string]bool),
	}
	m.loadHistory()
	return m
}

func (m *Manager) Start() {
	if m.cfg.IntervalMin <= 0 || len(m.cfg.Feeds) == 0 {
		return
	}

	log.Printf("Starting RSS Auto-Downloader (polling every %d minutes for %d feeds)", m.cfg.IntervalMin, len(m.cfg.Feeds))
	m.poll() // initial poll on startup
	for {
		time.Sleep(time.Duration(m.cfg.IntervalMin) * time.Minute)
		m.poll()
	}
}

func (m *Manager) poll() {
	for _, feed := range m.cfg.Feeds {
		m.processFeed(feed)
	}
}

func (m *Manager) processFeed(feed config.RSSFeed) {
	resp, err := http.Get(feed.URL)
	if err != nil {
		log.Printf("RSS Error fetching %s: %v", feed.URL, err)
		return
	}
	defer resp.Body.Close()

	var doc RSS
	if err := xml.NewDecoder(resp.Body).Decode(&doc); err != nil {
		log.Printf("RSS Error parsing %s: %v", feed.URL, err)
		return
	}

	// Compile regexes once per feed, not per item
	var compiledRegexes []*regexp.Regexp
	for _, pattern := range feed.Regex {
		if re, err := regexp.Compile("(?i)" + pattern); err == nil {
			compiledRegexes = append(compiledRegexes, re)
		} else {
			log.Printf("RSS Error compiling regex '%s': %v", pattern, err)
		}
	}

	changed := false
	for _, item := range doc.Channel.Items {
		// Resolve the actual magnet link from the item.
		// Priority: torrent:magnetURI > enclosure[magnet:] > link[magnet:]
		magnet := resolveMagnet(item.Link, item.Enclosure.URL, item.MagnetURI)
		if magnet == "" {
			log.Printf("RSS Skipping '%s': no magnet link found in item", item.Title)
			continue
		}

		// Use the magnet as the history key (stable and unique)
		if m.history[magnet] {
			continue // Already downloaded
		}

		match := len(compiledRegexes) == 0 // No regex = download everything
		for _, re := range compiledRegexes {
			if re.MatchString(item.Title) {
				match = true
				break
			}
		}

		if match {
			log.Printf("RSS Match found: %s", item.Title)
			_, err := m.client.AddMagnet(magnet, feed.Category)
			if err != nil {
				log.Printf("RSS Failed to add magnet for %s: %v", item.Title, err)
			} else {
				m.history[magnet] = true
				changed = true
			}
		}
	}

	// Save history once per feed poll, not once per matched item
	if changed {
		m.saveHistory()
	}
}

// resolveMagnet picks the best magnet URI from the available RSS item fields.
func resolveMagnet(link, enclosureURL, magnetURI string) string {
	if magnetURI != "" {
		return magnetURI
	}
	if strings.HasPrefix(enclosureURL, "magnet:") {
		return enclosureURL
	}
	if strings.HasPrefix(link, "magnet:") {
		return link
	}
	return ""
}

func (m *Manager) historyFile() string {
	return filepath.Join(m.dataDir, "rss_history.json")
}

func (m *Manager) loadHistory() {
	os.MkdirAll(m.dataDir, 0755)
	b, err := os.ReadFile(m.historyFile())
	if err == nil {
		json.Unmarshal(b, &m.history)
	}
}

func (m *Manager) saveHistory() {
	b, _ := json.MarshalIndent(m.history, "", "  ")
	os.WriteFile(m.historyFile(), b, 0644)
}
