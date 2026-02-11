package spotify

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

// addTrackRequest represents the request body for adding a track to a playlist.
type addTrackRequest struct {
	Uris []string `json:"uris"`
}
