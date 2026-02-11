# Overture

A high-performance, context-aware playlist orchestration engine.

## Project Roadmap & Status

### Phase 1: Core Infrastructure & Ingestion (Current)

- [x] Hexagonal Architecture Setup (Domain, Ports, Adapters)
- [x] Spotify Auth (Client Credentials Flow)
- [x] Playlist Management (Create, Store, Retrieve)
- [x] Pivot: Metadata Search (Title/Artist) replacing ISRC
- [ ] Stabilize API Connectivity (Fix 503/Proxy issues)
- [ ] Implement Deterministic Vibe Fallback (Handle missing API data)

### Phase 2: The Vibe Engine (Audio Analysis)

- [ ] Implement GET /playlists/{id}/analysis endpoint
- [ ] Build Background Worker for Audio Processing
- [ ] Integration: Go-Audio/FFmpeg for local BPM calculation
- [ ] Strategy: "Real" analysis with Deterministic Fallback safety net

### Phase 3: The Intent Engine (AI Integration)

- [ ] Integrate Ollama (Llama 3 / Mistral)
- [ ] Prompt Engineering: "Natural Language -> Vibe Vector" translation
- [ ] Generative UI: Frontend adapts to Vibe state

## Architecture

- **Backend:** Go 1.25+ (Hexagonal Architecture)
- **Frontend:** React (Feature-Sliced Design)
- **AI Engine:** Ollama / DeepSeek

## Audio Features Reference

- **Valence (0.0 - 1.0):** A measure of musical positiveness. High valence sounds happy/cheerful; low valence sounds sad/depressed/angry.
- **Energy (0.0 - 1.0):** Represents a perceptual measure of intensity and activity. High energy tracks feel fast, loud, and noisy.
- **Danceability (0.0 - 1.0):** Describes how suitable a track is for dancing based on tempo, rhythm stability, beat strength, and overall regularity.
- **Acousticness (0.0 - 1.0):** A confidence measure of whether the track is acoustic (1.0) versus electronic/amplified (0.0).
- **Instrumentalness (0.0 - 1.0):** Predicts whether a track contains no vocals. Values above 0.5 are intended to represent instrumental tracks.
- **Tempo (BPM):** The overall estimated tempo of a track in beats per minute.
