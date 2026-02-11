package ports

import (
	"context"

	"github.com/ewilliams-labs/overture/backend/internal/core/domain"
)

type SpotifyProvider interface {
	GetTrackByMetadata(ctx context.Context, title, artist string) (domain.Track, error)
}
