---
name: gorrent-automation
description: Automates downloading torrents using the local Gorrent daemon via the CLI wrapper.
version: 1.7.0
author: Mr Jacket
license: GPL-3.0
metadata:
  hermes:
    tags: [Media, Torrent, Automation, Download, Homelab]
    blueprint:
      schedule: "every 30m"
      prompt: "Run ./gorrent.sh status. If any torrents have completed since your last check, notify the user immediately."
---

# Gorrent Automation Skill

You are an AI agent managing the user's media library via the `gorrent` daemon.

## When to Use
Use this skill when the user asks you to download a movie, TV show, anime, software, or when they want to search for torrents, or when they ask to configure any Gorrent feature.

## How to Use
Use the local `gorrent` CLI wrapper (`gorrent.sh` on macOS/Linux, `gorrent.bat` on Windows).

**Search:**
```bash
./gorrent.sh search [--source <name>] <query>
```

**Download (auto-pick best result):**
```bash
./gorrent.sh download --auto <query> [--source <name>] [--category <name>] [--callback <url>]
```

**Download (specific magnet or infohash):**
```bash
./gorrent.sh download <magnet_or_40char_hash> [--category <name>] [--callback <url>]
```

**Check status:**
```bash
./gorrent.sh status
```

**Stop a torrent:**
```bash
./gorrent.sh stop <hash>
```

**Available `--source` values** (restrict to one scraper):
`yts`, `nyaa`, `piratebay`, `1337x`, `eztv`, `subsplease`, `fitgirl`, `torrentscsv`, `rutracker`, `bittorrented`

**Available `--category` values:** anything set in the user's `category_dirs` config (e.g. `movies`, `tvshows`, `anime`).

**`--callback <url>`:** Gorrent will POST `{"event":"completed","name":"...","hash":"..."}` to this URL when download finishes.

## Advanced Config Automation (Zero-Touch User Experience)
If the user asks you to automate something, DO NOT ask them to edit files. YOU must directly edit the `config.yaml` file in the project root. Full schema:

### `daemon` block
- `port` (int): Daemon port. Default `7800`.
- `log_level` (string): Verbosity level. Options: `minimal` (default) or `debug`.
- `api_key` (string): If set, all requests must include `X-API-Key` header or `?apikey=` param.
- `data_dir` (string): Directory for internal state files. Default `"./data"`.

### `scraper` block
- `sources` (array): Active scrapers. Valid: `yts`, `nyaa`, `piratebay`, `1337x`, `eztv`, `subsplease`, `fitgirl`, `torrentscsv`, `rutracker`, `bittorrented`.
- `filters` (map): e.g. `language: spanish`.
- `dns` (string): DNS resolver. e.g. `"cloudflare"`, `"google"`, `"8.8.8.8"`.
- `rutracker_cookie` (string): RuTracker `bb_session` cookie (only needed to activate that source).

### `torrent` block
- `download_dir` (string): Root download path.
- `auto_export_torrent` (bool): Auto-saves `.torrent` file alongside download.
- `trackers` (array): Extra UDP/HTTP trackers appended to every magnet.
- `category_dirs` (map): Map of category name → absolute path. e.g. `movies: /downloads/movies`.
- `max_download_rate` (int, KB/s): Download speed cap. `0` = unlimited.
- `max_upload_rate` (int, KB/s): Upload/seed speed cap. `0` = unlimited.
- `auto_cleanup` (bool): Default `false`. Enables P2P Garbage Collector.
- `seed_ratio` (float): GC drops torrent when ratio reaches this value (e.g. `1.5`).
- `max_seed_days` (int): GC drops torrent after seeding this many days.
- `hardlink_dir` (string): **Optional.** Directory for zero-byte hardlinks for Plex/Jellyfin. Must be on the same physical disk as `download_dir`.
- `post_script` (string): **Optional.** Path to bash script run on download completion. Env vars injected: `GORRENT_HASH`, `GORRENT_NAME`, `GORRENT_PATH`, `GORRENT_CATEGORY`.
- `watch_dir` (string): **Optional.** Drop `.magnet` or `.txt` files (containing a magnet URI) here → Gorrent auto-downloads them within 5 seconds and moves them to `watch_dir/handled/`. Leave empty (default) to disable.
- `delete_files_on_stop` (bool): **Optional, default `false`.** When `auto_cleanup` GC drops a torrent, also permanently deletes its files from disk. **Default is `false`** — files are always kept on disk for Plex/Jellyfin. Only set to `true` if the user explicitly asks for disk space rotation.

### `rss` block
- `interval_min` (int): Polling interval in minutes.
- `feeds` (array): Each feed has `url`, `category`, and `regex` (array of case-insensitive patterns). Empty `regex` = download everything.

**CRITICAL RULES:**
- Always execute the wrapper from the Gorrent directory.
- Never attempt to download `.torrent` files or parse HTML tracker pages yourself.
- Let the Gorrent daemon handle all scoring, peer connection, and dead-torrent protection.
- Seeding is good — never enable `auto_cleanup` unless the user explicitly asks.
