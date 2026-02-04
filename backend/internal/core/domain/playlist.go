package domain

import "errors"

var ErrDuplicateISRC = errors.New("domain: duplicate ISRC")

type Playlist struct {
    ID     string
    Name   string
    Tracks []Track
}

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
