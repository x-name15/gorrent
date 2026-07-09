---
name: gorrent
description: Search and download torrents natively using the local Gorrent daemon.
---

# Gorrent Skill

You are interacting with Gorrent, a headless torrent search and automation daemon running at `http://localhost:7800`.
If the user has enabled API Key security, you MUST include the `X-API-Key` header in all HTTP requests.

When the user asks you to search for torrents or download something, you should use the `web_fetch` or `exec` tool to communicate with the Gorrent REST API.

## Endpoints:

### 1. Search Torrents
Endpoint: `GET http://localhost:7800/api/search?q={query}&source={source}`
Use this to search for a specific movie, game, or anime. The `source` parameter is optional (e.g. `nyaa`, `yts`) to target a specific tracker. It returns a JSON array of torrent results. Pick the one with the highest `score` or `seeders`.

### 2. Add/Download Torrent
Endpoint: `POST http://localhost:7800/api/download`
Body: `{"magnet": "magnet:?xt=urn:...", "category": "movies"}` or `{"auto": "The Matrix", "category": "movies", "source": "yts"}`
Use this to start a download.
If the user asks to "download X", you can just send `{"auto": "X"}` and Gorrent will automatically find and start the best match. You can optionally include a `category` (like movies, tvshows, anime) to route the files to specific folders, and a `source` to restrict the search to a single tracker.

**Async Notifications:** If your platform supports receiving webhooks, include `"callback": "http://your-webhook-url"` in the JSON payload so Gorrent can notify you when the download hits 100%. Tell the user you will notify them when it's done, and do so once you receive the webhook.

### 3. Status
Endpoint: `GET http://localhost:7800/api/status`
Use this to check the status of active downloads. (Note: A live WebSocket is also available at `/api/ws` if you need real-time streaming).

### 4. Stop Torrent
Endpoint: `DELETE http://localhost:7800/api/torrent?hash={hash}`
Use this to cancel and remove an active download if the user requests it.

Always return the status or results nicely formatted to the user.
