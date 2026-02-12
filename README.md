# Overture

A high-performance, context-aware playlist orchestration engine.

## Project Roadmap & Status

### [x] Phase 1: Core Infrastructure & Ingestion (Current)

- [x] Hexagonal Architecture Setup (Domain, Ports, Adapters)
- [x] Spotify Auth (Client Credentials Flow)
- [x] Playlist Management (Create, Store, Retrieve)
- [x] Pivot: Metadata Search (Title/Artist) replacing ISRC
- [x] Stabilize API Connectivity (Fix 503/Proxy issues)
- [x] Implement Deterministic Vibe Fallback (Handle missing API data)

### [x] Phase 2: Background Audio Processing & Persistence

- [x] Implement GET /playlists/{id}/analysis endpoint
- [x] Build Background Worker for Audio Processing
	- (Worker pool deployed; currently using Deterministic Fallback for missing URLs)
	- Implemented real-time RMS energy analysis via 'go-mp3'.
	- Decoupled storage using Repository Factory pattern (SQLite/Postgres ready).
- [ ] Integration: Go-Audio/FFmpeg for local BPM calculation
- [ ] Strategy: "Real" analysis with Deterministic Fallback safety net

### [ ] Phase 3: The Intent Engine (AI Integration)

- [ ] Integrate Ollama (Llama 3 / Mistral)
- [ ] Prompt Engineering: "Natural Language -> Vibe Vector" translation
- [ ] Generative UI: Frontend adapts to Vibe state

## Technical Architecture

- **Backend:** Go 1.25+ (Hexagonal Architecture)
- **Frontend:** React (Feature-Sliced Design)
- **AI Engine:** Ollama / DeepSeek
- **Storage Decoupling:** The system uses a Repository Factory. Local development defaults to SQLite. Production can be toggled to Postgres via the 'STORAGE_DRIVER' environment variable without changing business logic.

## Audio Features Reference

- **Valence (0.0 - 1.0):** A measure of musical positiveness. High valence sounds happy/cheerful; low valence sounds sad/depressed/angry.
- **Energy (0.0 - 1.0):** Represents a perceptual measure of intensity and activity. High energy tracks feel fast, loud, and noisy.
- **Danceability (0.0 - 1.0):** Describes how suitable a track is for dancing based on tempo, rhythm stability, beat strength, and overall regularity.
- **Acousticness (0.0 - 1.0):** A confidence measure of whether the track is acoustic (1.0) versus electronic/amplified (0.0).
- **Instrumentalness (0.0 - 1.0):** Predicts whether a track contains no vocals. Values above 0.5 are intended to represent instrumental tracks.
- **Tempo (BPM):** The overall estimated tempo of a track in beats per minute.

## Getting Started

- `just setup`: Install dependencies.
- `just validate`: Primary entry point for verifying the full ingestion and analysis pipeline.

## ✅ Acceptance Criteria Status

- [x] Health Check (200 OK)
- [x] Valid Track Addition (Title/Artist Search)
- [x] Fuzzy Match Validation
- [x] Fail on Low Confidence (422 Unprocessable Entity)
- [x] Playlist Retrieval (Non-empty tracks)
- [x] Audio Feature Persistence (Non-zero Energy/Valence)
- [x] Async Worker Job Submission
- [x] Background Feature Persistence (Update Logic)
- [x] Background Worker Infrastructure (Async Job Pool)
- [x] Automated Verification of Audio Persistence (Polling in tests)
- [x] Deterministic Fallback for Missing Audio Metadata
