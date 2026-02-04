package services

import (
	"context"

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

// AddTrackToPlaylist fetches a track from Spotify, adds it to the playlist and saves it.
func (o *Orchestrator) AddTrackToPlaylist(ctx context.Context, playlistID string, isrc string) error {

	// fetch track from spotify provider
	track, err := o.spotify.GetTrackByISRC(ctx, isrc)
	if err != nil {
		return err
	}

	// load playlist from repository (returns a struct)
	plVal, err := o.repo.GetByID(ctx, playlistID)
	if err != nil {
		return err
	}

	// use pointer to call AddTrack (pointer receiver) and mutate playlist
	pl := &plVal
	if err := pl.AddTrack(track); err != nil {
		return err
	}

	// persist updated playlist (accepts struct)
	if err := o.repo.Save(ctx, *pl); err != nil {
		return err
	}

	return nil
}