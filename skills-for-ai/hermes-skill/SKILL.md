---
name: gorrent-automation
description: Automates downloading torrents using the local Gorrent daemon via the CLI wrapper.
version: 1.0.0
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
Use this skill when the user asks you to download a movie, TV show, anime, or software, or when they want to search for a specific torrent.

## How to Use
You must use the local `gorrent` CLI wrapper to interact with the background daemon.
The wrapper is available as `gorrent.sh` (macOS/Linux) or `gorrent.bat` (Windows) in the user's Gorrent directory.

To search and automatically download the best result based on the user's config:
```bash
./gorrent.sh download --source nyaa --category movies --auto "Arcane S01"
```
*(The `--category` and `--source` flags are optional but highly recommended to organize media and avoid unnecessary tracker hits!)*

*Async Notifications: If you have a webhook or notification endpoint, automatically append `--callback <YOUR_WEBHOOK_URL>` to the download command. Tell the user you will wait for Gorrent to finish. When you receive the POST callback, notify the user that the download finished.*

To search for torrents without downloading them (returns a list of the best candidates):
```bash
./gorrent.sh search --source yts "Movie Title"
```

To check the daemon status and active downloads:
```bash
./gorrent.sh status
```

### RSS Feed & Auto-Cleanup Management
If the user asks you to "subscribe to an RSS feed", "track an anime automatically", or "clean up old torrents":
DO NOT use the CLI or REST API. You must directly edit the `config.json` file in the project root.
- **For RSS:** Add their requested feed to the `"rss"` block. Include the URL, the target `category` (e.g. `anime`, `tvshows`), and an array of `"regex"` patterns (like `["[SubsPlease] Arcane", "Solo Leveling"]`) so Gorrent knows what to download.
- **For Cleanup:** Modify the `"torrent"` block to set `"auto_cleanup": true`, `"seed_ratio": 1.5`, and/or `"max_seed_days": 3` based on their request.
- **For Bandwidth Throttling:** If the user wants to limit download/upload speeds, edit `"max_download_rate"` and `"max_upload_rate"` (in KB/s) inside the `"torrent"` block in `config.json`.

To stop and delete an active download:
```bash
./gorrent.sh stop <hash>
```

**CRITICAL RULES:**
- Always execute the wrapper from the Gorrent directory.
- Never attempt to download `.torrent` files or parse HTML tracker pages yourself.
- Let the Gorrent headless daemon handle all scoring, peer connection, and dead-torrent protection logic.
