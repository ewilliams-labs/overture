package domain

import "errors"

// ErrDuplicateISRC is returned when attempting to add a track with a duplicate ISRC to a playlist.
var ErrDuplicateISRC = errors.New("domain: duplicate ISRC")

// Playlist represents a collection of tracks.
type Playlist struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Tracks []Track `json:"tracks"`
}

// NewPlaylist creates a new Playlist instance with the given ID and name.
// It returns an error if the ID or name are empty.
func NewPlaylist(id, name string) (*Playlist, error) {
	if id == "" || name == "" {
		return nil, errors.New("domain: invalid argument")
	}
	return &Playlist{
		ID:     id,
		Name:   name,
		Tracks: []Track{},
	}, nil
}

// AddTrack appends a track to the playlist while preventing duplicate ISRCs.
// If the incoming track has a non-empty ISRC and that ISRC already exists in
// the playlist, AddTrack returns ErrDuplicateISRC.
func (p *Playlist) AddTrack(t Track) error {
	if t.ISRC != "" {
		for _, ex := range p.Tracks {
			if ex.ISRC != "" && ex.ISRC == t.ISRC {
				return ErrDuplicateISRC
			}
		}
	}
	p.Tracks = append(p.Tracks, t)
	return nil
}

// Analyze returns the average audio features across all tracks in the playlist.
// If there are no tracks, it returns zero values.
func (p Playlist) Analyze() AudioFeatures {
	if len(p.Tracks) == 0 {
		return AudioFeatures{}
	}

	var sum AudioFeatures
	for _, tr := range p.Tracks {
		feat := tr.Features
		sum.Danceability += feat.Danceability
		sum.Energy += feat.Energy
		sum.Valence += feat.Valence
		sum.Tempo += feat.Tempo
		sum.Instrumentalness += feat.Instrumentalness
		sum.Acousticness += feat.Acousticness
	}

	count := float64(len(p.Tracks))
	return AudioFeatures{
		Danceability:     sum.Danceability / count,
		Energy:           sum.Energy / count,
		Valence:          sum.Valence / count,
		Tempo:            sum.Tempo / count,
		Instrumentalness: sum.Instrumentalness / count,
		Acousticness:     sum.Acousticness / count,
	}
}
