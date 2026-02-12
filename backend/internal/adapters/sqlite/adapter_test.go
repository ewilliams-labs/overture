package sqlite

import (
	"context"
	"errors"
	"testing"

	"github.com/ewilliams-labs/overture/backend/internal/core/domain"
)

func TestAdapter_GetByID(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(t *testing.T, a *Adapter) string
		wantErr    error
		wantID     string
		wantName   string
		wantTracks int
	}{
		{
			name: "not found",
			setup: func(t *testing.T, a *Adapter) string {
				return "missing"
			},
			wantErr: domain.ErrNotFound,
		},
		{
			name: "returns playlist with tracks",
			setup: func(t *testing.T, a *Adapter) string {
				p := domain.Playlist{
					ID:   "pl-1",
					Name: "Test Playlist",
					Tracks: []domain.Track{
						{
							ID:         "t1",
							Title:      "Song One",
							Artist:     "Artist A",
							Album:      "Album A",
							DurationMs: 123000,
							ISRC:       "ISRC-1",
							CoverURL:   "https://img.test/1.jpg",
							Features: domain.AudioFeatures{
								Danceability:     0.25,
								Energy:           0.5,
								Valence:          0.75,
								Tempo:            120,
								Instrumentalness: 0.1,
								Acousticness:     0.2,
							},
						},
					},
				}
				if err := a.Save(context.Background(), p); err != nil {
					t.Fatalf("save playlist: %v", err)
				}
				return p.ID
			},
			wantID:     "pl-1",
			wantName:   "Test Playlist",
			wantTracks: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, err := NewAdapter(":memory:")
			if err != nil {
				t.Fatalf("new adapter: %v", err)
			}
			defer a.Close()

			playlistID := tt.setup(t, a)
			got, err := a.GetByID(context.Background(), playlistID)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.ID != tt.wantID {
				t.Fatalf("id: got %q, want %q", got.ID, tt.wantID)
			}
			if got.Name != tt.wantName {
				t.Fatalf("name: got %q, want %q", got.Name, tt.wantName)
			}
			if len(got.Tracks) != tt.wantTracks {
				t.Fatalf("tracks: got %d, want %d", len(got.Tracks), tt.wantTracks)
			}
			if tt.wantTracks > 0 {
				track := got.Tracks[0]
				if track.ID == "" || track.Title == "" || track.Artist == "" {
					t.Fatalf("track fields not populated: %+v", track)
				}
				if track.Features.Danceability == 0 && track.Features.Energy == 0 {
					t.Fatalf("track features not populated: %+v", track.Features)
				}
			}
		})
	}
}

func TestAdapter_GetPlaylistAudioFeatures(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T, a *Adapter) string
		wantErr  error
		expected domain.AudioFeatures
	}{
		{
			name: "not found",
			setup: func(t *testing.T, a *Adapter) string {
				return "missing"
			},
			wantErr: domain.ErrNotFound,
		},
		{
			name: "returns averaged features",
			setup: func(t *testing.T, a *Adapter) string {
				p := domain.Playlist{
					ID:   "pl-avg",
					Name: "Avg Playlist",
					Tracks: []domain.Track{
						{
							ID:     "t1",
							Title:  "Song One",
							Artist: "Artist A",
							Features: domain.AudioFeatures{
								Danceability:     0.2,
								Energy:           0.4,
								Valence:          0.6,
								Tempo:            100,
								Instrumentalness: 0.1,
								Acousticness:     0.3,
							},
						},
						{
							ID:     "t2",
							Title:  "Song Two",
							Artist: "Artist B",
							Features: domain.AudioFeatures{
								Danceability:     0.6,
								Energy:           0.8,
								Valence:          0.2,
								Tempo:            120,
								Instrumentalness: 0.3,
								Acousticness:     0.5,
							},
						},
					},
				}
				if err := a.Save(context.Background(), p); err != nil {
					t.Fatalf("save playlist: %v", err)
				}
				return p.ID
			},
			expected: domain.AudioFeatures{
				Danceability:     0.4,
				Energy:           0.6,
				Valence:          0.4,
				Tempo:            110,
				Instrumentalness: 0.2,
				Acousticness:     0.4,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, err := NewAdapter(":memory:")
			if err != nil {
				t.Fatalf("new adapter: %v", err)
			}
			defer a.Close()

			playlistID := tt.setup(t, a)
			got, err := a.GetPlaylistAudioFeatures(context.Background(), playlistID)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !featuresEqual(got, tt.expected, 1e-9) {
				t.Fatalf("expected %+v, got %+v", tt.expected, got)
			}
		})
	}
}

func featuresEqual(a, b domain.AudioFeatures, tol float64) bool {
	return floatEquals(a.Danceability, b.Danceability, tol) &&
		floatEquals(a.Energy, b.Energy, tol) &&
		floatEquals(a.Valence, b.Valence, tol) &&
		floatEquals(a.Tempo, b.Tempo, tol) &&
		floatEquals(a.Instrumentalness, b.Instrumentalness, tol) &&
		floatEquals(a.Acousticness, b.Acousticness, tol)
}

func floatEquals(a, b, tol float64) bool {
	if a == b {
		return true
	}
	if a > b {
		return a-b <= tol
	}
	return b-a <= tol
}
