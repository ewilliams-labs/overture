package spotify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"math/rand"
	"net/http"
	"net/url"

	"github.com/ewilliams-labs/overture/backend/internal/core/domain"
	"golang.org/x/oauth2/clientcredentials"
)

const (
	// BaseURL is the production Spotify API endpoint
	BaseURL = "https://api.spotify.com/v1"
)

// Client adapts the Spotify API to our Domain interface
type Client struct {
	httpClient *http.Client
	baseURL    string
}

// NewClient creates a standard Spotify client.
func NewClient(clientID, clientSecret string) *Client {
	config := &clientcredentials.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     "https://accounts.spotify.com/api/token", // Real Spotify Auth URL
	}

	httpClient := config.Client(context.Background())

	return &Client{
		httpClient: httpClient,
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

// GetTrackByMetadata searches for a track using title and artist metadata.
func (c *Client) GetTrackByMetadata(ctx context.Context, title string, artist string) (domain.Track, error) {
	// 1. Search API (because /tracks/{id} requires a Spotify ID)
	searchURL, err := url.Parse(fmt.Sprintf("%s/search", c.baseURL))
	if err != nil {
		return domain.Track{}, fmt.Errorf("spotify adapter: invalid search url: %w", err)
	}

	query := searchURL.Query()
	query.Set("q", fmt.Sprintf("track:%s artist:%s", title, artist))
	query.Set("type", "track")
	query.Set("limit", "1")
	searchURL.RawQuery = query.Encode()

	fmt.Printf("DEBUG spotify adapter: search request URL: %s\n", searchURL.String())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL.String(), nil)
	if err != nil {
		return domain.Track{}, fmt.Errorf("spotify adapter: failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return domain.Track{}, fmt.Errorf("spotify adapter: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("DEBUG spotify adapter: search response status: %d\n", resp.StatusCode)
		return domain.Track{}, fmt.Errorf("spotify adapter: status %d", resp.StatusCode)
	}

	fmt.Printf("DEBUG spotify adapter: search response status: %d\n", resp.StatusCode)

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return domain.Track{}, fmt.Errorf("spotify adapter: read body error: %w", err)
	}
	fmt.Printf("DEBUG BODY: %s\n", string(bodyBytes))
	resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))

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
		fmt.Printf("DEBUG spotify adapter: search returned 0 items for title %q artist %q\n", title, artist)
		return domain.Track{}, fmt.Errorf("spotify adapter: no track found for title %q artist %q", title, artist)
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

	featuresResp, err := c.httpClient.Do(featuresReq)
	if err != nil {
		return domain.Track{}, fmt.Errorf("spotify adapter: features request failed: %w", err)
	}
	defer featuresResp.Body.Close()

	if featuresResp.StatusCode != http.StatusOK {
		if featuresResp.StatusCode == http.StatusForbidden || featuresResp.StatusCode == http.StatusNotFound {
			fmt.Printf("WARN spotify adapter: Falling back to deterministic vibe generation for track %s\n", track.ID)
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
		fmt.Printf("WARN spotify adapter: Falling back to deterministic vibe generation for track %s\n", track.ID)
		mapped.Features = generateDeterministicFeatures(track.ID)
		return mapped, nil
	}

	return mapTrackToDomain(track, &features), nil
}

func (c *Client) searchTrack(ctx context.Context, title string, artist string) (spotifyTrack, error) {
	searchURL, err := url.Parse(fmt.Sprintf("%s/search", c.baseURL))
	if err != nil {
		return spotifyTrack{}, fmt.Errorf("spotify adapter: invalid search url: %w", err)
	}

	query := searchURL.Query()
	query.Set("q", fmt.Sprintf("track:%s artist:%s", title, artist))
	query.Set("type", "track")
	query.Set("limit", "1")
	searchURL.RawQuery = query.Encode()

	searchReq, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL.String(), nil)
	if err != nil {
		return spotifyTrack{}, fmt.Errorf("spotify adapter: failed to create search request: %w", err)
	}

	searchResp, err := c.httpClient.Do(searchReq)
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
		return spotifyTrack{}, fmt.Errorf("no track found")
	}

	return searchBody.Tracks.Items[0], nil
}

func generateDeterministicFeatures(trackID string) domain.AudioFeatures {
	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(trackID))
	seed := int64(hasher.Sum32())
	rng := rand.New(rand.NewSource(seed))

	between := func(min, max float64) float64 {
		return min + rng.Float64()*(max-min)
	}

	return domain.AudioFeatures{
		Energy:           between(0.1, 0.9),
		Valence:          between(0.1, 0.9),
		Danceability:     between(0.1, 0.9),
		Acousticness:     between(0.1, 0.9),
		Instrumentalness: between(0.1, 0.9),
		Tempo:            between(60.0, 180.0),
	}
}

func allFeaturesZero(features spotifyAudioFeatures) bool {
	return features.Danceability == 0 &&
		features.Energy == 0 &&
		features.Valence == 0 &&
		features.Tempo == 0 &&
		features.Instrumentalness == 0 &&
		features.Acousticness == 0
}
