package domain

type AudioFeatures struct {
	Danceability     float64 `json:"danceability"`
	Energy           float64 `json:"energy"`
	Valence          float64 `json:"valence"`
	Tempo            float64 `json:"tempo"`
	Instrumentalness float64 `json:"instrumentalness"`
	Acousticness     float64 `json:"acousticness"`
}

type Track struct {
	ID         string        `json:"id"`
	Title      string        `json:"title"`
	Artist     string        `json:"artist"`
	Album      string        `json:"album"`
	CoverURL   string        `json:"cover_url"`
	DurationMs int           `json:"duration_ms"`
	ISRC       string        `json:"isrc"`
	Features   AudioFeatures `json:"features"`
}
