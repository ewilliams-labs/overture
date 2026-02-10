package services

import (
	"context"
	"errors"
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
			_, err := o.AddTrackToPlaylist(context.Background(), "pl-1", tc.fields.spotify.track.ISRC)

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

	calledISRC string
}

func (m *mockSpotify) GetTrackByISRC(ctx context.Context, isrc string) (domain.Track, error) {
	m.calledISRC = isrc
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
	getErr  error
	saveErr error

	saved *domain.Playlist // captured saved playlist (pointer for test inspection)
}

func (m *mockRepo) GetByID(ctx context.Context, id string) (domain.Playlist, error) {
	if m.getErr != nil {
		return domain.Playlist{}, m.getErr
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
