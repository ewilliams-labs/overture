package spotify

import "github.com/ewilliams-labs/overture/backend/internal/core/domain"

// spotifyTrack represents the Spotify API response for a track.
type spotifyTrack struct {
    ID     string             `json:"id"`
    Title  string             `json:"title"`
    Artist string             `json:"artist"`
    Album  string             `json:"album,omitempty"`
    ISRC   string             `json:"isrc"`
    Vibe   map[string]float64 `json:"vibe,omitempty"`
}

// toDomain converts a spotifyTrack to a domain.Track.
func (st spotifyTrack) toDomain() domain.Track {
    return domain.Track{
        ID:     st.ID,
        Title:  st.Title,
        Artist: st.Artist,
        Album:  st.Album,
        ISRC:   st.ISRC,
        Vibe:   st.Vibe,
    }
}

// spotifyPlaylist represents the Spotify API response for a playlist.
type spotifyPlaylist struct {
    ID     string         `json:"id"`
    Name   string         `json:"name"`
    Tracks []spotifyTrack `json:"tracks"`
}

// toDomain converts a spotifyPlaylist to a domain.Playlist.
func (sp spotifyPlaylist) toDomain() domain.Playlist {
    tracks := make([]domain.Track, len(sp.Tracks))
    for i, st := range sp.Tracks {
        tracks[i] = st.toDomain()
    }
    return domain.Playlist{
        ID:     sp.ID,
        Name:   sp.Name,
        Tracks: tracks,
    }
}

// addTrackRequest represents the request body for adding a track to a playlist.
type addTrackRequest struct {
    Uris []string `json:"uris"`
}