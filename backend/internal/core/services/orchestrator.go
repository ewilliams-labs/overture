// Package services provides business logic orchestration for the Overture application.
package services

import (
	"context"
	"fmt"

	"github.com/ewilliams-labs/overture/backend/internal/core/domain"
	"github.com/ewilliams-labs/overture/backend/internal/core/ports"
	"github.com/google/uuid"
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
func (o *Orchestrator) AddTrackToPlaylist(ctx context.Context, playlistID string, title string, artist string) (domain.Playlist, error) {
	// 1. Fetch track metadata from Spotify
	track, err := o.spotify.GetTrackByMetadata(ctx, title, artist)
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
	playlist, err := o.repo.GetByID(ctx, id)
	if err != nil {
		return domain.AudioFeatures{}, fmt.Errorf("service: failed to load playlist: %w", err)
	}

	return playlist.Analyze(), nil
}
