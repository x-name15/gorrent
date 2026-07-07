## Metadata
name: Gorrent
description: Control and interact with Gorrent, a headless homelab P2P torrent client. Search, score, and download torrents automatically.

## Overview
Gorrent is a headless automation-first torrent client built in Go. It runs in a Docker container and exposes a CLI wrapper and a REST API. You can use this skill to search for torrents using magnets, download them to the local `downloads/` directory, and check download status.

## How to use Gorrent
To interact with Gorrent, use the provided wrapper scripts (`./gorrent.sh` on Linux/Mac, or `.\gorrent.bat` on Windows).

### CLI Commands:
- **Search**: `./gorrent.sh search <query>`
- **Download by magnet**: `./gorrent.sh download "magnet:?xt=urn:btih:..."`
- **Auto-download best result**: `./gorrent.sh download --auto <query>`
- **Check Status**: `./gorrent.sh status`

**Async Notifications:** If your environment supports incoming webhooks, you can append `--callback <YOUR_WEBHOOK_URL>` to any download command. Gorrent will send a POST request to that URL when the download is 100% complete, allowing you to notify the user asynchronously.

### REST API
If the CLI wrapper is not available or you prefer HTTP requests, the daemon listens on `http://localhost:7800`.
- **Search**: `curl "http://localhost:7800/api/search?q=<query>"`
- **Download**: `curl -X POST "http://localhost:7800/api/download" -H "Content-Type: application/json" -d '{"magnet": "..."}'` or `{"auto": "<query>"}`
- **Status**: `curl "http://localhost:7800/api/status"`

## When to use this skill
- When the user asks you to find a movie, game, software, or book via torrent.
- When the user asks to download a specific magnet link.
- When the user asks about the status of their current torrent downloads.
