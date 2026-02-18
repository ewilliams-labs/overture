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
	"time"

	"github.com/ewilliams-labs/overture/backend/internal/adapters/sqlite"
	"github.com/ewilliams-labs/overture/backend/internal/core/domain"
	"github.com/ewilliams-labs/overture/backend/internal/core/ports"
	"github.com/ewilliams-labs/overture/backend/internal/core/services"
	"github.com/ewilliams-labs/overture/backend/internal/worker"
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
	err   error
	track domain.Track
}

func (m *mockSpotify) GetTrackByMetadata(ctx context.Context, title, artist string) (domain.Track, error) {
	if m.err != nil {
		return domain.Track{}, m.err
	}
	if m.track.ID != "" {
		return m.track, nil
	}
	return domain.Track{ID: "t1", Title: title, Artist: artist, PreviewURL: "http://example.com/preview.mp3"}, nil
}

func (m *mockSpotify) GetTrack(ctx context.Context, title, artist string) (domain.Track, error) {
	return m.GetTrackByMetadata(ctx, title, artist)
}

func (m *mockSpotify) AddTrackToPlaylist(ctx context.Context, playlistID, trackID string) (domain.Playlist, error) {
	return domain.Playlist{}, nil
}

func (m *mockSpotify) GetArtistTopTracks(ctx context.Context, artistName string) ([]domain.Track, error) {
	if m.err != nil {
		return nil, m.err
	}
	return []domain.Track{m.track}, nil
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

func (m *mockRepo) AddTracksToPlaylist(ctx context.Context, playlistID string, tracks []domain.Track) error {
	if m.shouldFailSave {
		return errors.New("db error")
	}
	return nil
}

type mockIntentCompiler struct {
	intent        domain.IntentObject
	err           error
	called        bool
	calledMessage string
}

func (m *mockIntentCompiler) AnalyzeIntent(ctx context.Context, message string) (domain.IntentObject, error) {
	m.called = true
	m.calledMessage = message
	if m.err != nil {
		return domain.IntentObject{}, m.err
	}
	return m.intent, nil
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
			svc := services.NewOrchestrator(spotify, repo, nil)

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
			svc := services.NewOrchestrator(&mockSpotify{}, repo, nil)
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
			svc := services.NewOrchestrator(&mockSpotify{}, repo, nil)
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
			svc := services.NewOrchestrator(&mockSpotify{}, repo, nil)
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

func TestHandler_AnalyzeIntent(t *testing.T) {
	intent := domain.IntentObject{}
	intent.Explanation = "test"
	intent.Entities.Artists = []string{"Willie Nelson"}

	t.Run("Success: returns SSE stream with intent", func(t *testing.T) {
		compiler := &mockIntentCompiler{intent: intent}
		repo := &mockRepo{}
		svc := services.NewOrchestrator(&mockSpotify{}, repo, compiler)
		h := NewHandler(svc, nil)

		bodyBytes, _ := json.Marshal(map[string]string{"message": "Give me Willie Nelson style songs"})
		req := httptest.NewRequest(http.MethodPost, "/playlists/p1/intent", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		// SSE always returns 200 OK
		if rec.Code != http.StatusOK {
			t.Errorf("Status Code: got %d, want %d", rec.Code, http.StatusOK)
		}

		// Check Content-Type header
		contentType := rec.Header().Get("Content-Type")
		if contentType != "text/event-stream" {
			t.Errorf("Content-Type: got %q, want %q", contentType, "text/event-stream")
		}

		body := rec.Body.String()

		// Should have initial "thinking" event
		if !strings.Contains(body, "event: status") {
			t.Errorf("Response should contain 'event: status', got %q", body)
		}
		if !strings.Contains(body, "\"status\":\"thinking\"") {
			t.Errorf("Response should contain thinking status, got %q", body)
		}

		// Should have final "complete" event with intent data
		if !strings.Contains(body, "event: complete") {
			t.Errorf("Response should contain 'event: complete', got %q", body)
		}
		if !strings.Contains(body, "\"explanation\":\"test\"") {
			t.Errorf("Response should contain explanation, got %q", body)
		}

		if !compiler.called {
			t.Error("expected compiler to be called")
		}
	})

	t.Run("Bad Request: missing message", func(t *testing.T) {
		compiler := &mockIntentCompiler{intent: intent}
		repo := &mockRepo{}
		svc := services.NewOrchestrator(&mockSpotify{}, repo, compiler)
		h := NewHandler(svc, nil)

		bodyBytes, _ := json.Marshal(map[string]string{"message": ""})
		req := httptest.NewRequest(http.MethodPost, "/playlists/p1/intent", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("Status Code: got %d, want %d", rec.Code, http.StatusBadRequest)
		}
		if !strings.Contains(rec.Body.String(), "message is required") {
			t.Errorf("Response Body: got %q, want substring %q", rec.Body.String(), "message is required")
		}
	})

	t.Run("Unsupported Media Type", func(t *testing.T) {
		compiler := &mockIntentCompiler{intent: intent}
		repo := &mockRepo{}
		svc := services.NewOrchestrator(&mockSpotify{}, repo, compiler)
		h := NewHandler(svc, nil)

		bodyBytes, _ := json.Marshal(map[string]string{"message": "test"})
		req := httptest.NewRequest(http.MethodPost, "/playlists/p1/intent", bytes.NewBuffer(bodyBytes))
		// No Content-Type header
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnsupportedMediaType {
			t.Errorf("Status Code: got %d, want %d", rec.Code, http.StatusUnsupportedMediaType)
		}
		if !strings.Contains(rec.Body.String(), "Content-Type must be application/json") {
			t.Errorf("Response Body: got %q, want substring %q", rec.Body.String(), "Content-Type must be application/json")
		}
	})

	t.Run("Not Implemented: compiler missing", func(t *testing.T) {
		repo := &mockRepo{}
		svc := services.NewOrchestrator(&mockSpotify{}, repo, nil) // nil compiler
		h := NewHandler(svc, nil)

		bodyBytes, _ := json.Marshal(map[string]string{"message": "test"})
		req := httptest.NewRequest(http.MethodPost, "/playlists/p1/intent", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotImplemented {
			t.Errorf("Status Code: got %d, want %d", rec.Code, http.StatusNotImplemented)
		}
		if !strings.Contains(rec.Body.String(), "intent compiler not configured") {
			t.Errorf("Response Body: got %q, want substring %q", rec.Body.String(), "intent compiler not configured")
		}
	})

	t.Run("SSE Error: compiler failure", func(t *testing.T) {
		compiler := &mockIntentCompiler{err: errors.New("intent error")}
		repo := &mockRepo{}
		svc := services.NewOrchestrator(&mockSpotify{}, repo, compiler)
		h := NewHandler(svc, nil)

		bodyBytes, _ := json.Marshal(map[string]string{"message": "test"})
		req := httptest.NewRequest(http.MethodPost, "/playlists/p1/intent", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		// SSE returns 200 OK even for errors after stream starts
		if rec.Code != http.StatusOK {
			t.Errorf("Status Code: got %d, want %d", rec.Code, http.StatusOK)
		}

		body := rec.Body.String()

		// Should have error event
		if !strings.Contains(body, "event: error") {
			t.Errorf("Response should contain 'event: error', got %q", body)
		}
		if !strings.Contains(body, "intent error") {
			t.Errorf("Response should contain error message, got %q", body)
		}

		if !compiler.called {
			t.Error("expected compiler to be called")
		}
	})
}

func TestHandler_AsyncAudioAnalysis(t *testing.T) {
	origAnalyze := worker.AnalyzePreviewFunc
	worker.AnalyzePreviewFunc = func(url string) (float64, error) {
		return 0.95, nil
	}
	defer func() { worker.AnalyzePreviewFunc = origAnalyze }()

	// Use shared cache mode so worker goroutines see the same in-memory database
	repo, err := sqlite.NewAdapter("file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("new adapter: %v", err)
	}
	defer repo.Close()

	track := domain.Track{ID: "t-async", Title: "Blinding Lights", Artist: "The Weeknd", PreviewURL: "http://example.com/preview.mp3"}
	spotifyMock := &mockSpotify{track: track}
	svc := services.NewOrchestrator(spotifyMock, repo, nil)

	pool := worker.NewPool(repo, 1, 10)
	pool.Start(1)
	defer pool.Stop()

	h := NewHandler(svc, pool)

	playlist, err := svc.CreatePlaylist(context.Background(), "Async Test")
	if err != nil {
		t.Fatalf("create playlist: %v", err)
	}

	body, _ := json.Marshal(map[string]string{"title": "Blinding Lights", "artist": "The Weeknd"})
	req := httptest.NewRequest(http.MethodPost, "/playlists/"+playlist.ID+"/tracks", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", playlist.ID)
	rec := httptest.NewRecorder()
	h.AddTrack(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, rec.Code)
	}

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		pollReq := httptest.NewRequest(http.MethodGet, "/playlists/"+playlist.ID, nil)
		pollReq.SetPathValue("id", playlist.ID)
		pollRec := httptest.NewRecorder()
		h.GetPlaylist(pollRec, pollReq)
		if pollRec.Code != http.StatusOK {
			t.Fatalf("poll status: got %d", pollRec.Code)
		}
		var got domain.Playlist
		if err := json.NewDecoder(pollRec.Body).Decode(&got); err != nil {
			t.Fatalf("decode playlist: %v", err)
		}
		if len(got.Tracks) > 0 && got.Tracks[0].Features.Energy != 0 {
			return
		}
		time.Sleep(500 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for async audio analysis")
}
