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

	"github.com/x-name15/gorrent/pkg/config"
	"github.com/x-name15/gorrent/pkg/search"
)

var DaemonURL = "http://localhost:7800" // default

func init() {
	if cfg, err := config.Load("config.json"); err == nil {
		DaemonURL = fmt.Sprintf("http://localhost:%d", cfg.Daemon.Port)
	}
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
	case "status":
		handleStatus()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`Usage: gorrent <command> [args]

Commands:
  search <query>          Search for torrents
  download --auto <query> Auto-search and download the best match
  download <magnet>       Download a specific magnet link
  status                  Show active downloads`)
}

func handleSearch(args []string) {
	if len(args) == 0 {
		fmt.Println("Error: missing query")
		return
	}
	query := args[0]

	u := fmt.Sprintf("%s/api/search?q=%s", DaemonURL, url.QueryEscape(query))
	resp, err := http.Get(u)
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
	downloadCmd.Parse(args)

	payload := map[string]string{}

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
	resp, err := http.Post(fmt.Sprintf("%s/api/download", DaemonURL), "application/json", bytes.NewBuffer(body))
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

func handleStatus() {
	resp, err := http.Get(fmt.Sprintf("%s/api/status", DaemonURL))
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
	}
}
