## Metadata
name: Gorrent
description: Control and interact with Gorrent, a headless homelab P2P torrent client. Search, score, and download torrents automatically.

## Overview
Gorrent is a headless automation-first torrent client built in Go. It runs in a Docker container and exposes a CLI wrapper and a REST API. You can use this skill to search for torrents using magnets, download them to the local `downloads/` directory, and check download status.

## How to use Gorrent
To interact with Gorrent, use the provided wrapper scripts (`./gorrent.sh` on Linux/Mac, or `.\gorrent.bat` on Windows).

### CLI Commands:
- **Search**: `./gorrent.sh search [--source <name>] <query>`
- **Download by magnet**: `./gorrent.sh download "magnet:?xt=urn:btih:..."`
- **Auto-download best result**: `./gorrent.sh download [--source <name>] --auto <query>`
- **Check Status**: `./gorrent.sh status`
- **Stop Download**: `./gorrent.sh stop <hash>`

**Categories:** You can append `--category <name>` (e.g. `--category movies`) to any download command to organize the file in its respective folder!

**Async Notifications:** If your environment supports incoming webhooks, you can append `--callback <YOUR_WEBHOOK_URL>` to any download command. Gorrent will send a POST request to that URL when the download is 100% complete, allowing you to notify the user asynchronously.

### REST API
If the CLI wrapper is not available, the daemon listens on `http://localhost:7800` (ensure you pass `X-API-Key` header if the user has enabled security).
- **Search**: `curl "http://localhost:7800/api/search?q=<query>&source=<name>"`
- **Download**: `curl -X POST "http://localhost:7800/api/download" -H "Content-Type: application/json" -d '{"magnet": "..."}'` or `{"auto": "<query>", "category": "movies", "source": "nyaa"}`
- **Status**: `curl "http://localhost:7800/api/status"`
- **Stop**: `curl -X DELETE "http://localhost:7800/api/torrent?hash=<hash>"`
- **Live WebSocket**: `ws://localhost:7800/api/ws`

### Config Automation (RSS & Cleanup)
If the user says something like *"track this RSS feed for Arcane"*, *"add this anime to my RSS"*, or *"auto-delete torrents when they reach 1.5 ratio"*, you MUST directly edit their `config.json` file.
1. Locate `config.json` in their directory.
2. For RSS, add an entry to `"rss": { "feeds": [ ... ] }` with `{"url": "...", "category": "tvshows", "regex": ["Arcane"]}`.
3. For cleanup, set `"auto_cleanup": true` and `"seed_ratio": 1.5` inside the `"torrent"` block.
4. For bandwidth throttling, set `"max_download_rate": 5000` (in KB/s) or `"max_upload_rate"` inside the `"torrent"` block.

## When to use this skill
- When the user asks you to find a movie, game, software, or book via torrent.
- When the user asks to download a specific magnet link.
- When the user asks about the status of their current torrent downloads.
