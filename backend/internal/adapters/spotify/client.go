package spotify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ewilliams-labs/overture/backend/internal/core/domain"
)

const (
	// BaseURL is the production Spotify API endpoint
	BaseURL = "https://api.spotify.com/v1"
)

// Client adapts the Spotify API to our Domain interface
type Client struct {
	httpClient *http.Client
	baseURL    string
	// In the future, we will add auth tokens here
}

// NewClient creates a standard Spotify client.
// This is what you use in main.go.
func NewClient(clientID, clientSecret string) *Client {
	// For now, we use a default HTTP client.
	// Later, we will add an OAuth2 transport here.
	return &Client{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		baseURL:    BaseURL,
	}
}

// NewClientWithBaseURL creates a client with a custom base URL.
// This is strictly for TESTS (injecting the mock server URL).
func NewClientWithBaseURL(httpClient *http.Client, baseURL string) *Client {
	return &Client{
		httpClient: httpClient,
		baseURL:    baseURL,
	}
}

// GetTrackByISRC searches for a track using its International Standard Recording Code.
func (c *Client) GetTrackByISRC(ctx context.Context, isrc string) (domain.Track, error) {
	// 1. Search API (because /tracks/{id} requires a Spotify ID, not ISRC)
	url := fmt.Sprintf("%s/search?type=track&q=isrc:%s", c.baseURL, isrc)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return domain.Track{}, fmt.Errorf("spotify adapter: failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return domain.Track{}, fmt.Errorf("spotify adapter: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return domain.Track{}, fmt.Errorf("spotify adapter: status %d", resp.StatusCode)
	}

	// 2. Decode Response (Wrapper -> Tracks -> Items)
	var searchResp struct {
		Tracks struct {
			Items []spotifyTrack `json:"items"`
		} `json:"tracks"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return domain.Track{}, fmt.Errorf("spotify adapter: decode error: %w", err)
	}

	if len(searchResp.Tracks.Items) == 0 {
		return domain.Track{}, fmt.Errorf("spotify adapter: no track found for ISRC %s", isrc)
	}

	// 3. Map result (passing nil for features, as Search doesn't return them)
	return mapTrackToDomain(searchResp.Tracks.Items[0], nil), nil
}

// AddTrackToPlaylist adds a track to a playlist and returns the updated playlist.
// Note: This is a simplified implementation. Real Spotify API returns a Snapshot ID,
// so we usually have to fetch the playlist again to return the full domain object.
func (c *Client) AddTrackToPlaylist(ctx context.Context, playlistID, trackID string) (domain.Playlist, error) {
	// 1. Prepare the Request Body
	// Spotify requires URIs in the format "spotify:track:{id}"
	requestBody := map[string][]string{
		"uris": {fmt.Sprintf("spotify:track:%s", trackID)},
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return domain.Playlist{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	// 2. Create the POST Request
	url := fmt.Sprintf("%s/playlists/%s/tracks", c.baseURL, playlistID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return domain.Playlist{}, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// 3. Execute
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return domain.Playlist{}, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return domain.Playlist{}, fmt.Errorf("spotify adapter: status %d", resp.StatusCode)
	}

	// 4. Decode the Response
	// The test Mock returns the full Playlist JSON, so we decode it directly.
	var spPlaylist spotifyPlaylist
	if err := json.NewDecoder(resp.Body).Decode(&spPlaylist); err != nil {
		return domain.Playlist{}, fmt.Errorf("failed to decode playlist: %w", err)
	}

	// 5. Map to Domain
	return mapPlaylistToDomain(spPlaylist), nil
}
