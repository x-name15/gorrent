## Metadata
name: Gorrent
description: Control and interact with Gorrent, a headless homelab P2P torrent client. Search, score, and download torrents automatically.

## Overview
Gorrent is a headless automation-first torrent client built in Go. It runs in a Docker container and exposes a CLI wrapper and a REST API. You can use this skill to search for torrents, download them to the local `downloads/` directory, check download status, and fully configure all automation via `config.yaml`.

## How to use Gorrent

### CLI Commands
Use the provided wrapper scripts (`./gorrent.sh` on Linux/Mac, `.\gorrent.bat` on Windows).
These are thin wrappers around `docker exec -it gorrent /gorrent "$@"`.

**Search:**
```bash
./gorrent.sh search [--source <name>] <query>
```

**Download:**
```bash
# Auto-search and download best match:
./gorrent.sh download --auto <query> [--source <name>] [--category <name>] [--callback <url>]
# Download a specific magnet link or 40-char infohash:
./gorrent.sh download <magnet_or_hash> [--category <name>] [--callback <url>]
```

**Check Status:**
```bash
./gorrent.sh status
```

**Stop Download:**
```bash
./gorrent.sh stop <hash>
```

**Available `--source` values** (restrict search to one scraper):
`yts`, `nyaa`, `piratebay`, `1337x`, `eztv`, `subsplease`, `fitgirl`, `torrentscsv`, `rutracker`, `bittorrented`

**Available `--category` values** (routes file to category folder configured in `config.yaml`):
e.g. `movies`, `tvshows`, `anime` — or whatever the user has set in `category_dirs`.

**`--callback <url>`**: Gorrent will POST `{"event":"completed","name":"...","hash":"..."}` to this URL when the download hits 100%.

### REST API
Daemon listens on `http://localhost:7800`. If `api_key` is set in config, include `X-API-Key: <key>` header or `?apikey=<key>` query param.

- **Search**: `GET /api/search?q=<query>[&source=<name>]`
- **Download**: `POST /api/download` — body: `{"magnet":"..."}` or `{"auto":"...","category":"...","source":"...","callback":"..."}`
- **Status**: `GET /api/status` — returns `[{hash, name, downloaded, length, peers}]`
- **Stop**: `DELETE /api/torrent?hash=<hash>`
- **WebSocket**: `ws://localhost:7800/api/ws` — streams status every 1s
- **Metrics**: `GET /metrics` — Prometheus text (no auth needed). Exports: `gorrent_torrents_active`, `gorrent_bytes_downloaded`, `gorrent_bytes_uploaded`
- **Health**: `GET /health` — returns `{"status":"ok"}` (no auth needed)
- **File Streaming**: `GET /files/<relative/path>` — serves files from `download_dir` with full HTTP Range support. Ideal for streaming video to VLC, browser, or media players without moving files. Auth required if `api_key` is set.
- **OpenAPI Docs**: `GET /api/docs` — returns the full OpenAPI YAML spec (no auth needed)

## Advanced Config Automation (Zero-Touch User Experience)
If the user asks you to configure anything, you MUST directly edit `config.yaml` — do NOT ask them to do it manually. You have full awareness of every config field:

### `daemon` block
- `port` (int): Port the daemon listens on. Default: `7800`.
- `api_key` (string): If set, all API requests must include `X-API-Key` header or `?apikey=` param.
- `data_dir` (string): Where internal state files (like `rss_history.json`) are stored. Default: `"./data"`.

### `scraper` block
- `sources` (array of strings): Active scrapers. Valid values: `yts`, `nyaa`, `piratebay`, `1337x`, `eztv`, `subsplease`, `fitgirl`, `torrentscsv`, `rutracker`, `bittorrented`.
- `filters` (map): Key/value filters (e.g. `language: spanish` and `quality: 1080p`).
- `dns` (string): DNS resolver for all HTTP requests. e.g. `"cloudflare"`, `"google"`, or a raw IP `"8.8.8.8"`.
- `rutracker_cookie` (string): Your RuTracker `bb_session` cookie. Only needed to activate that scraper.

### `torrent` block
- `download_dir` (string): Root download directory.
- `auto_export_torrent` (bool): Auto-save a `.torrent` file alongside each download.
- `trackers` (array of strings): Extra UDP/HTTP trackers appended to every magnet.
- `category_dirs` (map): Map of category → absolute path. e.g. `movies: /downloads/movies`.
- `max_download_rate` (int, KB/s): Bandwidth cap for downloads. `0` = unlimited.
- `max_upload_rate` (int, KB/s): Bandwidth cap for uploads/seeding. `0` = unlimited.
- `auto_cleanup` (bool): **Optional, default false.** Enable the P2P Garbage Collector.
- `seed_ratio` (float): GC trigger — drops torrent when upload/download ratio reaches this value (e.g. `1.5`).
- `max_seed_days` (int): GC trigger — drops torrent after it has been seeding for this many days.
- `hardlink_dir` (string): **Optional.** Root directory where zero-byte hardlinks of completed torrents are created for Plex/Jellyfin. MUST be on the same physical disk as `download_dir`.
- `post_script` (string): **Optional.** Path to a script executed on download completion. Receives env vars: `GORRENT_HASH`, `GORRENT_NAME`, `GORRENT_PATH`, `GORRENT_CATEGORY`. For Docker use the `callback` webhook instead.
- `watch_dir` (string): **Optional, default empty (disabled).** Gorrent polls this directory every 5 seconds. Drop a `.magnet` or `.txt` file containing a magnet URI and Gorrent auto-downloads it. Processed files are archived to `watch_dir/handled/`.
- `delete_files_on_stop` (bool): **Optional, default false.** When the GC drops a torrent, also permanently delete its files from disk. Default is `false` — Gorrent's philosophy is to always keep files for Plex/Jellyfin. Only set to `true` if the user explicitly wants disk space rotation. This is irreversible.

### `rss` block
- `interval_min` (int): How often to poll all RSS feeds (in minutes).
- `feeds` (array): List of RSS feed objects:
  - `url` (string): Full RSS feed URL (e.g. `https://nyaa.si/?page=rss&q=subsplease+1080p`).
  - `category` (string): Category to download into (e.g. `anime`).
  - `regex` (array of strings): Case-insensitive patterns to match against torrent title (e.g. `["Arcane", "Solo Leveling"]`). Leave empty to download everything.

## When to use this skill
- When the user asks you to find a movie, game, software, or book via torrent.
- When the user asks to download a specific magnet link.
- When the user asks about the status of their current torrent downloads.
- When the user says anything about configuring Gorrent (RSS, Plex, bandwidth, cleanup, etc.).
