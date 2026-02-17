# Overture

A high-performance, context-aware playlist orchestration engine with AI-powered musical reasoning.

---

## Getting Started

### Prerequisites

- **Go 1.22+**
- **Just** command runner (`brew install just` or `cargo install just`)
- **Spotify Developer Credentials** ([Create an app](https://developer.spotify.com/dashboard))

### Quick Start

```bash
# 1. Clone and setup
cd backend
just setup

# 2. Configure environment
cp .env.example .env
# Edit .env with your Spotify credentials

# 3. Run the server
just run
```

### Environment Variables

| Variable | Required | Description |
| -------- | -------- | ----------- |
| `SPOTIFY_CLIENT_ID` | Yes | Spotify API client ID |
| `SPOTIFY_CLIENT_SECRET` | Yes | Spotify API client secret |
| `OLLAMA_HOST` | No | Ollama server URL (auto-detected in WSL2) |
| `OLLAMA_MODEL` | No | Model name (auto-detected from available models) |
| `STORAGE_DRIVER` | No | `sqlite` (default) or `postgres` |

---

## Hardware Acceleration (GPU vs CPU)

Overture uses **Ollama** for AI-powered musical reasoning—translating natural language requests like *"give me a chill acoustic set"* into structured vibe constraints.

### Automatic GPU Detection

When running in **WSL2**, Overture automatically bridges to the Windows host for GPU acceleration:

```text
🔍 WSL2 detected. Using Windows host at 172.26.16.1
✅ Ollama detected at http://172.26.16.1:11434. Enabling AI tests.
📦 Using model: deepseek-v2:latest
```

Supported GPUs: **AMD RX 7900 XT**, **NVIDIA RTX series**, or any Ollama-compatible accelerator.

### Graceful Degradation

If Ollama is unavailable:

- AI-dependent tests are **automatically skipped**
- Core playlist functionality remains fully operational
- Perfect for **CI/CD pipelines** or low-power devices (Surface Pro, etc.)

```text
⚠️  Ollama not reachable. Skipping AI tests.
```

---

## Justfile Command Reference

| Command | Description |
| ------- | ----------- |
| `just run` | Start the development server on `:8080` |
| `just test` | Run unit tests with verbose output |
| `just validate` | **Full-stack integration suite** — handles server lifecycle, GPU detection, and acceptance tests |
| `just demo` | Create a demo playlist and add sample tracks |
| `just clean` | Remove database, binaries, and temp logs |

> **Note:** `just validate` is the primary entry point for CI/CD verification.

---

## Example API Usage (cURL)

### Health Check

```bash
curl http://localhost:8080/health
```

### Create Playlist

```bash
curl -X POST http://localhost:8080/playlists \
  -H "Content-Type: application/json" \
  -d '{"name": "Late Night Vibes"}'
```

### Add Track

```bash
curl -X POST http://localhost:8080/playlists/{id}/tracks \
  -H "Content-Type: application/json" \
  -d '{"title": "Blinding Lights", "artist": "The Weeknd"}'
```

### Intent Processing (SSE Streaming)

The intent endpoint uses **Server-Sent Events (SSE)** for real-time streaming. Use `-N` to disable buffering:

```bash
curl -N -X POST http://localhost:8080/playlists/{id}/intent \
  -H "Content-Type: application/json" \
  -d '{"message": "I want a chill acoustic set with Willie Nelson vibes"}'
```

**Example SSE Response:**

```text
event: status
data: {"status":"thinking","message":"Overture is analyzing the vibe..."}

event: status
data: {"status":"heartbeat"}

event: complete
data: {"status":"complete","artists_found":1,"tracks_added":5,"tracks_filtered":2}
```

---

## Technical Design Notes

### Detached Contexts

Background database writes use `context.WithoutCancel()` to ensure persistence completes even if the client disconnects mid-stream. This prevents partial writes during long-running AI operations.

### SSE Heartbeats

The intent endpoint sends periodic heartbeat events (`event: status`) to keep connections alive during extended reasoning operations (up to 120s for larger models).

### Architecture

- **Hexagonal / Ports & Adapters** — Domain logic isolated from infrastructure
- **Repository Factory** — SQLite (dev) / Postgres (prod) via `STORAGE_DRIVER`
- **Worker Pool** — Background audio analysis with job queue

---

## Project Roadmap & Status

### [x] Phase 1: Core Infrastructure & Ingestion

- [x] Hexagonal Architecture Setup (Domain, Ports, Adapters)
- [x] Spotify Auth (Client Credentials Flow)
- [x] Playlist Management (Create, Store, Retrieve)
- [x] Metadata Search (Title/Artist)
- [x] Deterministic Vibe Fallback

### [x] Phase 2: Background Audio Processing & Persistence

- [x] GET /playlists/{id}/analysis endpoint
- [x] Background Worker Pool
- [x] Real-time RMS energy analysis via 'go-mp3'
- [x] Repository Factory pattern (SQLite/Postgres ready)

### [x] Phase 3: AI Intent Engine (Ollama Integration)

- [x] Intent Schema (ADR 003) implemented
- [x] SSE streaming with heartbeats
- [x] Automatic GPU detection (WSL2 → Windows host)
- [x] Vibe constraint matching and track filtering
- [x] Artist top tracks integration

---

## Audio Features Reference

| Feature | Range | Description |
| ------- | ----- | ----------- |
| **Valence** | 0.0 - 1.0 | Musical positiveness (high = happy, low = sad) |
| **Energy** | 0.0 - 1.0 | Intensity and activity level |
| **Danceability** | 0.0 - 1.0 | Suitability for dancing |
| **Acousticness** | 0.0 - 1.0 | Acoustic vs electronic |
| **Instrumentalness** | 0.0 - 1.0 | Vocal presence (> 0.5 = instrumental) |
| **Tempo** | BPM | Beats per minute |

---

## Acceptance Criteria Status

- [x] Health Check (200 OK)
- [x] Valid Track Addition (Title/Artist Search)
- [x] Fuzzy Match Validation
- [x] Fail on Low Confidence (422 Unprocessable Entity)
- [x] Playlist Retrieval (Non-empty tracks)
- [x] Audio Feature Persistence
- [x] Background Worker Infrastructure
- [x] Natural Language Intent Parsing (SSE)
- [x] GPU-Accelerated AI Reasoning
