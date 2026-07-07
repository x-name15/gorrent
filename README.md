# ⛵ Gorrent

[![Release](https://img.shields.io/badge/Release-v1.0.0-green?style=flat-square)](https://github.com/x-name15/gorrent/releases)
[![Go Version](https://img.shields.io/badge/Go-1.25-00ADD8?style=flat-square&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-GPLv3-blue?style=flat-square)](LICENSE)
[![Build Status](https://img.shields.io/github/actions/workflow/status/x-name15/gorrent/entry.yaml?style=flat-square&logo=githubactions&logoColor=white)](https://github.com/x-name15/gorrent/actions)
[![Docker Image Size](https://img.shields.io/badge/Image-%3C30MB-informational?style=flat-square&logo=docker)](https://github.com/x-name15/gorrent/pkgs/container/gorrent)

The original [torlink](https://github.com/baairon/torlink) is an excellent project that deliberately focuses on being an interactive terminal application. Later, a fork of [torlink](https://github.com/WarlaxZ/torlink) expanded upon it by adding features and wider tracker support.

Rather than trying to change their core interactive philosophy, **Gorrent** takes inspiration from both projects and explores a completely different direction: a headless, automation-first implementation written entirely in Go for homelab environments.

## Features

| Feature             | Supported |
| ------------------- | --------- |
| REST API            | ✅         |
| Docker              | ✅         |
| DNS-over-HTTPS (DoH)| ✅         |
| Concurrent scraping | ✅         |
| Auto scoring        | ✅         |
| RuTracker           | ✅         |
| Optional CLI        | ✅         |
| OpenClaw            | ✅         |
| Claude Skill        | ✅         |

## Fault-Tolerant Architecture

Gorrent is built from the ground up to be a headless daemon.

### Flow Overview

```text
            REST API
               │
               ▼
         Search Request
               │
               ▼
      Concurrent Scrapers
    ┌────┬────┬────┬────┐
    │YTS │1337│Nyaa│... │
    └────┴────┴────┴────┘
               │
               ▼
        Scoring Engine
               │
               ▼
       Best Torrent Found
               │
               ▼
     Download + Metadata
```

### Reliability Features

Gorrent incorporates techniques to ensure automated environments remain resilient:
*   **DNS-over-HTTPS (DoH):** Bypasses ISP DNS sinkholing on port 53 by routing internal DNS lookups over HTTPS (port 443) using public resolvers like Cloudflare (`1.1.1.1`).
*   **Circuit Breakers (Source Health):** If a source goes down, the engine won't hang waiting for timeouts. Consecutive failures temporarily bench a source, ensuring searches remain fast.
*   **Dead Torrent Protection:** If a torrent is added but has 0 seeders and fails to fetch metadata within 30 seconds, Gorrent automatically aborts and drops the torrent to prevent hanging the daemon.
*   **RuTracker Integration:** Access to RuTracker's extensive catalog, unlocked by providing your session cookie in the config.

## Agnostic Scoring Engine

Gorrent scores and ranks torrents based on your preferences using a data-driven Regex engine under the hood:
*   **Seeders Base Score:** Every torrent starts with a score equal to its active seeders.
*   **Agnostic Term Matching:** You provide a comma-separated list of terms in your `config.json` (e.g., `"language": "latino, es"`). Gorrent automatically compiles these into precise, word-boundary Regular Expressions. This ensures that a search for `es` matches the exact language code "es", but ignores the "es" inside the word "Series", without requiring you to write a regex yourself.

## API

Gorrent exposes a modern REST API designed for AI agents and automation.

**Auto-Download Request:**
```http
POST /api/download
Content-Type: application/json

{
    "auto": "Oppenheimer"
}
```

**Response:**
```json
{
    "status": "started",
    "magnet": "magnet:?xt=urn:btih:..."
}
```
## AI Native Integrations

Gorrent exposes interfaces specifically designed for AI agents:
*   **OpenClaw**: The `docs/openclaw-skill/SKILL.md` file and `openapi.yaml` endpoint allow OpenClaw agents to manage downloads natively via the API.
*   **Claude Desktop**: The `docs/claude-skill` folder provides a Custom Skill for Claude. By using Code Execution, Claude can invoke the local CLI wrapper (`./gorrent.sh`) to automate media fetching for you via the terminal.


## Deployment Guide (Docker)

The multi-arch image is built on `scratch` and weighs just a few megabytes.

### Quick Run (Zero Config)
If you just want to run it instantly with default settings and no configuration files, use this one-liner:
```bash
docker run -d --name gorrent -p 7800:7800 -v $(pwd)/downloads:/downloads ghcr.io/x-name15/gorrent:latest
```

### Docker Compose (Recommended)
For homelab users, mapping a config file is highly recommended.

1. **Create your `docker-compose.yml`**:
```yaml
services:
  gorrent:
    image: ghcr.io/x-name15/gorrent:latest
    container_name: gorrent
    ports:
      - "7800:7800"
    volumes:
      # Map your config file and downloads folder
      - ./config.json:/config.json
      - ./downloads:/downloads
    restart: unless-stopped
```

2. **Create your `config.json`**:
```json
{
  "daemon": {
    "port": 7800
  },
  "scraper": {
    "dns": "cloudflare",
    "rutracker_cookie": "tu_bb_session_cookie_aqui_opcional",
    "sources": [
      "yts", "1337x", "nyaa", "piratebay", 
      "fitgirl", "subsplease", "torrentscsv", "rutracker"
    ],
    "filters": {
      "language": "latino, castellano, multi-subs, es",
      "resolution": "1080p, 1080i",
      "min_seeders": "5"
    }
  },
  "torrent": {
    "download_dir": "/downloads"
  }
}
```

3. **Start the daemon**:
```bash
docker compose up -d
```

If you don't provide a `config.json`, Gorrent will load sane defaults, but mapping it is highly recommended to tune your preferred sources and filters.
> **Tip:** Don't want to use a specific tracker? Simply remove its name from the `"sources"` array in your config file.

## Optional CLI

If you do want to run it manually, it ships with CLI wrappers (`gorrent.sh` / `.bat`) that tunnel transparently into the Docker container.

```bash
$ ./gorrent.sh download --auto "Arcane S01"

Sending auto-download request for: Arcane S01
Download started successfully!
```

## Releases

Releases are generated automatically from [`CHANGELOG.md`](./CHANGELOG.md).

## License

GOrrent is licensed under the GPL v3. See [`LICENSE`](./LICENSE) for details.

## Credits

**Author:** Mr Jacket / Felix Manrique
