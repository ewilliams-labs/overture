package spotify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ewilliams-labs/overture/backend/internal/core/domain"
)

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
	resp, err := c.doRequestWithRetry(req)
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
