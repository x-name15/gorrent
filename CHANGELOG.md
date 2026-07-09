# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.6.0] - 2026-07-09 — Huge, MAJOR update with all bs

### Added
- **Native Hardlinking**: Added `hardlink_dir` config parameter. Gorrent now automatically creates zero-byte hardlinks (preserving category folders) for completed downloads, allowing seamless integration with Plex/Jellyfin without interrupting the seeding process.
- **Post-Processing Scripts**: Added `post_script` config parameter. Gorrent can now execute any script when a torrent reaches 100%. Environment variables (`GORRENT_HASH`, `GORRENT_NAME`, `GORRENT_PATH`, `GORRENT_CATEGORY`) are injected automatically. For Docker deployments, use the `callback` webhook instead.
- **Post-Processing State Manager**: Created a background engine that writes `gorrent_processed.json` to `data_dir` to guarantee scripts and links are executed exactly once per torrent.
- **AI Skills Full Awareness**: Taught Claude, Hermes, and OpenClaw how to manipulate all v1.6.0 parameters via `config.json` so users can enable these features strictly via natural language.

### Fixed
- **RSS — Nyaa/ShowRSS magnet resolution**: RSS parser now correctly extracts magnet links from `<enclosure url="magnet:...">` and `<torrent:magnetURI>` fields. Previously only read `<link>`, which for Nyaa is a page URL, not a magnet.
- **RSS — history key stability**: History tracker now uses the magnet URI as key instead of `<link>`, making it consistent across feeds that report items differently.
- **RSS — unnecessary disk writes**: `rss_history.json` is now saved once per feed poll instead of once per matched item.
- **Post-Processor — wrong state directory**: `gorrent_processed.json` now goes to `data_dir` instead of `download_dir`, consistent with `rss_history.json`.
- **Post-Processor — `bash` hardcoded**: `post_script` now runs the script directly as an executable instead of via `bash`, allowing native binaries and properly-configured scripts to work.
- **Security — Hardlink path traversal**: Added validation that hardlink destination is always under `hardlink_dir` to prevent malicious torrent names from escaping the target directory.
- **WebSocket — idle goroutine leak**: WS handler now sends periodic pings and sets a 60s read deadline, disconnecting dead clients automatically.
- **Status API — `BytesReadData` serialized as object**: Fixed `stats.BytesReadData` to use `.Int64()` so it serializes correctly as a number in JSON.
- **Performance — regex compiled per request**: Infohash validation regex is now a package-level `var` instead of being recompiled on every download request.
- **Docker — `data/` volume not mounted**: `docker-compose.yml` now mounts `./data:/data` so RSS history and post-processing state survive container restarts.
- **Docker — missing healthcheck**: `docker-compose.yml` now uses `/health` endpoint for Docker health monitoring.

---
## [1.5.5] - 2026-07-09 — The Automation Engine

### Added
- **RSS Auto-Downloader**: Added native, zero-dependency XML parser to monitor RSS feeds (e.g. Nyaa, ShowRSS) and automatically download torrents matching your Regex rules into specific categories. Maintains a clean `rss_history.json` state in your data directory to prevent duplicates.
- **P2P Garbage Collector**: Added optional Auto-Cleanup system (`auto_cleanup` in `config.json`). Drops active torrents from the daemon once they reach a target `seed_ratio` (e.g. 1.5) or `max_seed_days` to conserve bandwidth and RAM. Downloaded files are kept 100% intact for Plex/Jellyfin.
- **AI Agent Skills Updated**: Taught Claude, Hermes, and OpenClaw how to manage the new RSS feeds, Auto-Cleanup settings, and Bandwidth Throttling directly via `config.json`, as well as interacting with the public `/metrics` endpoint.
- **OpenAPI Specification**: Fully synchronized `cmd/daemon/openapi.yaml` with v1.5.x, including exact `X-API-Key` authentication mappings, the new `source` search parameters, and async webhook `callback` fields.

---
## [1.5.1] - 2026-07-09 — Source Targeting Optimization

### Added
- **Source Targeting**: The API and CLI now support filtering by a specific tracker (e.g. `--source nyaa` or `?source=yts`). This stops the engine from concurrently hitting all 9 scrapers, reducing unnecessary noise and preventing IP bans when searching for niche content.
- **AI Agent Skills Updated**: Fully updated the bundled `.md` skills for Claude, Hermes, and OpenClaw so that AI agents natively understand how to use the new `--source` targeting.

---
## [1.5.0] - 2026-07-08 — The Homelab Grail Update

### Added
- **Bandwidth Throttling**: Added `max_download_rate` and `max_upload_rate` (in KB/s) to `config.json` to prevent Gorrent from choking your local network.
- **WebSocket Endpoint**: Added `ws://localhost:7800/api/ws` to stream live download status at 1Hz, paving the way for real-time Web UIs.
- **Prometheus Metrics & Healthcheck**: Added `/metrics` exposing raw Prometheus format stats (`gorrent_bytes_downloaded`, etc.) and a `/health` endpoint for Docker auto-healing.
- **AI Agent Skills Updated**: Fully updated the bundled `.md` skills for Claude, Hermes, and OpenClaw so that AI agents natively understand how to use the new `--category` flag, `stop` commands, `X-API-Key` headers, and WebSockets.

---
## [1.1.5] - 2026-07-08 — Auto-Export, Security & Download Management

### Added
- **Auto-Export .torrent files**: Gorrent can now optionally export a `.torrent` backup file into your `downloads` directory the moment it finishes fetching the metadata for any magnet link (Issue #61). This feature is disabled by default to prevent clutter, and can be enabled by setting `"auto_export_torrent": true` in the `torrent` section of your `config.json`.
- **Stop & Delete Torrents**: Added a new `DELETE /api/torrent?hash=...` endpoint and a `./gorrent.sh stop <hash>` CLI command to abort and clean up active downloads.
- **API Key Security**: Added optional API Key authentication. Set `"api_key": "your_secret"` in the `daemon` config block to secure the REST API against unauthorized access on your local network. The CLI wrapper automatically uses it.
- **Bare Infohash Support**: Gorrent can now accept a raw 40-character infohash instead of a full magnet link for downloads 
- **Custom Trackers**: You can now define an array of `"trackers"` in the `torrent` block of your `config.json`. These trackers will be automatically injected into every magnet link processed by the daemon, boosting DHT peer discovery
- **Category-Based Directories**: The CLI and API now support an optional `--category` flag (e.g. `--category movies`). Gorrent will save the torrent in a subfolder named after the category (e.g. `/downloads/movies`), or map it to a specific directory if defined in the new `"category_dirs"` config object. Perfect for homelab media server organization!

---
## [1.1.1] - 2026-07-07 — Zero-Config AI Callbacks

### Added
- **AI Asynchronous Notifications**: Introduced a zero-configuration `--callback <URL>` flag to the CLI wrapper. This allows AI agents (like OpenClaw or Hermes) to pass their webhook URL natively when starting a download. The Gorrent daemon will automatically POST to this webhook when the download hits 100%, allowing AI agents to notify users proactively.

---
## [1.1.0] - 2026-07-07 — Hermes Skill Support

### Added
- **Hermes Agent Integration**: Added native `SKILL.md` support in `skills-for-ai/hermes-skill/` to allow Hermes Agent to easily control the Gorrent daemon via the CLI wrappers.

### Changed
- **Docker Image Optimization**: Added the `skills-for-ai/` directory to `.dockerignore` to prevent bloating the production container with AI-specific metadata and instructions.

---
## [1.0.2] - 2026-07-07 — Ups, hotfixes Part 2!

### Fixed
- **Release Pipeline**: Fixed script filenames (`gorrent.sh` and `gorrent.bat`) in the GitHub Actions bundle step, which caused the v1.0.1 release to halt before publishing Docker images and GitHub Release archives.
- **Release Pipeline**: Added `go mod tidy` step before compilation to resolve missing `go.sum` entries for multi-arch/OS builds.

---
## [1.0.1] - 2026-07-07 — Ups, hotfixes!

### Fixed
- **FitGirl Scraper**: Removed duplicate HTTP requests and deprecated stream-of-consciousness code comments for improved scraping performance.
- **RuTracker Scraper**: Cleaned up deprecated internal monologue comments regarding CP1251 encoding.
- **EZTV Scraper**: Cleaned up deprecated internal monologue comments regarding legacy TypeScript ports.

---
## [1.0.0] - 2026-07-07 — GOrrent is Alive!

### Added
- **Gorrent Engine**: Initial release of the Go rewrite, focused on a headless, automation-first architecture.
- **REST API**: Endpoints (`/api/search`, `/api/download`, `/api/status`) designed for seamless AI integration.
- **CLI Wrappers**: Included `gorrent.sh` and `gorrent.bat` for transparent local usage.
- **Concurrent Scraping**: 9 supported sources out of the box (YTS, 1337x, Nyaa, PirateBay, FitGirl, RuTracker, SubsPlease, EZTV, TorrentsCSV).
- **Agnostic Scoring Engine**: Dynamic regex word-boundary filtering loaded from `config.json` for precise ranking.
- **Embedded P2P Client**: Fully integrated BitTorrent client using `anacrolix/torrent` with a 30-second Dead Torrent Protection feature.
- **DoH (DNS-over-HTTPS)**: Built-in hijacking of HTTP transport to evade ISP censorship via Cloudflare or other public resolvers.
- **Circuit Breakers**: Intelligent cooldown for failing trackers to prevent search hangs.
- **AI Native Skills**: Packaged definitions for OpenClaw (`docs/openclaw-skill/SKILL.md` / `openapi.yaml`) and Claude Desktop (`docs/claude-skill/skill.md`).
- **CI/CD Pipelines**: Automated GitHub Actions for fast testing, multi-arch binary generation (`amd64`/`arm64`), and GHCR Docker publishing.
- **Docker Scratch Build**: Microscopic zero-dependency container deployment.
