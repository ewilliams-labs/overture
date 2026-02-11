package spotify

import (
	"strings"

	"github.com/ewilliams-labs/overture/backend/internal/core/domain"
)

// spotifyTrack represents the messy JSON from Spotify
type spotifyTrack struct {
	ID         string `json:"id"`
	Name       string `json:"name"` // API uses "name", Domain uses "Title"
	DurationMs int    `json:"duration_ms"`
	Artists    []struct {
		Name string `json:"name"`
	} `json:"artists"` // API is a list, Domain is a string
	Album struct {
		Name   string `json:"name"`
		Images []struct {
			URL string `json:"url"`
		} `json:"images"`
	} `json:"album"` // API is an object, Domain is a string
}

// spotifyAudioFeatures represents the separate API call for "Vibes"
type spotifyAudioFeatures struct {
	Danceability     float64 `json:"danceability"`
	Energy           float64 `json:"energy"`
	Valence          float64 `json:"valence"`
	Tempo            float64 `json:"tempo"`
	Instrumentalness float64 `json:"instrumentalness"`
	Acousticness     float64 `json:"acousticness"`
}

type spotifyPlaylist struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// Spotify wraps the list of tracks in a paging object
	Tracks struct {
		Items []struct {
			Track spotifyTrack `json:"track"` // The wrapper!
		} `json:"items"`
	} `json:"tracks"`
}

// toDomain combines the track metadata and audio features into a clean Domain entity
func (st spotifyTrack) toDomain(features spotifyAudioFeatures) domain.Track {
	// 1. Flatten Artists (List -> String)
	var artistNames []string
	for _, a := range st.Artists {
		artistNames = append(artistNames, a.Name)
	}

	// 2. Extract Album Cover
	coverURL := ""
	if len(st.Album.Images) > 0 {
		coverURL = st.Album.Images[0].URL
	}

	// 3. Map to Domain
	return domain.Track{
		ID:         st.ID,
		Title:      st.Name,
		Artist:     strings.Join(artistNames, ", "),
		Album:      st.Album.Name,
		CoverURL:   coverURL,
		DurationMs: st.DurationMs,

		// Map the features cleanly
		Features: domain.AudioFeatures{
			Danceability:     features.Danceability,
			Energy:           features.Energy,
			Valence:          features.Valence,
			Tempo:            features.Tempo,
			Instrumentalness: features.Instrumentalness,
			Acousticness:     features.Acousticness,
		},
	}
}

// addTrackRequest represents the request body for adding a track to a playlist.
type addTrackRequest struct {
	Uris []string `json:"uris"`
}
