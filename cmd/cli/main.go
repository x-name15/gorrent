package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/x-name15/gorrent/pkg/config"
	"github.com/x-name15/gorrent/pkg/search"
)

var DaemonURL string
var APIKey string

func init() {
	cfg, err := config.Load("config.yaml")
	if err != nil {
		cfg = config.Default()
	}
	DaemonURL = fmt.Sprintf("http://localhost:%d", cfg.Daemon.Port)
	APIKey = cfg.Daemon.APIKey
}

func doRequest(req *http.Request) (*http.Response, error) {
	if APIKey != "" {
		req.Header.Set("X-API-Key", APIKey)
	}
	client := &http.Client{}
	return client.Do(req)
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "search":
		handleSearch(os.Args[2:])
	case "download":
		handleDownload(os.Args[2:])
	case "seed":
		handleSeed(os.Args[2:])
	case "status":
		handleStatus()
	case "stop":
		handleStop(os.Args[2:])
	case "seed-time":
		handleSeedTime(os.Args[2:])
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`Usage: gorrent <command> [args]

Commands:
  search [--source <name>] <query>                              Search for torrents
  download [--source <name>] [--seed-time <dur>] --auto <query> Auto-search and download
  download [--seed-time <dur>] <magnet>                         Download a specific magnet link
  seed [--category <name>] [--seed-time <dur>] <path>           Turn folder/file into torrent and seed it
  seed-time <hash> <duration>                                   Set custom seed limit (e.g. 30d, 2h, 0 = infinite)
  status                                                        Show active downloads
  stop <hash>                                                   Stop and delete an active download`)
}

func handleSearch(args []string) {
	searchCmd := flag.NewFlagSet("search", flag.ExitOnError)
	sourceFlag := searchCmd.String("source", "", "Specific source to search (e.g. nyaa, yts)")
	searchCmd.Parse(args)

	if len(searchCmd.Args()) == 0 {
		fmt.Println("Error: missing query")
		return
	}
	query := searchCmd.Args()[0]

	u := fmt.Sprintf("%s/api/search?q=%s", DaemonURL, url.QueryEscape(query))
	if *sourceFlag != "" {
		u += "&source=" + url.QueryEscape(*sourceFlag)
	}
	req, _ := http.NewRequest(http.MethodGet, u, nil)
	resp, err := doRequest(req)
	if err != nil {
		log.Fatal("Failed to connect to daemon:", err)
	}
	defer resp.Body.Close()

	var results []search.TorrentResult
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		log.Fatal("Failed to parse response:", err)
	}

	if len(results) == 0 {
		fmt.Println("No results found.")
		return
	}

	fmt.Printf("Found %d results:\n\n", len(results))
	for i, r := range results {
		fmt.Printf("[%d] %s\n", i+1, r.Name)
		fmt.Printf("    Size: %d MB | Seeders: %d | Source: %s | Score: %d\n", r.SizeBytes/1024/1024, r.Seeders, r.Source, r.Score)
		fmt.Printf("    Magnet: %s\n\n", r.Magnet[:80]+"...") // truncate magnet for display
	}
}

func handleDownload(args []string) {
	downloadCmd := flag.NewFlagSet("download", flag.ExitOnError)
	autoFlag := downloadCmd.String("auto", "", "Auto-search and download best match")
	sourceFlag := downloadCmd.String("source", "", "Specific source to search (e.g. nyaa, yts)")
	callbackFlag := downloadCmd.String("callback", "", "Webhook URL to notify upon completion")
	categoryFlag := downloadCmd.String("category", "", "Save torrent to a specific category folder")
	seedTimeFlag := downloadCmd.String("seed-time", "", "Custom seed duration (e.g. 30d, 2h, 0 for infinite)")
	downloadCmd.Parse(args)

	payload := map[string]string{}

	if *sourceFlag != "" {
		payload["source"] = *sourceFlag
	}

	if *callbackFlag != "" {
		payload["callback"] = *callbackFlag
	}

	if *categoryFlag != "" {
		payload["category"] = *categoryFlag
	}

	if *seedTimeFlag != "" {
		payload["seed_time"] = *seedTimeFlag
	}

	if *autoFlag != "" {
		payload["auto"] = *autoFlag
		fmt.Printf("Sending auto-download request for: %s\n", *autoFlag)
	} else if len(downloadCmd.Args()) > 0 {
		payload["magnet"] = downloadCmd.Args()[0]
		fmt.Println("Sending direct magnet download request...")
	} else {
		fmt.Println("Error: must provide --auto <query> or a <magnet_link>")
		return
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/download", DaemonURL), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := doRequest(req)
	if err != nil {
		log.Fatal("Failed to connect to daemon:", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		log.Fatalf("Daemon error: %s", string(b))
	}

	fmt.Println("Download started successfully!")
}

func formatDuration(sec int64) string {
	if sec < 0 {
		return "infinite"
	}
	d := time.Duration(sec) * time.Second
	if d >= 24*time.Hour {
		days := d / (24 * time.Hour)
		hours := (d % (24 * time.Hour)) / time.Hour
		return fmt.Sprintf("%dd %dh remaining", days, hours)
	}
	if d >= time.Hour {
		hours := d / time.Hour
		mins := (d % time.Hour) / time.Minute
		return fmt.Sprintf("%dh %dm remaining", hours, mins)
	}
	if d >= time.Minute {
		mins := d / time.Minute
		secs := (d % time.Minute) / time.Second
		return fmt.Sprintf("%dm %ds remaining", mins, secs)
	}
	return fmt.Sprintf("%ds remaining", sec)
}

func handleStatus() {
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/api/status", DaemonURL), nil)
	resp, err := doRequest(req)
	if err != nil {
		log.Fatal("Failed to connect to daemon:", err)
	}
	defer resp.Body.Close()

	var stats []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		log.Fatal("Failed to parse response:", err)
	}

	if len(stats) == 0 {
		fmt.Println("No active downloads.")
		return
	}

	fmt.Println("Active Downloads:")
	for _, s := range stats {
		dl := s["downloaded"].(float64)
		total := s["length"].(float64)
		peers := s["peers"].(float64)
		progress := 0.0
		if total > 0 {
			progress = (dl / total) * 100
		}

		fmt.Printf("- %s\n  Progress: %.1f%% (%.2f / %.2f MB) | Peers: %.0f\n",
			s["name"], progress, dl/1024/1024, total/1024/1024, peers)

		if st, ok := s["seed_time"].(string); ok && st != "" {
			remStr := ""
			if rem, ok := s["seed_remaining_sec"].(float64); ok {
				remStr = " | " + formatDuration(int64(rem))
			}
			fmt.Printf("  Seed Limit: %s%s\n", st, remStr)
		}
	}
}

func handleStop(args []string) {
	if len(args) == 0 {
		fmt.Println("Error: missing hash")
		return
	}
	hash := args[0]

	req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/api/torrent?hash=%s", DaemonURL, hash), nil)
	resp, err := doRequest(req)
	if err != nil {
		log.Fatal("Failed to connect to daemon:", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		log.Fatalf("Daemon error: %s", string(b))
	}

	fmt.Printf("Successfully stopped torrent: %s\n", hash)
}

func handleSeed(args []string) {
	seedCmd := flag.NewFlagSet("seed", flag.ExitOnError)
	categoryFlag := seedCmd.String("category", "", "Optional category for the seeded torrent")
	seedTimeFlag := seedCmd.String("seed-time", "", "Custom seed duration (e.g. 30d, 2h, 0 for infinite)")
	seedCmd.Parse(args)

	if seedCmd.NArg() < 1 {
		fmt.Println("Usage: gorrent seed [--category <name>] [--seed-time <duration>] <path>")
		os.Exit(1)
	}

	targetPath := seedCmd.Arg(0)
	absPath, err := filepath.Abs(targetPath)
	if err != nil {
		log.Fatalf("Invalid path: %v", err)
	}

	payload := map[string]string{
		"path":     absPath,
		"category": *categoryFlag,
	}
	if *seedTimeFlag != "" {
		payload["seed_time"] = *seedTimeFlag
	}

	b, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPost, DaemonURL+"/api/seed", bytes.NewBuffer(b))
	if err != nil {
		log.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := doRequest(req)
	if err != nil {
		log.Fatalf("Failed to connect to daemon: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Fatalf("Daemon error (%d): %s", resp.StatusCode, string(body))
	}

	var res map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&res)

	fmt.Println("Seeding started successfully!")
	fmt.Printf("Name:        %v\n", res["name"])
	fmt.Printf("InfoHash:    %v\n", res["info_hash"])
	fmt.Printf("Total Bytes: %v\n", res["total_bytes"])
	if tf, ok := res["torrent_file"].(string); ok && tf != "" {
		fmt.Printf(".torrent:    %s\n", tf)
	}
	fmt.Printf("Magnet:      %v\n", res["magnet"])
}

func handleSeedTime(args []string) {
	if len(args) < 2 {
		fmt.Println("Usage: gorrent seed-time <hash> <duration>")
		fmt.Println("Example: gorrent seed-time 4a6c8e... 2h")
		fmt.Println("Pass '0' to seed indefinitely, or '' to clear custom limit.")
		return
	}
	hash := args[0]
	duration := args[1]

	payload := map[string]string{
		"hash":      hash,
		"seed_time": duration,
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/api/torrent/seed-time", DaemonURL), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := doRequest(req)
	if err != nil {
		log.Fatal("Failed to connect to daemon:", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		log.Fatalf("Daemon error: %s", string(b))
	}

	fmt.Printf("Successfully updated seed limit for %s to: %s\n", hash, duration)
}
