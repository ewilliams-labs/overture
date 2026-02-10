# Overture
A high-performance, context-aware playlist orchestration engine.

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
