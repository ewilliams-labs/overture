package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ewilliams-labs/overture/backend/internal/core/domain"
	"github.com/ewilliams-labs/overture/backend/internal/core/ports"
	"github.com/ewilliams-labs/overture/backend/internal/core/services"
)

// --- Mocks ---

// MockService satisfies the Orchestrator logic needed by the Handler.
// Note: In a real integration test, we might mock the ports, but here we mock the Service struct methods directly
// if we were using an interface. Since Orchestrator is a struct, we technically can't "mock" it easily
// without an interface.
//
// However, since we are injecting the *Service* into the Handler, and the Service is a concrete struct,
// unit testing the Handler in isolation is hard without mocking the *dependencies* of the Service.
//
// BUT, for this test to work with your current architecture (Handler -> *Service),
// we actually need to create a REAL Service with MOCK Adapters.

type mockSpotify struct {
	err error
}

func (m *mockSpotify) GetTrackByMetadata(ctx context.Context, title, artist string) (domain.Track, error) {
	if m.err != nil {
		return domain.Track{}, m.err
	}
	return domain.Track{ID: "t1", Title: title, Artist: artist}, nil
}

func (m *mockSpotify) GetTrack(ctx context.Context, title, artist string) (domain.Track, error) {
	return m.GetTrackByMetadata(ctx, title, artist)
}

func (m *mockSpotify) AddTrackToPlaylist(ctx context.Context, playlistID, trackID string) (domain.Playlist, error) {
	return domain.Playlist{}, nil
}

type mockRepo struct {
	shouldFailSave bool
	getErr         error
	playlist       domain.Playlist
	audioErr       error
	features       domain.AudioFeatures
}

func (m *mockRepo) GetByID(ctx context.Context, id string) (domain.Playlist, error) {
	if m.getErr != nil {
		return domain.Playlist{}, m.getErr
	}
	if m.playlist.ID != "" {
		return m.playlist, nil
	}
	return domain.Playlist{ID: id, Name: "Test Playlist", Tracks: []domain.Track{}}, nil
}

func (m *mockRepo) Save(ctx context.Context, p domain.Playlist) error {
	if m.shouldFailSave {
		return errors.New("db error")
	}
	return nil
}

func (m *mockRepo) GetPlaylistAudioFeatures(ctx context.Context, playlistID string) (domain.AudioFeatures, error) {
	if m.audioErr != nil {
		return domain.AudioFeatures{}, m.audioErr
	}
	return m.features, nil
}

func (m *mockRepo) UpdateTrackFeatures(ctx context.Context, trackID string, features domain.AudioFeatures) error {
	return nil
}

// --- Tests ---

func TestHandler_AddTrack(t *testing.T) {
	tests := []struct {
		name           string
		body           map[string]string // Use map to control JSON keys explicitly
		spotifyErr     error
		mockRepoFail   bool
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Success: valid JSON returns StatusCreated",
			body: map[string]string{
				"title":  "Song One", // Matches json:"title"
				"artist": "Artist A", // Matches json:"artist"
			},
			mockRepoFail:   false,
			expectedStatus: http.StatusCreated,
			expectedBody:   "\"id\":\"p1\"",
		},
		{
			name: "Bad Request: missing fields",
			body: map[string]string{
				// missing title/artist
			},
			mockRepoFail:   false,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "title and artist are required",
		},
		{
			name: "Unprocessable: no confident match",
			body: map[string]string{
				"title":  "Song One",
				"artist": "Artist A",
			},
			spotifyErr:     &ports.NoConfidentMatchError{Title: "Song One", Artist: "Artist A"},
			mockRepoFail:   false,
			expectedStatus: http.StatusUnprocessableEntity,
			expectedBody:   "\"code\":\"NO_CONFIDENT_MATCH\"",
		},
		{
			name: "Service Error: orchestrator returns error -> StatusInternalServerError",
			body: map[string]string{
				"title":  "Song One",
				"artist": "Artist A",
			},
			mockRepoFail:   true, // This triggers the error in the Service
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "service: failed to save playlist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 1. Setup Dependencies
			// Since Handler depends on concrete *Orchestrator, we build a real one with mock adapters
			spotify := &mockSpotify{err: tt.spotifyErr}
			repo := &mockRepo{shouldFailSave: tt.mockRepoFail}
			svc := services.NewOrchestrator(spotify, repo)

			// 2. Setup Handler
			h := NewHandler(svc, nil)

			// 3. Create Request
			jsonBody, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/playlists/p1/tracks", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			// 4. Execute
			h.ServeHTTP(rec, req)

			// 5. Assertions
			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d, body: %s", tt.expectedStatus, rec.Code, strings.TrimSpace(rec.Body.String()))
			}

			if tt.expectedBody != "" && !strings.Contains(rec.Body.String(), tt.expectedBody) {
				t.Errorf("expected body to contain %q, got %q", tt.expectedBody, rec.Body.String())
			}
		})
	}
}

func TestHandler_CreatePlaylist(t *testing.T) {
	tests := []struct {
		name           string
		body           map[string]string
		mockRepoFail   bool
		expectedStatus int
		expectedBody   string // substring match
	}{
		{
			name:           "Success: creates playlist",
			body:           map[string]string{"name": "Chill Vibes"},
			mockRepoFail:   false,
			expectedStatus: http.StatusCreated,
			expectedBody:   `"name":"Chill Vibes"`,
		},
		{
			name:           "Bad Request: empty name",
			body:           map[string]string{"name": ""},
			mockRepoFail:   false,
			expectedStatus: http.StatusBadRequest,                    // Service returns error for empty name
			expectedBody:   "service: playlist name cannot be empty", // Check error message
		},
		{
			name:           "Bad Request: malformed json",
			body:           nil, // Will send empty body
			mockRepoFail:   false,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid request body",
		},
		{
			name:           "Server Error: repo save fails",
			body:           map[string]string{"name": "Crash DB"},
			mockRepoFail:   true,
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "service: failed to persist new playlist",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// 1. Setup
			repo := &mockRepo{shouldFailSave: tc.mockRepoFail}
			svc := services.NewOrchestrator(&mockSpotify{}, repo)
			h := NewHandler(svc, nil)

			// 2. Request
			var bodyBytes []byte
			if tc.body != nil {
				bodyBytes, _ = json.Marshal(tc.body)
			}
			// Special case for malformed JSON test
			if tc.name == "Bad Request: malformed json" {
				bodyBytes = []byte(`{invalid-json`)
			}

			req := httptest.NewRequest(http.MethodPost, "/playlists", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			// 3. Execute
			h.ServeHTTP(rec, req)

			// 4. Verify
			if rec.Code != tc.expectedStatus {
				t.Errorf("Status Code: got %d, want %d", rec.Code, tc.expectedStatus)
			}
			if !strings.Contains(rec.Body.String(), tc.expectedBody) {
				t.Errorf("Response Body: got %q, want substring %q", rec.Body.String(), tc.expectedBody)
			}
		})
	}
}

func TestHandler_GetPlaylist(t *testing.T) {
	tests := []struct {
		name           string
		playlistID     string
		mockGetErr     error
		expectedStatus int
		expectedBody   string
		useRouter      bool
	}{
		{
			name:           "Bad Request: empty id",
			playlistID:     "",
			mockGetErr:     nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "service: playlist id cannot be empty",
			useRouter:      false,
		},
		{
			name:           "Server Error: repo get fails",
			playlistID:     "pl-1",
			mockGetErr:     errors.New("get failed"),
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "service: failed to load playlist",
			useRouter:      true,
		},
		{
			name:           "Not Found: missing playlist",
			playlistID:     "pl-404",
			mockGetErr:     domain.ErrNotFound,
			expectedStatus: http.StatusNotFound,
			expectedBody:   domain.ErrNotFound.Error(),
			useRouter:      true,
		},
		{
			name:           "Success: returns playlist",
			playlistID:     "pl-2",
			mockGetErr:     nil,
			expectedStatus: http.StatusOK,
			expectedBody:   "\"id\":\"pl-2\"",
			useRouter:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockRepo{getErr: tt.mockGetErr}
			svc := services.NewOrchestrator(&mockSpotify{}, repo)
			h := NewHandler(svc, nil)

			var req *http.Request
			if tt.useRouter {
				req = httptest.NewRequest(http.MethodGet, "/playlists/"+tt.playlistID, nil)
			} else {
				req = httptest.NewRequest(http.MethodGet, "/playlists", nil)
				req.SetPathValue("id", tt.playlistID)
			}

			rec := httptest.NewRecorder()
			if tt.useRouter {
				h.ServeHTTP(rec, req)
			} else {
				h.GetPlaylist(rec, req)
			}

			if rec.Code != tt.expectedStatus {
				t.Errorf("Status Code: got %d, want %d", rec.Code, tt.expectedStatus)
			}
			if !strings.Contains(rec.Body.String(), tt.expectedBody) {
				t.Errorf("Response Body: got %q, want substring %q", rec.Body.String(), tt.expectedBody)
			}
		})
	}
}

func TestHandler_GetPlaylistAnalysis(t *testing.T) {
	tests := []struct {
		name           string
		playlistID     string
		mockGetErr     error
		features       domain.AudioFeatures
		expectedStatus int
		expectedBody   string
		useRouter      bool
	}{
		{
			name:           "Bad Request: empty id",
			playlistID:     "",
			mockGetErr:     nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "playlist id is required",
			useRouter:      false,
		},
		{
			name:           "Not Found: missing playlist",
			playlistID:     "pl-404",
			mockGetErr:     domain.ErrNotFound,
			expectedStatus: http.StatusNotFound,
			expectedBody:   domain.ErrNotFound.Error(),
			useRouter:      true,
		},
		{
			name:           "Server Error: repo get fails",
			playlistID:     "pl-1",
			mockGetErr:     errors.New("get failed"),
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "service: failed to load playlist analysis",
			useRouter:      true,
		},
		{
			name:       "Success: returns analysis",
			playlistID: "pl-2",
			features: domain.AudioFeatures{
				Danceability:     0.5,
				Energy:           0.5,
				Valence:          0.5,
				Tempo:            110,
				Instrumentalness: 0.5,
				Acousticness:     0.5,
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "\"danceability\":0.5",
			useRouter:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockRepo{audioErr: tt.mockGetErr, features: tt.features}
			svc := services.NewOrchestrator(&mockSpotify{}, repo)
			h := NewHandler(svc, nil)

			var req *http.Request
			if tt.useRouter {
				req = httptest.NewRequest(http.MethodGet, "/playlists/"+tt.playlistID+"/analysis", nil)
			} else {
				req = httptest.NewRequest(http.MethodGet, "/playlists/analysis", nil)
				req.SetPathValue("id", tt.playlistID)
			}

			rec := httptest.NewRecorder()
			if tt.useRouter {
				h.ServeHTTP(rec, req)
			} else {
				h.GetPlaylistAnalysis(rec, req)
			}

			if rec.Code != tt.expectedStatus {
				t.Errorf("Status Code: got %d, want %d", rec.Code, tt.expectedStatus)
			}
			if !strings.Contains(rec.Body.String(), tt.expectedBody) {
				t.Errorf("Response Body: got %q, want substring %q", rec.Body.String(), tt.expectedBody)
			}
		})
	}
}
