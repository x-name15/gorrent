package main

import (
	"github.com/x-name15/gorrent/pkg/search"

	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"time"

	"github.com/gorilla/websocket"
	"github.com/x-name15/gorrent/pkg/config"
	"github.com/x-name15/gorrent/pkg/netutil"
	"github.com/x-name15/gorrent/pkg/scraper"
	"github.com/x-name15/gorrent/pkg/torrent"
)

//go:embed openapi.yaml
var openapiYAML []byte

type Server struct {
	cfg        *config.Config
	scraperMgr *search.Manager
	torrentCli *torrent.Client
}

func main() {
	cfg, err := config.Load("config.json")
	if err != nil {
		log.Println("Could not load config.json, using defaults:", err)
		cfg = config.Default()
	}

	torrentCli, err := torrent.NewClient(&cfg.Torrent)
	if err != nil {
		log.Fatal("Failed to initialize torrent client:", err)
	}
	defer torrentCli.Close()

	// Enable DoH for all default http clients
	if cfg.Scraper.DNS != "" {
		http.DefaultTransport = netutil.NewDoHTransport(cfg.Scraper.DNS)
		log.Printf("Enabled DNS-over-HTTPS using resolver: %s\n", cfg.Scraper.DNS)
	}

	scraperMgr := search.NewManager(&cfg.Scraper)

	cacheTTL := 5 * time.Minute
	wrap := func(s search.Source) search.Source {
		return search.NewCircuitBreakerSource(search.NewCachingSource(s, cacheTTL))
	}

	scraperMgr.Register(wrap(scraper.NewYTS()))
	scraperMgr.Register(wrap(scraper.NewNyaa()))
	scraperMgr.Register(wrap(scraper.NewPirateBay()))
	scraperMgr.Register(wrap(scraper.New1337x()))
	scraperMgr.Register(wrap(scraper.NewEZTV()))
	scraperMgr.Register(wrap(scraper.NewSubsPlease()))
	scraperMgr.Register(wrap(scraper.NewFitGirl()))
	scraperMgr.Register(wrap(scraper.NewTorrentsCSV()))

	if cfg.Scraper.RutrackerCookie != "" {
		scraperMgr.Register(wrap(scraper.NewRuTracker(cfg.Scraper.RutrackerCookie)))
		log.Println("RuTracker source activated (cookie provided)")
	}

	srv := &Server{
		cfg:        cfg,
		scraperMgr: scraperMgr,
		torrentCli: torrentCli,
	}

	http.HandleFunc("/api/search", srv.authMiddleware(srv.handleSearch))
	http.HandleFunc("/api/download", srv.authMiddleware(srv.handleDownload))
	http.HandleFunc("/api/status", srv.authMiddleware(srv.handleStatus))
	http.HandleFunc("/api/torrent", srv.authMiddleware(srv.handleStop))
	http.HandleFunc("/api/ws", srv.authMiddleware(srv.handleWS))
	http.HandleFunc("/health", srv.handleHealth)
	http.HandleFunc("/metrics", srv.handleMetrics)
	http.HandleFunc("/api/docs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/yaml")
		w.Write(openapiYAML)
	})

	addr := fmt.Sprintf(":%d", cfg.Daemon.Port)
	log.Printf("Gorrent Daemon listening on %s...", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}

func (s *Server) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.cfg.Daemon.APIKey != "" {
			key := r.Header.Get("X-API-Key")
			if key != s.cfg.Daemon.APIKey && r.URL.Query().Get("apikey") != s.cfg.Daemon.APIKey {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}
		next(w, r)
	}
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Missing query parameter 'q'", http.StatusBadRequest)
		return
	}

	results := s.scraperMgr.Search(r.Context(), query)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func (s *Server) handleDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Magnet   string `json:"magnet"`
		Auto     string `json:"auto"` // query to auto-download best match
		Callback string `json:"callback"`
		Category string `json:"category"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	magnetToDownload := req.Magnet

	if req.Auto != "" {
		// Auto mode: Search and pick best
		results := s.scraperMgr.Search(r.Context(), req.Auto)
		if len(results) == 0 {
			http.Error(w, "No results found for auto-download", http.StatusNotFound)
			return
		}
		magnetToDownload = results[0].Magnet // Pick highest scored
	}

	if magnetToDownload == "" {
		http.Error(w, "Missing magnet or auto parameter", http.StatusBadRequest)
		return
	}

	// 1. Accept bare infohashes
	if len(magnetToDownload) == 40 && regexp.MustCompile(`^[a-fA-F0-9]{40}$`).MatchString(magnetToDownload) {
		magnetToDownload = "magnet:?xt=urn:btih:" + magnetToDownload
	}

	// 2. Add custom trackers
	if len(s.cfg.Torrent.Trackers) > 0 {
		for _, tr := range s.cfg.Torrent.Trackers {
			magnetToDownload += "&tr=" + url.QueryEscape(tr)
		}
	}

	go func() {
		t, err := s.torrentCli.AddMagnet(magnetToDownload, req.Category)
		if err != nil {
			log.Printf("Failed to add magnet: %v", err)
			return
		}

		if req.Callback != "" {
			// Wait for metadata
			<-t.GotInfo()
			info := t.Info()
			if info == nil {
				return
			}

			// Wait for download to finish
			for {
				if t.BytesCompleted() >= info.TotalLength() {
					payload := map[string]string{
						"event": "completed",
						"name":  info.Name,
						"hash":  t.InfoHash().HexString(),
					}
					b, _ := json.Marshal(payload)
					http.Post(req.Callback, "application/json", bytes.NewBuffer(b))
					log.Printf("Triggered callback for %s", info.Name)
					break
				}
				time.Sleep(10 * time.Second)
			}
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "started", "magnet": magnetToDownload})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats := s.torrentCli.Status()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (s *Server) handleStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	hash := r.URL.Query().Get("hash")
	if hash == "" {
		http.Error(w, "Missing hash parameter", http.StatusBadRequest)
		return
	}

	if err := s.torrentCli.StopTorrent(hash); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "stopped", "hash": hash})
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WS upgrade error: %v", err)
		return
	}
	defer c.Close()

	for {
		stats := s.torrentCli.Status()
		if err := c.WriteJSON(stats); err != nil {
			break
		}
		time.Sleep(1 * time.Second)
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	stats := s.torrentCli.Status()
	connStats := s.torrentCli.ConnStats()

	fmt.Fprintf(w, "# HELP gorrent_torrents_active Number of active torrents\n")
	fmt.Fprintf(w, "# TYPE gorrent_torrents_active gauge\n")
	fmt.Fprintf(w, "gorrent_torrents_active %d\n", len(stats))

	fmt.Fprintf(w, "# HELP gorrent_bytes_downloaded Total bytes downloaded\n")
	fmt.Fprintf(w, "# TYPE gorrent_bytes_downloaded counter\n")
	fmt.Fprintf(w, "gorrent_bytes_downloaded %d\n", connStats.BytesReadData.Int64())

	fmt.Fprintf(w, "# HELP gorrent_bytes_uploaded Total bytes uploaded\n")
	fmt.Fprintf(w, "# TYPE gorrent_bytes_uploaded counter\n")
	fmt.Fprintf(w, "gorrent_bytes_uploaded %d\n", connStats.BytesWrittenData.Int64())
}
