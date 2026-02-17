package services

import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/ewilliams-labs/overture/backend/internal/core/domain"
	"github.com/ewilliams-labs/overture/backend/internal/core/ports"
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
		wantErrIs     error
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
			name: "Spotify no confident match",
			fields: fields{
				spotify: mockSpotify{
					err: ports.ErrNoConfidentMatch,
				},
				repo: mockRepo{
					getErr:  nil,
					saveErr: nil,
				},
			},
			wantErr:   true,
			wantErrIs: ports.ErrNoConfidentMatch,
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
				intent:  nil,
			}

			playlistID, trackID, _, err := o.AddTrackToPlaylist(context.Background(), "pl-1", tc.fields.spotify.track.Title, tc.fields.spotify.track.Artist)

			// Check error expectation
			if (err != nil) != tc.wantErr {
				t.Fatalf("unexpected error state: got err=%v wantErr=%v", err, tc.wantErr)
			}
			if tc.wantErrIs != nil && !errors.Is(err, tc.wantErrIs) {
				t.Fatalf("expected error %v, got %v", tc.wantErrIs, err)
			}

			if !tc.wantErr && playlistID != "pl-1" {
				t.Fatalf("expected playlist id %q, got %q", "pl-1", playlistID)
			}
			if !tc.wantErr && tc.fields.spotify.track.ID != "" && trackID != tc.fields.spotify.track.ID {
				t.Fatalf("expected track id %q, got %q", tc.fields.spotify.track.ID, trackID)
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

func (m *mockSpotify) GetTrack(ctx context.Context, title, artist string) (domain.Track, error) {
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

// GetArtistTopTracks stub to satisfy ports.SpotifyProvider interface.
func (m *mockSpotify) GetArtistTopTracks(ctx context.Context, artistName string) ([]domain.Track, error) {
	if m.err != nil {
		return nil, m.err
	}
	return []domain.Track{m.track}, nil
}

// mockRepo is a minimal mock for PlaylistRepository.
type mockRepo struct {
	getErr   error
	saveErr  error
	playlist domain.Playlist
	audioErr error
	features domain.AudioFeatures

	called        bool
	calledID      string
	calledAudio   bool
	calledAudioID string

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

func (m *mockRepo) GetPlaylistAudioFeatures(ctx context.Context, playlistID string) (domain.AudioFeatures, error) {
	m.calledAudio = true
	m.calledAudioID = playlistID
	if m.audioErr != nil {
		return domain.AudioFeatures{}, m.audioErr
	}
	return m.features, nil
}

func (m *mockRepo) UpdateTrackFeatures(ctx context.Context, trackID string, features domain.AudioFeatures) error {
	return nil
}

func (m *mockRepo) AddTracksToPlaylist(ctx context.Context, playlistID string, tracks []domain.Track) error {
	if m.saveErr != nil {
		return m.saveErr
	}
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

			o := NewOrchestrator(mockSpotify, mockRepo, nil)

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

			o := NewOrchestrator(mockSpotify, mockRepo, nil)

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
		features    domain.AudioFeatures
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
			features: domain.AudioFeatures{
				Danceability:     0.4,
				Energy:           0.6,
				Valence:          0.4,
				Tempo:            110,
				Instrumentalness: 0.2,
				Acousticness:     0.4,
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
			mockRepo := &mockRepo{audioErr: tc.mockGetErr, features: tc.features}
			mockSpotify := &mockSpotify{}

			o := NewOrchestrator(mockSpotify, mockRepo, nil)

			features, err := o.GetPlaylistAnalysis(context.Background(), tc.playlistID)

			if (err != nil) != tc.wantErr {
				t.Fatalf("GetPlaylistAnalysis() error = %v, wantErr %v", err, tc.wantErr)
			}

			if mockRepo.calledAudio != tc.wantCalled {
				t.Fatalf("GetPlaylistAudioFeatures() called = %v, wantCalled %v", mockRepo.calledAudio, tc.wantCalled)
			}

			if tc.wantIDMatch && mockRepo.calledAudioID != tc.playlistID {
				t.Fatalf("expected called ID %q, got %q", tc.playlistID, mockRepo.calledAudioID)
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

// mockIntentCompiler is a mock implementation of ports.IntentCompiler.
type mockIntentCompiler struct {
	intent domain.IntentObject
	err    error
	called bool
}

func (m *mockIntentCompiler) AnalyzeIntent(ctx context.Context, message string) (domain.IntentObject, error) {
	m.called = true
	if m.err != nil {
		return domain.IntentObject{}, m.err
	}
	return m.intent, nil
}

func TestOrchestrator_ProcessIntent(t *testing.T) {
	tests := []struct {
		name       string
		compiler   *mockIntentCompiler
		message    string
		wantErr    bool
		wantCalled bool
	}{
		{
			name: "Success: returns intent",
			compiler: &mockIntentCompiler{
				intent: domain.IntentObject{Explanation: "test explanation"},
			},
			message:    "Give me some chill vibes",
			wantErr:    false,
			wantCalled: true,
		},
		{
			name:       "Error: compiler not configured",
			compiler:   nil,
			message:    "test",
			wantErr:    true,
			wantCalled: false,
		},
		{
			name: "Error: compiler returns error",
			compiler: &mockIntentCompiler{
				err: errors.New("analysis failed"),
			},
			message:    "test",
			wantErr:    true,
			wantCalled: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mockRepo := &mockRepo{}
			mockSpotify := &mockSpotify{}

			var compiler ports.IntentCompiler
			if tc.compiler != nil {
				compiler = tc.compiler
			}

			o := NewOrchestrator(mockSpotify, mockRepo, compiler)

			result, err := o.ProcessIntent(context.Background(), "test-playlist-id", tc.message)

			if (err != nil) != tc.wantErr {
				t.Fatalf("ProcessIntent() error = %v, wantErr %v", err, tc.wantErr)
			}

			if tc.compiler != nil && tc.compiler.called != tc.wantCalled {
				t.Fatalf("expected called=%v, got %v", tc.wantCalled, tc.compiler.called)
			}

			if !tc.wantErr && tc.compiler != nil && result.Intent.Explanation != tc.compiler.intent.Explanation {
				t.Fatalf("expected explanation %q, got %q", tc.compiler.intent.Explanation, result.Intent.Explanation)
			}
		})
	}
}

func TestOrchestrator_HasIntentCompiler(t *testing.T) {
	t.Run("returns true when compiler is set", func(t *testing.T) {
		compiler := &mockIntentCompiler{}
		o := NewOrchestrator(&mockSpotify{}, &mockRepo{}, compiler)

		if !o.HasIntentCompiler() {
			t.Error("expected HasIntentCompiler to return true")
		}
	})

	t.Run("returns false when compiler is nil", func(t *testing.T) {
		o := NewOrchestrator(&mockSpotify{}, &mockRepo{}, nil)

		if o.HasIntentCompiler() {
			t.Error("expected HasIntentCompiler to return false")
		}
	})
}

func TestMatchesConstraints(t *testing.T) {
	tests := []struct {
		name        string
		features    domain.AudioFeatures
		constraints domain.IntentObject
		want        bool
	}{
		{
			name: "all constraints nil - passes",
			features: domain.AudioFeatures{
				Energy:           0.8,
				Valence:          0.6,
				Acousticness:     0.3,
				Instrumentalness: 0.1,
			},
			constraints: domain.IntentObject{},
			want:        true,
		},
		{
			name: "energy within range - passes",
			features: domain.AudioFeatures{
				Energy: 0.7,
			},
			constraints: domain.IntentObject{
				VibeConstraints: struct {
					Energy     *domain.VibeConstraint `json:"energy,omitempty"`
					Valence    *domain.VibeConstraint `json:"valence,omitempty"`
					Acoustic   *domain.VibeConstraint `json:"acousticness,omitempty"`
					Instrument *domain.VibeConstraint `json:"instrumentalness,omitempty"`
				}{
					Energy: &domain.VibeConstraint{Min: 0.5, Max: 0.9},
				},
			},
			want: true,
		},
		{
			name: "energy below range - fails",
			features: domain.AudioFeatures{
				Energy: 0.3,
			},
			constraints: domain.IntentObject{
				VibeConstraints: struct {
					Energy     *domain.VibeConstraint `json:"energy,omitempty"`
					Valence    *domain.VibeConstraint `json:"valence,omitempty"`
					Acoustic   *domain.VibeConstraint `json:"acousticness,omitempty"`
					Instrument *domain.VibeConstraint `json:"instrumentalness,omitempty"`
				}{
					Energy: &domain.VibeConstraint{Min: 0.5, Max: 0.9},
				},
			},
			want: false,
		},
		{
			name: "energy above range - fails",
			features: domain.AudioFeatures{
				Energy: 0.95,
			},
			constraints: domain.IntentObject{
				VibeConstraints: struct {
					Energy     *domain.VibeConstraint `json:"energy,omitempty"`
					Valence    *domain.VibeConstraint `json:"valence,omitempty"`
					Acoustic   *domain.VibeConstraint `json:"acousticness,omitempty"`
					Instrument *domain.VibeConstraint `json:"instrumentalness,omitempty"`
				}{
					Energy: &domain.VibeConstraint{Min: 0.5, Max: 0.9},
				},
			},
			want: false,
		},
		{
			name: "constraint with zero bounds - skipped",
			features: domain.AudioFeatures{
				Energy: 0.1, // Would fail if constraint was checked
			},
			constraints: domain.IntentObject{
				VibeConstraints: struct {
					Energy     *domain.VibeConstraint `json:"energy,omitempty"`
					Valence    *domain.VibeConstraint `json:"valence,omitempty"`
					Acoustic   *domain.VibeConstraint `json:"acousticness,omitempty"`
					Instrument *domain.VibeConstraint `json:"instrumentalness,omitempty"`
				}{
					Energy: &domain.VibeConstraint{Min: 0, Max: 0},
				},
			},
			want: true,
		},
		{
			name: "multiple constraints all pass",
			features: domain.AudioFeatures{
				Energy:           0.7,
				Valence:          0.5,
				Acousticness:     0.2,
				Instrumentalness: 0.8,
			},
			constraints: domain.IntentObject{
				VibeConstraints: struct {
					Energy     *domain.VibeConstraint `json:"energy,omitempty"`
					Valence    *domain.VibeConstraint `json:"valence,omitempty"`
					Acoustic   *domain.VibeConstraint `json:"acousticness,omitempty"`
					Instrument *domain.VibeConstraint `json:"instrumentalness,omitempty"`
				}{
					Energy:     &domain.VibeConstraint{Min: 0.5, Max: 0.9},
					Valence:    &domain.VibeConstraint{Min: 0.3, Max: 0.7},
					Acoustic:   &domain.VibeConstraint{Min: 0.0, Max: 0.5},
					Instrument: &domain.VibeConstraint{Min: 0.6, Max: 1.0},
				},
			},
			want: true,
		},
		{
			name: "multiple constraints one fails",
			features: domain.AudioFeatures{
				Energy:           0.7,
				Valence:          0.1, // Below range
				Acousticness:     0.2,
				Instrumentalness: 0.8,
			},
			constraints: domain.IntentObject{
				VibeConstraints: struct {
					Energy     *domain.VibeConstraint `json:"energy,omitempty"`
					Valence    *domain.VibeConstraint `json:"valence,omitempty"`
					Acoustic   *domain.VibeConstraint `json:"acousticness,omitempty"`
					Instrument *domain.VibeConstraint `json:"instrumentalness,omitempty"`
				}{
					Energy:     &domain.VibeConstraint{Min: 0.5, Max: 0.9},
					Valence:    &domain.VibeConstraint{Min: 0.3, Max: 0.7},
					Acoustic:   &domain.VibeConstraint{Min: 0.0, Max: 0.5},
					Instrument: &domain.VibeConstraint{Min: 0.6, Max: 1.0},
				},
			},
			want: false,
		},
		{
			name: "value at boundary min - passes",
			features: domain.AudioFeatures{
				Energy: 0.5,
			},
			constraints: domain.IntentObject{
				VibeConstraints: struct {
					Energy     *domain.VibeConstraint `json:"energy,omitempty"`
					Valence    *domain.VibeConstraint `json:"valence,omitempty"`
					Acoustic   *domain.VibeConstraint `json:"acousticness,omitempty"`
					Instrument *domain.VibeConstraint `json:"instrumentalness,omitempty"`
				}{
					Energy: &domain.VibeConstraint{Min: 0.5, Max: 0.9},
				},
			},
			want: true,
		},
		{
			name: "value at boundary max - passes",
			features: domain.AudioFeatures{
				Energy: 0.9,
			},
			constraints: domain.IntentObject{
				VibeConstraints: struct {
					Energy     *domain.VibeConstraint `json:"energy,omitempty"`
					Valence    *domain.VibeConstraint `json:"valence,omitempty"`
					Acoustic   *domain.VibeConstraint `json:"acousticness,omitempty"`
					Instrument *domain.VibeConstraint `json:"instrumentalness,omitempty"`
				}{
					Energy: &domain.VibeConstraint{Min: 0.5, Max: 0.9},
				},
			},
			want: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := matchesConstraints(tc.features, tc.constraints)
			if got != tc.want {
				t.Errorf("matchesConstraints() = %v, want %v", got, tc.want)
			}
		})
	}
}
