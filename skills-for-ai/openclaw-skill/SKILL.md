---
name: gorrent
description: Search and download torrents natively using the local Gorrent daemon.
---

# Gorrent Skill

You are interacting with Gorrent, a headless torrent search and automation daemon running at `http://localhost:7800`.

When the user asks you to search for torrents or download something, you should use the `web_fetch` or `exec` tool to communicate with the Gorrent REST API.

## Endpoints:

### 1. Search Torrents
Endpoint: `GET http://localhost:7800/api/search?q={query}`
Use this to search for a specific movie, game, or anime. It returns a JSON array of torrent results. Pick the one with the highest `score` or `seeders`.

### 2. Add/Download Torrent
Endpoint: `POST http://localhost:7800/api/download`
Body: `{"magnet": "magnet:?xt=urn:..."}` or `{"auto": "The Matrix"}`
Use this to start a download.
If the user asks to "download X", you can just send `{"auto": "X"}` and Gorrent will automatically find and start the best match.

**Async Notifications:** If your platform supports receiving webhooks, include `"callback": "http://your-webhook-url"` in the JSON payload so Gorrent can notify you when the download hits 100%. Tell the user you will notify them when it's done, and do so once you receive the webhook.

### 3. Status
Endpoint: `GET http://localhost:7800/api/status`
Use this to check the status of active downloads.

Always return the status or results nicely formatted to the user.
