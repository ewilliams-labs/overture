package spotify

import (
	"strings"

	"github.com/ewilliams-labs/overture/backend/internal/core/domain"
)

// mapTrackToDomain converts a raw Spotify track to a clean Domain track.
// features can be nil if we are mapping from a Playlist (where features aren't provided).
func mapTrackToDomain(st spotifyTrack, features *spotifyAudioFeatures) domain.Track {
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

	// 3. Map Basic Metadata
	dt := domain.Track{
		ID:         st.ID,
		Title:      st.Name,
		Artist:     strings.Join(artistNames, ", "),
		Album:      st.Album.Name,
		CoverURL:   coverURL,
		DurationMs: st.DurationMs,
		ISRC:       st.ExternalIDs.ISRC,
	}

	// 4. Map Features (if provided)
	if features != nil {
		dt.Features = domain.AudioFeatures{
			Danceability:     features.Danceability,
			Energy:           features.Energy,
			Valence:          features.Valence,
			Tempo:            features.Tempo,
			Instrumentalness: features.Instrumentalness,
			Acousticness:     features.Acousticness,
		}
	}

	return dt
}

// mapPlaylistToDomain converts a raw Spotify playlist.
// Note: Tracks inside this playlist will NOT have AudioFeatures populated yet.
func mapPlaylistToDomain(sp spotifyPlaylist) domain.Playlist {
	// 1. Get the slice of wrapper objects
	items := sp.Tracks.Items
	tracks := make([]domain.Track, 0, len(items))

	// 2. Iterate over the wrappers
	for _, item := range items {
		// item.Track is the actual spotifyTrack data
		// We pass 'nil' for features because playlists don't provide them
		domainTrack := mapTrackToDomain(item.Track, nil)
		tracks = append(tracks, domainTrack)
	}

	return domain.Playlist{
		ID:     sp.ID,
		Name:   sp.Name,
		Tracks: tracks,
	}
}
