package rss

import (
	"encoding/json"
	"encoding/xml"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/x-name15/gorrent/pkg/config"
	"github.com/x-name15/gorrent/pkg/torrent"
)

type RSS struct {
	Channel struct {
		Items []struct {
			Title string `xml:"title"`
			Link  string `xml:"link"`
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
	m.poll() // initial poll
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

	var compiledRegexes []*regexp.Regexp
	for _, pattern := range feed.Regex {
		if re, err := regexp.Compile("(?i)" + pattern); err == nil {
			compiledRegexes = append(compiledRegexes, re)
		} else {
			log.Printf("RSS Error compiling regex '%s': %v", pattern, err)
		}
	}

	for _, item := range doc.Channel.Items {
		if m.history[item.Link] {
			continue // Already downloaded
		}

		match := false
		if len(compiledRegexes) == 0 {
			match = true // No regex means download everything
		} else {
			for _, re := range compiledRegexes {
				if re.MatchString(item.Title) {
					match = true
					break
				}
			}
		}

		if match {
			log.Printf("RSS Match found: %s", item.Title)
			_, err := m.client.AddMagnet(item.Link, feed.Category)
			if err != nil {
				log.Printf("RSS Failed to add magnet for %s: %v", item.Title, err)
			} else {
				m.history[item.Link] = true
				m.saveHistory()
			}
		}
	}
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
