// Package ports defines the interfaces (ports) for the core domain.
package ports

import (
	"context"

	"github.com/ewilliams-labs/overture/backend/internal/core/domain"
)

type PlaylistRepository interface {
	GetByID(ctx context.Context, id string) (domain.Playlist, error)
	GetPlaylistAudioFeatures(ctx context.Context, playlistID string) (domain.AudioFeatures, error)
	UpdateTrackFeatures(ctx context.Context, trackID string, features domain.AudioFeatures) error
	Save(ctx context.Context, p domain.Playlist) error
	AddTracksToPlaylist(ctx context.Context, playlistID string, tracks []domain.Track) error
}
