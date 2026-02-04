package domain

import (
    "errors"
    "reflect"
    "testing"
)

func TestPlaylist_AddTrack(t *testing.T) {
    tests := []struct {
        name          string
        initialTracks []Track
        toAdd         Track
        wantErr       error
        wantLen       int
    }{
        {
            name:          "adds new track successfully",
            initialTracks: []Track{},
            toAdd:         Track{ID: "t1", Title: "Song One", Artist: "Artist A", ISRC: "ISRC-1"},
            wantErr:       nil,
            wantLen:       1,
        },
        {
            name: "fails when adding track with duplicate ISRC",
            initialTracks: []Track{
                {ID: "t_existing", Title: "Existing", Artist: "Artist A", ISRC: "ISRC-1"},
            },
            toAdd:   Track{ID: "t2", Title: "Song Two", Artist: "Artist B", ISRC: "ISRC-1"},
            wantErr: ErrDuplicateISRC,
            wantLen: 1,
        },
    }

    for _, tc := range tests {
        tc := tc // capture range variable
        t.Run(tc.name, func(t *testing.T) {
            p, err := NewPlaylist("pl-1", "Test Playlist")
            if err != nil {
                t.Fatalf("failed to create playlist: %v", err)
            }
            // seed initial tracks directly
            p.Tracks = append(p.Tracks, tc.initialTracks...)

            err = p.AddTrack(tc.toAdd)
            if tc.wantErr == nil {
                if err != nil {
                    t.Fatalf("expected no error, got: %v", err)
                }
            } else {
                if !errors.Is(err, tc.wantErr) {
                    t.Fatalf("expected error %v, got %v", tc.wantErr, err)
                }
            }

            if got := len(p.Tracks); got != tc.wantLen {
                t.Fatalf("expected %d tracks, got %d", tc.wantLen, got)
            }

            if tc.wantErr == nil {
                last := p.Tracks[len(p.Tracks)-1]
                if !reflect.DeepEqual(last, tc.toAdd) {
                    t.Fatalf("last track mismatch: want %+v, got %+v", tc.toAdd, last)
                }
            }
        })
    }
}