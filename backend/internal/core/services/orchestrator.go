package services

import (
	"context"
	"fmt"

	"github.com/ewilliams-labs/overture/backend/internal/core/domain"
	"github.com/ewilliams-labs/overture/backend/internal/core/ports"
)

// Orchestrator coordinates spotify and playlist repository operations.
type Orchestrator struct {
	spotify ports.SpotifyProvider
	repo    ports.PlaylistRepository
}

// NewOrchestrator constructs an Orchestrator.
func NewOrchestrator(spotify ports.SpotifyProvider, repo ports.PlaylistRepository) *Orchestrator {
	return &Orchestrator{
		spotify: spotify,
		repo:    repo,
	}
}

// AddTrackToPlaylist fetches a track from Spotify, adds it to the local playlist, and saves it.
// UPDATED: Now returns (domain.Playlist, error)
func (o *Orchestrator) AddTrackToPlaylist(ctx context.Context, playlistID string, isrc string) (domain.Playlist, error) {
	// 1. Fetch track metadata from Spotify
	track, err := o.spotify.GetTrackByISRC(ctx, isrc)
	if err != nil {
		return domain.Playlist{}, fmt.Errorf("service: failed to fetch track: %w", err)
	}

	// 2. Load playlist from local repository
	plVal, err := o.repo.GetByID(ctx, playlistID)
	if err != nil {
		return domain.Playlist{}, fmt.Errorf("service: failed to load playlist: %w", err)
	}

	// 3. Mutate the playlist (Pure Domain Logic)
	pl := &plVal
	if err := pl.AddTrack(track); err != nil {
		return domain.Playlist{}, fmt.Errorf("service: domain rule violation: %w", err)
	}

	// 4. Persist the updated playlist
	if err := o.repo.Save(ctx, *pl); err != nil {
		return domain.Playlist{}, fmt.Errorf("service: failed to save playlist: %w", err)
	}

	// 5. Return the updated playlist so the UI can update immediately
	return *pl, nil
}
