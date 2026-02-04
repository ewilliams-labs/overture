package domain

// Track represents a musical track in the domain layer.
type Track struct {
	ID     string
	Title  string
	Artist string
	Album  string // optional
	ISRC   string // International Standard Recording Code for matching
	Vibe   map[string]float64 // audio features like "energy", "mood", etc.
}