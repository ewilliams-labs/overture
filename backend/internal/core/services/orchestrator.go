// Package services provides business logic orchestration for the Overture application.
package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/ewilliams-labs/overture/backend/internal/core/domain"
	"github.com/ewilliams-labs/overture/backend/internal/core/ports"
	"github.com/google/uuid"
)

// Orchestrator coordinates spotify and playlist repository operations.
type Orchestrator struct {
	spotify ports.SpotifyProvider
	repo    ports.PlaylistRepository
	intent  ports.IntentCompiler
}

// NewOrchestrator constructs an Orchestrator.
func NewOrchestrator(spotify ports.SpotifyProvider, repo ports.PlaylistRepository, intent ports.IntentCompiler) *Orchestrator {
	return &Orchestrator{
		spotify: spotify,
		repo:    repo,
		intent:  intent,
	}
}

// IntentResult contains the result of processing an intent, including the parsed
// intent object and a summary of the playlist population.
type IntentResult struct {
	Intent          domain.IntentObject
	TracksEvaluated int
	TracksAdded     int
	Summary         string
}

// ProcessIntent analyzes a user message, fetches matching tracks, filters them
// based on vibe constraints, and adds them to the specified playlist.
//
// Note: The caller should pass a detached context (e.g., context.WithoutCancel)
// if this is called from a background goroutine where client disconnection
// should not cancel the operation.
func (o *Orchestrator) ProcessIntent(ctx context.Context, playlistID string, message string) (IntentResult, error) {
	if o.intent == nil {
		return IntentResult{}, fmt.Errorf("service: intent compiler not configured")
	}

	// 1. Analyze intent from message
	intent, err := o.intent.AnalyzeIntent(ctx, message)
	if err != nil {
		return IntentResult{}, fmt.Errorf("service: failed to analyze intent: %w", err)
	}

	// 2. Get existing playlist to check for duplicates
	playlist, err := o.repo.GetByID(ctx, playlistID)
	if err != nil {
		return IntentResult{}, fmt.Errorf("service: failed to load playlist: %w", err)
	}

	// Build a set of existing track IDs for deduplication
	existingTracks := make(map[string]bool)
	for _, t := range playlist.Tracks {
		existingTracks[t.ID] = true
	}

	// 3. Fetch top tracks for each artist
	var allTracks []domain.Track
	seenTracks := make(map[string]bool) // For deduplication across artists

	for _, artist := range intent.Entities.Artists {
		tracks, err := o.spotify.GetArtistTopTracks(ctx, artist)
		if err != nil {
			// Log but continue with other artists
			continue
		}

		for _, track := range tracks {
			// Skip if we've already seen this track from another artist
			if seenTracks[track.ID] {
				continue
			}
			seenTracks[track.ID] = true
			allTracks = append(allTracks, track)
		}
	}

	// 4. Filter tracks based on vibe constraints
	var matchingTracks []domain.Track
	for _, track := range allTracks {
		// Skip if already in playlist
		if existingTracks[track.ID] {
			continue
		}

		// Check against vibe constraints
		if matchesConstraints(track.Features, intent) {
			matchingTracks = append(matchingTracks, track)
		}
	}

	// 5. Add matching tracks to playlist
	if len(matchingTracks) > 0 {
		if err := o.repo.AddTracksToPlaylist(ctx, playlistID, matchingTracks); err != nil {
			return IntentResult{}, fmt.Errorf("service: failed to add tracks to playlist: %w", err)
		}
	}

	// 6. Build summary
	artistNames := ""
	if len(intent.Entities.Artists) > 0 {
		artistNames = intent.Entities.Artists[0]
		if len(intent.Entities.Artists) > 1 {
			artistNames += " and others"
		}
	}

	summary := fmt.Sprintf("Found %d tracks, added %d matching your '%s' vibe",
		len(allTracks), len(matchingTracks), artistNames)

	return IntentResult{
		Intent:          intent,
		TracksEvaluated: len(allTracks),
		TracksAdded:     len(matchingTracks),
		Summary:         summary,
	}, nil
}

// HasIntentCompiler returns true if an intent compiler is configured.
func (o *Orchestrator) HasIntentCompiler() bool {
	return o.intent != nil
}

// AddTrackToPlaylist fetches a track from Spotify, adds it to the local playlist, and saves it.
// It returns the playlist ID on success.
func (o *Orchestrator) AddTrackToPlaylist(ctx context.Context, playlistID string, title string, artist string) (string, string, string, error) {
	// 1. Fetch track metadata from Spotify
	track, err := o.spotify.GetTrack(ctx, title, artist)
	if err != nil {
		return "", "", "", fmt.Errorf("service: failed to fetch track: %w", err)
	}

	// 2. Load playlist from local repository
	plVal, err := o.repo.GetByID(ctx, playlistID)
	if err != nil {
		return "", "", "", fmt.Errorf("service: failed to load playlist: %w", err)
	}

	// 3. Mutate the playlist (Pure Domain Logic)
	pl := &plVal
	if err := pl.AddTrack(track); err != nil {
		return "", "", "", fmt.Errorf("service: domain rule violation: %w", err)
	}

	// 4. Persist the updated playlist
	if err := o.repo.Save(ctx, *pl); err != nil {
		return "", "", "", fmt.Errorf("service: failed to save playlist: %w", err)
	}

	// 5. Return the playlist ID so clients can fetch details if needed
	return playlistID, track.ID, track.PreviewURL, nil
}

// CreatePlaylist initializes a new empty playlist and persists it.
func (o *Orchestrator) CreatePlaylist(ctx context.Context, name string) (domain.Playlist, error) {
	if name == "" {
		return domain.Playlist{}, fmt.Errorf("service: playlist name cannot be empty")
	}

	// 1. Create the Domain Entity
	// We generate the ID here so the entity is valid before it ever touches the DB.
	newPlaylist := domain.Playlist{
		ID:     uuid.New().String(),
		Name:   name,
		Tracks: []domain.Track{}, // Empty slice, not nil, is safer for JSON serialization
	}

	// 2. Persist to Repository
	if err := o.repo.Save(ctx, newPlaylist); err != nil {
		return domain.Playlist{}, fmt.Errorf("service: failed to persist new playlist: %w", err)
	}

	return newPlaylist, nil
}

// GetPlaylist loads a playlist by ID from the repository.
func (o *Orchestrator) GetPlaylist(ctx context.Context, playlistID string) (domain.Playlist, error) {
	if playlistID == "" {
		return domain.Playlist{}, fmt.Errorf("service: playlist id cannot be empty")
	}

	pl, err := o.repo.GetByID(ctx, playlistID)
	if err != nil {
		return domain.Playlist{}, fmt.Errorf("service: failed to load playlist: %w", err)
	}

	return pl, nil
}

// GetPlaylistAnalysis loads a playlist and returns its analyzed audio features.
func (o *Orchestrator) GetPlaylistAnalysis(ctx context.Context, id string) (domain.AudioFeatures, error) {
	features, err := o.repo.GetPlaylistAudioFeatures(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.AudioFeatures{}, err
		}
		return domain.AudioFeatures{}, fmt.Errorf("service: failed to load playlist analysis: %w", err)
	}

	return features, nil
}

// matchesConstraints checks if a track's audio features satisfy the given vibe constraints.
// Returns true if all non-nil constraints are satisfied (track passes the "vibe check").
//
// For each constraint field (Energy, Valence, Acousticness, Instrumentalness):
//   - If the constraint is nil, the check is skipped (no filtering on that dimension)
//   - If the constraint's Min and Max are both 0, the check is skipped
//   - Otherwise, the track's value must fall within [Min, Max] range
func matchesConstraints(features domain.AudioFeatures, constraints domain.IntentObject) bool {
	vc := constraints.VibeConstraints

	// Check Energy constraint
	if !checkConstraint(features.Energy, vc.Energy) {
		return false
	}

	// Check Valence constraint
	if !checkConstraint(features.Valence, vc.Valence) {
		return false
	}

	// Check Acousticness constraint
	if !checkConstraint(features.Acousticness, vc.Acoustic) {
		return false
	}

	// Check Instrumentalness constraint
	if !checkConstraint(features.Instrumentalness, vc.Instrument) {
		return false
	}

	return true
}

// checkConstraint validates a single audio feature value against a constraint.
// Returns true if the constraint is nil, has zero bounds, or the value is within range.
func checkConstraint(value float64, constraint *domain.VibeConstraint) bool {
	// Skip if constraint is nil
	if constraint == nil {
		return true
	}

	// Skip if both Min and Max are 0 (no meaningful constraint set)
	if constraint.Min == 0 && constraint.Max == 0 {
		return true
	}

	// Check if value falls within the range
	return value >= constraint.Min && value <= constraint.Max
}
