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

type mockSpotify struct{}

func (m *mockSpotify) GetTrackByISRC(ctx context.Context, isrc string) (domain.Track, error) {
	return domain.Track{ID: "t1", Title: "Test Song", ISRC: isrc}, nil
}

func (m *mockSpotify) AddTrackToPlaylist(ctx context.Context, playlistID, trackID string) (domain.Playlist, error) {
	return domain.Playlist{}, nil
}

type mockRepo struct {
	shouldFailSave bool
}

func (m *mockRepo) GetByID(ctx context.Context, id string) (domain.Playlist, error) {
	return domain.Playlist{ID: id, Name: "Test Playlist", Tracks: []domain.Track{}}, nil
}

func (m *mockRepo) Save(ctx context.Context, p domain.Playlist) error {
	if m.shouldFailSave {
		return errors.New("db error")
	}
	return nil
}

// --- Tests ---

func TestHandler_AddTrack(t *testing.T) {
	tests := []struct {
		name           string
		body           map[string]string // Use map to control JSON keys explicitly
		mockRepoFail   bool
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Success: valid JSON returns StatusCreated",
			body: map[string]string{
				"playlist_id": "p1",      // Matches json:"playlist_id"
				"isrc":        "US12345", // Matches json:"isrc"
			},
			mockRepoFail:   false,
			expectedStatus: http.StatusCreated,
			expectedBody:   "", // We check if it contains JSON, not exact match
		},
		{
			name: "Bad Request: missing fields",
			body: map[string]string{
				"playlist_id": "p1",
				// missing isrc
			},
			mockRepoFail:   false,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "playlist_id and isrc are required",
		},
		{
			name: "Service Error: orchestrator returns error -> StatusInternalServerError",
			body: map[string]string{
				"playlist_id": "p1",
				"isrc":        "US12345",
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
			spotify := &mockSpotify{}
			repo := &mockRepo{shouldFailSave: tt.mockRepoFail}
			svc := services.NewOrchestrator(spotify, repo)

			// 2. Setup Handler
			h := NewHandler(svc)

			// 3. Create Request
			jsonBody, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/tracks", bytes.NewBuffer(jsonBody))
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
