# ADR 002: Pivot Vibe Engine Strategy due to Spotify API Feb 2026 Deprecation

## Status

Accepted

## Context

The February 2026 Spotify Web API update introduced breaking changes that disabled Overture's core "Vibe" functionality:

1. The "Get Audio Features" endpoint returns 403 Forbidden/404, preventing us from fetching Energy, Valence, and BPM.
2. The "external_ids" field was removed from Track objects, breaking ISRC-based search and requiring a pivot to "Title + Artist" metadata search.

The "North Star" of Overture (Generative UI) relies on this data to function.

## Options Considered

1. Option A: Local Audio Analysis (Selected)
   - Logic: Analyze the raw audio bytes (via "preview_url") using a Go-based DSP library to calculate BPM and Energy.
   - Pros: Removes dependency on third-party metadata; High engineering value/robustness.
   - Cons: Complex implementation; "preview_url" availability is inconsistent.

2. Option B: Data Augmentation (Secondary APIs)
   - Logic: Fetch missing metadata from Last.fm or AcoustID.
   - Pros: Easier than DSP; Real human-tagged data.
   - Cons: Adds external dependencies; Taxonomy doesn't map 1:1 to "Valence" or "Energy".

3. Option C: Random/Mock Data
   - Logic: Generate random numbers to unblock the UI.
   - Pros: Zero effort; Immediate fix.
   - Cons: Data is meaningless; Fails to demonstrate the core value of the product.

## Decision

We choose Option A (Local Audio Analysis) with a "Deterministic Fallback" strategy.

- Primary Strategy: We will build a local analysis engine to process audio samples when available.
- Immediate Mitigation (The Fallback): To unblock UI development immediately, we will implement a deterministic generator that hashes the TrackID to produce consistent, pseudo-random Vibe stats.
- Rationale: This approach guarantees stable data for the Frontend immediately while laying the groundwork for a truly independent, audio-first architecture.

## Consequences

- The backend must now handle "missing" audio features gracefully.
- We need to introduce a DSP library (e.g., go-audio) in the future.
- Search logic has been permanently refactored to use "Title + Artist" queries instead of ISRC.
