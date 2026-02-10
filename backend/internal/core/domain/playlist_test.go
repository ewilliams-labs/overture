package domain

import (
	"errors"
	"math"
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

func TestPlaylist_Analyze(t *testing.T) {
	tests := []struct {
		name     string
		tracks   []Track
		expected AudioFeatures
		wantZero bool
	}{
		{
			name:     "returns zero values for empty playlist",
			tracks:   []Track{},
			expected: AudioFeatures{},
			wantZero: true,
		},
		{
			name: "averages features across tracks",
			tracks: []Track{
				{
					ID: "t1",
					Features: AudioFeatures{
						Danceability:     0.4,
						Energy:           0.6,
						Valence:          0.2,
						Tempo:            100,
						Instrumentalness: 0.1,
						Acousticness:     0.3,
					},
				},
				{
					ID: "t2",
					Features: AudioFeatures{
						Danceability:     0.6,
						Energy:           0.8,
						Valence:          0.4,
						Tempo:            120,
						Instrumentalness: 0.3,
						Acousticness:     0.5,
					},
				},
			},
			expected: AudioFeatures{
				Danceability:     0.5,
				Energy:           0.7,
				Valence:          0.3,
				Tempo:            110,
				Instrumentalness: 0.2,
				Acousticness:     0.4,
			},
			wantZero: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			p := Playlist{ID: "pl-1", Name: "Test", Tracks: tc.tracks}
			got := p.Analyze()

			if tc.wantZero {
				if got != (AudioFeatures{}) {
					t.Fatalf("expected zero values, got %+v", got)
				}
				return
			}

			if !featuresEqual(got, tc.expected, 1e-9) {
				t.Fatalf("expected %+v, got %+v", tc.expected, got)
			}
		})
	}
}

func featuresEqual(a, b AudioFeatures, tol float64) bool {
	return floatEquals(a.Danceability, b.Danceability, tol) &&
		floatEquals(a.Energy, b.Energy, tol) &&
		floatEquals(a.Valence, b.Valence, tol) &&
		floatEquals(a.Tempo, b.Tempo, tol) &&
		floatEquals(a.Instrumentalness, b.Instrumentalness, tol) &&
		floatEquals(a.Acousticness, b.Acousticness, tol)
}

func floatEquals(a, b, tol float64) bool {
	return math.Abs(a-b) <= tol
}
