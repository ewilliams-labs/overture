// Package domain contains the core business entities and logic for the Overture music application.
package domain

// AudioFeatures represents the audio characteristics of a track.
type AudioFeatures struct {
	// Danceability describes how suitable a track is for dancing based on a combination of musical elements including tempo, rhythm stability, beat strength, and overall regularity. A value of 0.0 is least danceable and 1.0 is most danceable.
	Danceability float64 `json:"danceability"`
	// Energy represents a perceptual measure of intensity and activity. Typically, energetic tracks feel fast, loud, and noisy. For example, death metal has high energy, while a Bach prelude scores low on the scale. Perceptual features contributing to this attribute include dynamic range, perceived loudness, timbre, onset rate, and general entropy.
	Energy float64 `json:"energy"`
	// Valence describes the musical positiveness conveyed by a track. Tracks with high valence sound more positive (e.g. happy, cheerful, euphoric), while tracks with low valence sound more negative (e.g. sad, depressed, angry).
	Valence float64 `json:"valence"`
	// Tempo is the overall estimated tempo of a track in beats per minute (BPM).
	Tempo float64 `json:"tempo"`
	// Instrumentalness predicts whether a track contains no vocals. "Ooh" and "aah" sounds are treated as instrumental in this context. Rap or spoken word tracks are clearly "vocal". The closer the instrumentalness value is to 1.0, the greater likelihood the track contains no vocal content.
	Instrumentalness float64 `json:"instrumentalness"`
	// Acousticness is a confidence measure from 0.0 to 1.0 of whether the track is acoustic. 1.0 represents high confidence the track is acoustic.
	Acousticness float64 `json:"acousticness"`
}

// Track represents a single music track.
type Track struct {
	// ID is the unique identifier for the track.
	ID string `json:"id"`
	// Title is the name of the track.
	Title string `json:"title"`
	// Artist is the name of the track's primary artist.
	Artist string `json:"artist"`
	// Album is the name of the album the track belongs to.
	Album string `json:"album"`
	// CoverURL is the URL to the album cover image.
	CoverURL string `json:"cover_url"`
	// DurationMs is the duration of the track in milliseconds.
	DurationMs int `json:"duration_ms"`
	// ISRC (International Standard Recording Code) for the track.
	ISRC string `json:"isrc"`
	// Features contains detailed audio characteristics of the track.
	Features AudioFeatures `json:"features"`
}
