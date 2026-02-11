package spotify

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/ewilliams-labs/overture/backend/internal/core/domain"
)

// GetTrack fetches a track by metadata and enriches it with audio features.
func (c *Client) GetTrack(ctx context.Context, title string, artist string) (domain.Track, error) {
	track, err := c.searchTrack(ctx, title, artist)
	if err != nil {
		return domain.Track{}, err
	}

	mapped := mapTrackToDomain(track, nil)

	featuresURL := fmt.Sprintf("%s/audio-features/%s", c.baseURL, track.ID)
	featuresReq, err := http.NewRequestWithContext(ctx, http.MethodGet, featuresURL, nil)
	if err != nil {
		return domain.Track{}, fmt.Errorf("spotify adapter: failed to create features request: %w", err)
	}

	featuresResp, err := c.doRequestWithRetry(featuresReq)
	if err != nil {
		return domain.Track{}, fmt.Errorf("spotify adapter: features request failed: %w", err)
	}
	defer featuresResp.Body.Close()

	if featuresResp.StatusCode != http.StatusOK {
		if featuresResp.StatusCode == http.StatusForbidden || featuresResp.StatusCode == http.StatusNotFound {
			log.Printf("WARN spotify adapter: falling back to deterministic vibe generation for track %s", track.ID)
			mapped.Features = generateDeterministicFeatures(track.ID)
			return mapped, nil
		}
		return domain.Track{}, fmt.Errorf("spotify adapter: features status %d", featuresResp.StatusCode)
	}

	var features spotifyAudioFeatures
	if err := json.NewDecoder(featuresResp.Body).Decode(&features); err != nil {
		return domain.Track{}, fmt.Errorf("spotify adapter: features decode error: %w", err)
	}

	if allFeaturesZero(features) {
		log.Printf("WARN spotify adapter: falling back to deterministic vibe generation for track %s", track.ID)
		mapped.Features = generateDeterministicFeatures(track.ID)
		return mapped, nil
	}

	return mapTrackToDomain(track, &features), nil
}
