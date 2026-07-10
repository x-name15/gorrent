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
	case "status":
		handleStatus()
	case "stop":
		handleStop(os.Args[2:])
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`Usage: gorrent <command> [args]

Commands:
  search [--source <name>] <query>          Search for torrents
  download [--source <name>] --auto <query> Auto-search and download the best match
  download <magnet>                         Download a specific magnet link
  status                  Show active downloads
  stop <hash>             Stop and delete an active download`)
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
