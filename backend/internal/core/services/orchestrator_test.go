package services

import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/ewilliams-labs/overture/backend/internal/core/domain"
)

// TestOrchestrator_AddTrackToPlaylist verifies AddTrackToPlaylist behavior.
func TestOrchestrator_AddTrackToPlaylist(t *testing.T) {
	type fields struct {
		spotify mockSpotify
		repo    mockRepo
	}
	tests := []struct {
		name          string
		fields        fields
		wantErr       bool
		wantSaved     bool
		wantSavedISRC string
	}{
		{
			name: "Happy Path",
			fields: fields{
				spotify: mockSpotify{
					track: domain.Track{ID: "t1", Title: "Song One", Artist: "Artist A", ISRC: "ISRC-1"},
					err:   nil,
				},
				repo: mockRepo{
					getErr:  nil,
					saveErr: nil,
				},
			},
			wantErr:       false,
			wantSaved:     true,
			wantSavedISRC: "ISRC-1",
		},
		{
			name: "Spotify error",
			fields: fields{
				spotify: mockSpotify{
					err: errors.New("spotify failure"),
				},
				repo: mockRepo{
					getErr:  nil,
					saveErr: nil,
				},
			},
			wantErr:   true,
			wantSaved: false,
		},
		{
			name: "Repository save error",
			fields: fields{
				spotify: mockSpotify{
					track: domain.Track{ID: "t2", Title: "Song Two", Artist: "Artist B", ISRC: "ISRC-2"},
					err:   nil,
				},
				repo: mockRepo{
					getErr:  nil,
					saveErr: errors.New("save failed"),
				},
			},
			wantErr:   true,
			wantSaved: false,
		},
	}

	for _, tc := range tests {
		tc := tc // capture range variable
		t.Run(tc.name, func(t *testing.T) {
			// Wire up orchestrator with pointers to the mocks in this test case
			o := &Orchestrator{
				spotify: &tc.fields.spotify,
				repo:    &tc.fields.repo,
			}

			// UPDATED: Capture both return values (playlist and error)
			_, err := o.AddTrackToPlaylist(context.Background(), "pl-1", tc.fields.spotify.track.Title, tc.fields.spotify.track.Artist)

			// Check error expectation
			if (err != nil) != tc.wantErr {
				t.Fatalf("unexpected error state: got err=%v wantErr=%v", err, tc.wantErr)
			}

			// Check persistence expectation
			if tc.wantSaved {
				if tc.fields.repo.saved == nil {
					t.Fatalf("expected playlist to be saved, but Save was not called")
				}
				found := false
				for _, tr := range tc.fields.repo.saved.Tracks {
					if tr.ISRC == tc.wantSavedISRC {
						found = true
						break
					}
				}
				if !found {
					t.Fatalf("saved playlist does not contain expected track ISRC %s", tc.wantSavedISRC)
				}
			} else {
				if tc.fields.repo.saved != nil {
					t.Fatalf("did not expect Save to be called, but it was")
				}
			}
		})
	}
}

// --- Mocks ---

// mockSpotify is a lightweight mock of the spotify provider.
type mockSpotify struct {
	track domain.Track
	err   error

	calledTitle  string
	calledArtist string
}

func (m *mockSpotify) GetTrackByMetadata(ctx context.Context, title, artist string) (domain.Track, error) {
	m.calledTitle = title
	m.calledArtist = artist
	if m.err != nil {
		return domain.Track{}, m.err
	}
	return m.track, nil
}

// AddTrackToPlaylist stub to satisfy ports.SpotifyProvider interface.
// Even if the Orchestrator doesn't call it, the interface requires it.
func (m *mockSpotify) AddTrackToPlaylist(ctx context.Context, playlistID, trackID string) (domain.Playlist, error) {
	return domain.Playlist{}, nil
}

// mockRepo is a minimal mock for PlaylistRepository.
type mockRepo struct {
	getErr   error
	saveErr  error
	playlist domain.Playlist

	called   bool
	calledID string

	saved *domain.Playlist // captured saved playlist (pointer for test inspection)
}

func (m *mockRepo) GetByID(ctx context.Context, id string) (domain.Playlist, error) {
	m.called = true
	m.calledID = id
	if m.getErr != nil {
		return domain.Playlist{}, m.getErr
	}
	if m.playlist.ID != "" {
		return m.playlist, nil
	}
	// return a valid empty playlist (struct) with the provided id
	return domain.Playlist{ID: id, Name: "Test Playlist", Tracks: []domain.Track{}}, nil
}

func (m *mockRepo) Save(ctx context.Context, p domain.Playlist) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	// capture saved playlist (store address for inspection)
	m.saved = &p
	return nil
}

func TestOrchestrator_CreatePlaylist(t *testing.T) {
	tests := []struct {
		name      string
		inputName string
		mockErr   error
		wantErr   bool
	}{
		{
			name:      "Success: valid name creates playlist",
			inputName: "My New Mix",
			mockErr:   nil,
			wantErr:   false,
		},
		{
			name:      "Validation Error: empty name",
			inputName: "",
			mockErr:   nil,
			wantErr:   true,
		},
		{
			name:      "Repo Error: save fails",
			inputName: "Database Failure",
			mockErr:   errors.New("db error"),
			wantErr:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Setup Mocks
			mockRepo := &mockRepo{saveErr: tc.mockErr}
			mockSpotify := &mockSpotify{}

			o := NewOrchestrator(mockSpotify, mockRepo)

			// Execute
			pl, err := o.CreatePlaylist(context.Background(), tc.inputName)

			// Verify Error
			if (err != nil) != tc.wantErr {
				t.Fatalf("CreatePlaylist() error = %v, wantErr %v", err, tc.wantErr)
			}

			// Verify Success State
			if !tc.wantErr {
				if pl.ID == "" {
					t.Error("Expected UUID to be generated, got empty string")
				}
				if pl.Name != tc.inputName {
					t.Errorf("Expected name %q, got %q", tc.inputName, pl.Name)
				}
				// Verify it was actually passed to the repo
				if mockRepo.saved == nil || mockRepo.saved.ID != pl.ID {
					t.Error("Repository Save() was not called with the correct playlist")
				}
			}
		})
	}
}

func TestOrchestrator_GetPlaylist(t *testing.T) {
	tests := []struct {
		name        string
		playlistID  string
		mockGetErr  error
		wantErr     bool
		wantCalled  bool
		wantIDMatch bool
	}{
		{
			name:        "Validation Error: empty id",
			playlistID:  "",
			mockGetErr:  nil,
			wantErr:     true,
			wantCalled:  false,
			wantIDMatch: false,
		},
		{
			name:        "Repo Error: get fails",
			playlistID:  "pl-1",
			mockGetErr:  errors.New("get failed"),
			wantErr:     true,
			wantCalled:  true,
			wantIDMatch: false,
		},
		{
			name:        "Success: returns playlist",
			playlistID:  "pl-2",
			mockGetErr:  nil,
			wantErr:     false,
			wantCalled:  true,
			wantIDMatch: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo := &mockRepo{getErr: tc.mockGetErr}
			mockSpotify := &mockSpotify{}

			o := NewOrchestrator(mockSpotify, mockRepo)

			pl, err := o.GetPlaylist(context.Background(), tc.playlistID)

			if (err != nil) != tc.wantErr {
				t.Fatalf("GetPlaylist() error = %v, wantErr %v", err, tc.wantErr)
			}

			if mockRepo.called != tc.wantCalled {
				t.Fatalf("GetByID() called = %v, wantCalled %v", mockRepo.called, tc.wantCalled)
			}

			if tc.wantIDMatch && pl.ID != tc.playlistID {
				t.Fatalf("expected playlist ID %q, got %q", tc.playlistID, pl.ID)
			}
		})
	}
}

func TestOrchestrator_GetPlaylistAnalysis(t *testing.T) {
	tests := []struct {
		name        string
		playlistID  string
		mockGetErr  error
		playlist    domain.Playlist
		wantErr     bool
		expected    domain.AudioFeatures
		wantCalled  bool
		wantIDMatch bool
	}{
		{
			name:       "Repo Error: get fails",
			playlistID: "pl-1",
			mockGetErr: errors.New("get failed"),
			wantErr:    true,
			wantCalled: true,
		},
		{
			name:       "Success: returns analyzed features",
			playlistID: "pl-2",
			playlist: domain.Playlist{
				ID:   "pl-2",
				Name: "Test Playlist",
				Tracks: []domain.Track{
					{
						ID: "t1",
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
						ID: "t2",
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
			},
			wantErr:     false,
			wantCalled:  true,
			wantIDMatch: true,
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

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo := &mockRepo{getErr: tc.mockGetErr, playlist: tc.playlist}
			mockSpotify := &mockSpotify{}

			o := NewOrchestrator(mockSpotify, mockRepo)

			features, err := o.GetPlaylistAnalysis(context.Background(), tc.playlistID)

			if (err != nil) != tc.wantErr {
				t.Fatalf("GetPlaylistAnalysis() error = %v, wantErr %v", err, tc.wantErr)
			}

			if mockRepo.called != tc.wantCalled {
				t.Fatalf("GetByID() called = %v, wantCalled %v", mockRepo.called, tc.wantCalled)
			}

			if tc.wantIDMatch && mockRepo.calledID != tc.playlistID {
				t.Fatalf("expected called ID %q, got %q", tc.playlistID, mockRepo.calledID)
			}

			if !tc.wantErr && !featuresEqual(features, tc.expected, 1e-9) {
				t.Fatalf("expected %+v, got %+v", tc.expected, features)
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
	return math.Abs(a-b) <= tol
}
