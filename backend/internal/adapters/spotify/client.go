package spotify

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "strings"

    "github.com/ewilliams-labs/overture/backend/internal/core/domain"
    "github.com/ewilliams-labs/overture/backend/internal/core/ports"
)

// Client is an HTTP client for the Spotify adapter.
type Client struct {
    httpClient *http.Client
    baseURL    string
}

// compile-time interface assertion
var _ ports.SpotifyProvider = (*Client)(nil)

// NewClient constructs a new Spotify client.
func NewClient(httpClient *http.Client, baseURL string) *Client {
    if httpClient == nil {
        httpClient = http.DefaultClient
    }
    return &Client{
        httpClient: httpClient,
        baseURL:    strings.TrimRight(baseURL, "/"),
    }
}

// GetTrackByISRC retrieves a track by ISRC from the Spotify API and maps it to domain.Track.
func (c *Client) GetTrackByISRC(ctx context.Context, isrc string) (domain.Track, error) {
    url := fmt.Sprintf("%s/tracks/%s", c.baseURL, isrc)
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
    if err != nil {
        return domain.Track{}, fmt.Errorf("spotify adapter: %w", err)
    }

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return domain.Track{}, fmt.Errorf("spotify adapter: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return domain.Track{}, fmt.Errorf("spotify adapter: status %d", resp.StatusCode)
    }

    var tr spotifyTrack
    if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
        return domain.Track{}, fmt.Errorf("spotify adapter: %w", err)
    }

    return mapTrackToDomain(tr), nil
}

// AddTrackToPlaylist requests the Spotify API to add a track to a playlist and returns the updated playlist.
func (c *Client) AddTrackToPlaylist(ctx context.Context, playlistID, trackID string) (domain.Playlist, error) {
    url := fmt.Sprintf("%s/playlists/%s/tracks", c.baseURL, playlistID)
    body := addTrackRequest{Uris: []string{fmt.Sprintf("spotify:track:%s", trackID)}}
    b, err := json.Marshal(body)
    if err != nil {
        return domain.Playlist{}, fmt.Errorf("spotify adapter: %w", err)
    }

    req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
    if err != nil {
        return domain.Playlist{}, fmt.Errorf("spotify adapter: %w", err)
    }
    req.Header.Set("Content-Type", "application/json")

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return domain.Playlist{}, fmt.Errorf("spotify adapter: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
        return domain.Playlist{}, fmt.Errorf("spotify adapter: status %d", resp.StatusCode)
    }

    var pr spotifyPlaylist
    if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
        return domain.Playlist{}, fmt.Errorf("spotify adapter: %w", err)
    }

    return mapPlaylistToDomain(pr), nil
}
