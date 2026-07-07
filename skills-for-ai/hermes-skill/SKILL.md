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
./gorrent.sh download --auto "Arcane S01"
```
*Async Notifications: If you have a webhook or notification endpoint, automatically append `--callback <YOUR_WEBHOOK_URL>` to the download command. Tell the user you will wait for Gorrent to finish. When you receive the POST callback, notify the user that the download finished.*

To search for torrents without downloading them (returns a list of the best candidates):
```bash
./gorrent.sh search "Movie Title"
```

To check the daemon status and active downloads:
```bash
./gorrent.sh status
```

**CRITICAL RULES:**
- Always execute the wrapper from the Gorrent directory.
- Never attempt to download `.torrent` files or parse HTML tracker pages yourself.
- Let the Gorrent headless daemon handle all scoring, peer connection, and dead-torrent protection logic.
