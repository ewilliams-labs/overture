package ports

import (
	"context"
	"github.com/ewilliams-labs/overture/backend/internal/core/domain"
)
type SpotifyProvider interface {
	GetTrackByISRC(ctx context.Context, isrc string) (domain.Track, error)
}