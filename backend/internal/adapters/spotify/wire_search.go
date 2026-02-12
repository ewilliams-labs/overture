package spotify

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/ewilliams-labs/overture/backend/internal/core/domain"
	"github.com/ewilliams-labs/overture/backend/internal/core/ports"
)

const searchMatchThreshold = 0.8

// GetTrackByMetadata searches for a track using title and artist metadata.
func (c *Client) GetTrackByMetadata(ctx context.Context, title string, artist string) (domain.Track, error) {
	track, err := c.searchTrack(ctx, title, artist)
	if err != nil {
		return domain.Track{}, err
	}

	return mapTrackToDomain(track, nil), nil
}

func (c *Client) searchTrack(ctx context.Context, title string, artist string) (spotifyTrack, error) {
	searchURL, err := url.Parse(fmt.Sprintf("%s/search", c.baseURL))
	if err != nil {
		return spotifyTrack{}, fmt.Errorf("spotify adapter: invalid search url: %w", err)
	}

	normalizedTitle, normalizedArtist := normalizeTitleArtist(title, artist)
	queryTitle := fallbackIfEmpty(normalizedTitle, title)
	queryArtist := fallbackIfEmpty(normalizedArtist, artist)

	query := searchURL.Query()
	query.Set("q", fmt.Sprintf("track:%s artist:%s", queryTitle, queryArtist))
	query.Set("type", "track")
	query.Set("limit", "5")
	searchURL.RawQuery = query.Encode()

	log.Printf("DEBUG spotify adapter: search request URL: %s", searchURL.String())

	searchReq, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL.String(), nil)
	if err != nil {
		return spotifyTrack{}, fmt.Errorf("spotify adapter: failed to create search request: %w", err)
	}

	searchResp, err := c.doRequestWithRetry(searchReq)
	if err != nil {
		return spotifyTrack{}, fmt.Errorf("spotify adapter: search request failed: %w", err)
	}
	defer searchResp.Body.Close()

	if searchResp.StatusCode != http.StatusOK {
		return spotifyTrack{}, fmt.Errorf("spotify adapter: search status %d", searchResp.StatusCode)
	}

	var searchBody struct {
		Tracks struct {
			Items []spotifyTrack `json:"items"`
		} `json:"tracks"`
	}

	if err := json.NewDecoder(searchResp.Body).Decode(&searchBody); err != nil {
		return spotifyTrack{}, fmt.Errorf("spotify adapter: search decode error: %w", err)
	}

	if len(searchBody.Tracks.Items) == 0 {
		return spotifyTrack{}, fmt.Errorf("spotify adapter: no track found for title %q artist %q", title, artist)
	}

	maxItems := len(searchBody.Tracks.Items)
	if maxItems > 5 {
		maxItems = 5
	}
	bestScore := 0.0
	bestIndex := -1
	for i := 0; i < maxItems; i++ {
		candidate := searchBody.Tracks.Items[i]
		candidateArtist := joinArtistNames(candidate)
		score := ScoreResult(artist, title, candidateArtist, candidate.Name)
		log.Printf("DEBUG spotify adapter: Spotify Match: %s - %s (Score: %.2f)", candidateArtist, candidate.Name, score)
		if score >= searchMatchThreshold && score > bestScore {
			bestScore = score
			bestIndex = i
		}
	}

	if bestIndex == -1 {
		return spotifyTrack{}, fmt.Errorf("spotify adapter: %w", &ports.NoConfidentMatchError{Title: title, Artist: artist})
	}

	return searchBody.Tracks.Items[bestIndex], nil
}
