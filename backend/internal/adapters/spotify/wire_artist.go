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

// spotifyArtist represents an artist from the Spotify API.
type spotifyArtist struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// GetArtistTopTracks searches for an artist by name and returns their top tracks.
// Returns up to 10 tracks (Spotify's maximum for top tracks endpoint).
func (c *Client) GetArtistTopTracks(ctx context.Context, artistName string) ([]domain.Track, error) {
	// 1. Search for the artist to get their ID
	artistID, err := c.searchArtist(ctx, artistName)
	if err != nil {
		return nil, fmt.Errorf("spotify adapter: failed to find artist %q: %w", artistName, err)
	}

	// 2. Fetch the artist's top tracks
	tracks, err := c.getTopTracks(ctx, artistID)
	if err != nil {
		return nil, fmt.Errorf("spotify adapter: failed to get top tracks for artist %q: %w", artistName, err)
	}

	// 3. Fetch audio features for all tracks in batch
	trackIDs := make([]string, len(tracks))
	for i, t := range tracks {
		trackIDs[i] = t.ID
	}

	features, err := c.getAudioFeaturesBatch(ctx, trackIDs)
	if err != nil {
		// Log but don't fail - features are optional for filtering
		log.Printf("WARN spotify adapter: failed to get audio features: %v", err)
		features = make(map[string]spotifyAudioFeatures)
	}

	// 4. Map to domain tracks with features
	domainTracks := make([]domain.Track, len(tracks))
	for i, st := range tracks {
		var f *spotifyAudioFeatures
		if feat, ok := features[st.ID]; ok {
			f = &feat
		}
		domainTracks[i] = mapTrackToDomain(st, f)
	}

	return domainTracks, nil
}

// searchArtist searches for an artist by name and returns their Spotify ID.
func (c *Client) searchArtist(ctx context.Context, artistName string) (string, error) {
	searchURL, err := url.Parse(fmt.Sprintf("%s/search", c.baseURL))
	if err != nil {
		return "", fmt.Errorf("invalid search url: %w", err)
	}

	query := searchURL.Query()
	query.Set("q", artistName)
	query.Set("type", "artist")
	query.Set("limit", "1")
	query.Set("market", "US")
	searchURL.RawQuery = query.Encode()

	log.Printf("DEBUG spotify adapter: artist search URL: %s", searchURL.String()) // #nosec G706 -- URL is internally constructed from trusted baseURL constant

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL.String(), nil)
	if err != nil {
		return "", fmt.Errorf("failed to create search request: %w", err)
	}

	resp, err := c.doRequestWithRetry(req)
	if err != nil {
		return "", fmt.Errorf("search request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("search status %d", resp.StatusCode)
	}

	var searchBody struct {
		Artists struct {
			Items []spotifyArtist `json:"items"`
		} `json:"artists"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&searchBody); err != nil {
		return "", fmt.Errorf("search decode error: %w", err)
	}

	if len(searchBody.Artists.Items) == 0 {
		return "", fmt.Errorf("no artist found with name %q", artistName)
	}

	return searchBody.Artists.Items[0].ID, nil
}

// getTopTracks fetches an artist's top tracks from Spotify.
func (c *Client) getTopTracks(ctx context.Context, artistID string) ([]spotifyTrack, error) {
	topTracksURL := fmt.Sprintf("%s/artists/%s/top-tracks?market=US", c.baseURL, artistID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, topTracksURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create top tracks request: %w", err)
	}

	resp, err := c.doRequestWithRetry(req)
	if err != nil {
		return nil, fmt.Errorf("top tracks request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("top tracks status %d", resp.StatusCode)
	}

	var body struct {
		Tracks []spotifyTrack `json:"tracks"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("top tracks decode error: %w", err)
	}

	return body.Tracks, nil
}

// getAudioFeaturesBatch fetches audio features for multiple tracks in a single request.
func (c *Client) getAudioFeaturesBatch(ctx context.Context, trackIDs []string) (map[string]spotifyAudioFeatures, error) {
	if len(trackIDs) == 0 {
		return make(map[string]spotifyAudioFeatures), nil
	}

	featuresURL, err := url.Parse(fmt.Sprintf("%s/audio-features", c.baseURL))
	if err != nil {
		return nil, fmt.Errorf("invalid features url: %w", err)
	}

	query := featuresURL.Query()
	ids := ""
	for i, id := range trackIDs {
		if i > 0 {
			ids += ","
		}
		ids += id
	}
	query.Set("ids", ids)
	featuresURL.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, featuresURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create features request: %w", err)
	}

	resp, err := c.doRequestWithRetry(req)
	if err != nil {
		return nil, fmt.Errorf("features request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("features status %d", resp.StatusCode)
	}

	var body struct {
		AudioFeatures []struct {
			ID               string  `json:"id"`
			Danceability     float64 `json:"danceability"`
			Energy           float64 `json:"energy"`
			Valence          float64 `json:"valence"`
			Tempo            float64 `json:"tempo"`
			Instrumentalness float64 `json:"instrumentalness"`
			Acousticness     float64 `json:"acousticness"`
		} `json:"audio_features"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("features decode error: %w", err)
	}

	result := make(map[string]spotifyAudioFeatures, len(body.AudioFeatures))
	for _, f := range body.AudioFeatures {
		if f.ID != "" { // Spotify returns null for some tracks
			result[f.ID] = spotifyAudioFeatures{
				Danceability:     f.Danceability,
				Energy:           f.Energy,
				Valence:          f.Valence,
				Tempo:            f.Tempo,
				Instrumentalness: f.Instrumentalness,
				Acousticness:     f.Acousticness,
			}
		}
	}

	return result, nil
}
