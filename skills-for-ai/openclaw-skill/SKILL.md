---
name: gorrent
description: Search and download torrents natively using the local Gorrent daemon.
---

# Gorrent Skill

You are interacting with Gorrent, a headless torrent search and automation daemon running at `http://localhost:7800`.
If the user has enabled API Key security, you MUST include the `X-API-Key` header in all HTTP requests (or pass `?apikey=<key>` as a query param).

When the user asks you to search for torrents or download something, use your HTTP tools to communicate with the Gorrent REST API.

## Endpoints

### 1. Search Torrents
`GET /api/search?q=<query>[&source=<name>]`

Returns a JSON array of results. Pick the one with the highest `score` or `seeders`.

**Available `source` values:** `yts`, `nyaa`, `piratebay`, `1337x`, `eztv`, `subsplease`, `fitgirl`, `torrentscsv`, `rutracker`

### 2. Download Torrent
`POST /api/download`

```json
{
  "magnet": "magnet:?xt=urn:btih:... (or bare 40-char hash)",
  "auto": "search query (Gorrent auto-picks best result)",
  "category": "movies | tvshows | anime | ...",
  "source": "restrict auto-search to one scraper",
  "callback": "https://your-webhook-url (POST when 100% done)"
}
```
Use `magnet` OR `auto`, not both. `category`, `source`, and `callback` are all optional.

**Webhook payload** (sent to `callback` when complete):
```json
{"event": "completed", "name": "...", "hash": "..."}
```

### 3. Status
`GET /api/status`

Returns array of active torrents:
```json
[{"hash":"...", "name":"...", "downloaded":102400, "length":1073741824, "peers":12}]
```

### 4. Stop Torrent
`DELETE /api/torrent?hash=<40-char-hash>`

Drops the torrent from the engine. **Files remain on disk** — safe for Plex/Jellyfin.

### 5. WebSocket (Real-Time)
`ws://localhost:7800/api/ws` — streams the full status array every 1 second.

### 6. Prometheus Metrics (No Auth)
`GET /metrics` — exports:
- `gorrent_torrents_active`
- `gorrent_bytes_downloaded`
- `gorrent_bytes_uploaded`

### 7. Health Check (No Auth)
`GET /health` → `{"status": "ok"}`

### 8. OpenAPI Spec (No Auth)
`GET /api/docs` → returns the full OpenAPI YAML specification.

## Config Automation (Zero-Touch UX)
If the user asks you to configure anything, directly modify `config.json`. Full schema:

### `daemon` block
- `port` (int): Daemon port. Default `7800`.
- `api_key` (string): Shared secret for auth.
- `data_dir` (string): State files directory. Default `"./data"`.

### `scraper` block
- `sources` (array): Active scrapers. Valid: `yts`, `nyaa`, `piratebay`, `1337x`, `eztv`, `subsplease`, `fitgirl`, `torrentscsv`, `rutracker`.
- `filters` (object): e.g. `{"language": "spanish", "quality": "1080p"}`.
- `dns` (string): DNS-over-HTTPS resolver. e.g. `"cloudflare"`, `"google"`, or raw IP.
- `rutracker_cookie` (string): RuTracker `bb_session` cookie.

### `torrent` block
- `download_dir` (string): Root download directory.
- `auto_export_torrent` (bool): Auto-save `.torrent` file alongside each download.
- `trackers` (array): Extra UDP/HTTP trackers appended to every magnet link.
- `category_dirs` (object): Category → absolute path mapping. e.g. `{"movies": "/downloads/movies"}`.
- `max_download_rate` (int, KB/s): Download cap. `0` = unlimited.
- `max_upload_rate` (int, KB/s): Upload/seed cap. `0` = unlimited.
- `auto_cleanup` (bool): Default `false`. Enables Garbage Collector.
- `seed_ratio` (float): GC drops torrent when ratio reaches this (e.g. `1.5`).
- `max_seed_days` (int): GC drops torrent after seeding this many days.
- `hardlink_dir` (string): **Optional.** Directory for Plex/Jellyfin hardlinks. Must be on the same physical disk as `download_dir` or it will fail.
- `post_script` (string): **Optional.** Bash script path run on completion. Env vars: `GORRENT_HASH`, `GORRENT_NAME`, `GORRENT_PATH`, `GORRENT_CATEGORY`.

### `rss` block
- `interval_min` (int): Polling interval in minutes.
- `feeds` (array):
  - `url` (string): RSS feed URL.
  - `category` (string): Target category for downloads.
  - `regex` (array of strings): Case-insensitive title patterns. Empty = download all.

Always return status/results nicely formatted to the user.
