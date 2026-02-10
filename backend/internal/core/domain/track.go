package domain

// AudioFeatures represents the analysis data returned by Spotify.
// Reference: https://developer.spotify.com/documentation/web-api/reference/get-audio-features
type AudioFeatures struct {
	Danceability     float64 `json:"danceability"`
	Energy           float64 `json:"energy"`  // Perceptual measure of intensity/activity
	Valence          float64 `json:"valence"` // Musical positiveness (0.0 = sad, 1.0 = happy)
	Tempo            float64 `json:"tempo"`   // BPM
	Instrumentalness float64 `json:"instrumentalness"`
	Acousticness     float64 `json:"acousticness"`
}

type Track struct {
	ID         string
	Title      string // Mapped from Spotify's "name"
	Artist     string
	Album      string
	CoverURL   string
	DurationMs int           // Standard Spotify field
	ISRC       string        // Standard ID for matching
	Features   AudioFeatures // Renamed from "Vibe" to be more explicit
}
