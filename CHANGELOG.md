# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.2] - 2026-07-07 — Ups, hotfixes Part 2!
### Fixed
- **Release Pipeline**: Fixed script filenames (`gorrent.sh` and `gorrent.bat`) in the GitHub Actions bundle step, which caused the v1.0.1 release to halt before publishing Docker images and GitHub Release archives.

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
