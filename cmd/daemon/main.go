package main

import (
	"github.com/x-name15/gorrent/pkg/search"

	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

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

	http.HandleFunc("/api/search", srv.handleSearch)
	http.HandleFunc("/api/download", srv.handleDownload)
	http.HandleFunc("/api/status", srv.handleStatus)
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
		Magnet string `json:"magnet"`
		Auto   string `json:"auto"` // query to auto-download best match
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

	go func() {
		_, err := s.torrentCli.AddMagnet(magnetToDownload)
		if err != nil {
			log.Printf("Failed to add magnet: %v", err)
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
