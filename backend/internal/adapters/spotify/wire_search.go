package spotify

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/ewilliams-labs/overture/backend/internal/core/domain"
)

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
	query.Set("limit", "1")
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

	candidate := searchBody.Tracks.Items[0]
	score, ok := trackMatchScore(title, artist, candidate)
	if !ok {
		return spotifyTrack{}, fmt.Errorf("spotify adapter: no match for title %q artist %q (score %.2f)", title, artist, score)
	}

	return candidate, nil
}
