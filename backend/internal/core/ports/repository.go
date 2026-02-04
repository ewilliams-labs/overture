package ports

import (
	"context"
	"github.com/ewilliams-labs/overture/backend/internal/core/domain"
)

type PlaylistRepository interface {
	GetByID(ctx context.Context, id string) (domain.Playlist, error)
	Save(ctx context.Context, p domain.Playlist) error
}