package ports

import (
	"context"
	"errors"
	"fmt"

	"github.com/ewilliams-labs/overture/backend/internal/core/domain"
)

// ErrNoConfidentMatch indicates search results did not meet the confidence threshold.
var ErrNoConfidentMatch = errors.New("no confident match")

// NoConfidentMatchError provides context for a failed track match.
type NoConfidentMatchError struct {
	Title  string
	Artist string
}

func (e NoConfidentMatchError) Error() string {
	if e.Title == "" && e.Artist == "" {
		return ErrNoConfidentMatch.Error()
	}
	return fmt.Sprintf("no confident match found for title %q artist %q", e.Title, e.Artist)
}

func (e NoConfidentMatchError) Is(target error) bool {
	return target == ErrNoConfidentMatch
}

type SpotifyProvider interface {
	GetTrackByMetadata(ctx context.Context, title, artist string) (domain.Track, error)
}
