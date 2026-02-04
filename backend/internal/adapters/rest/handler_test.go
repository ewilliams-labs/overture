package rest

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"github.com/ewilliams-labs/overture/backend/internal/core/domain"
	"github.com/ewilliams-labs/overture/backend/internal/core/services"
)

// fakeSpotify records the requested ISRC and can be configured to return an error or a track.
type fakeSpotify struct {
	calledISRC string
	track      domain.Track
	err        error
}

func (f *fakeSpotify) GetTrackByISRC(ctx context.Context, isrc string) (domain.Track, error) {
	f.calledISRC = isrc
	if f.err != nil {
		return domain.Track{}, f.err
	}
	return f.track, nil
}

// fakeRepo captures GetByID and Save calls and can be configured to return errors.
type fakeRepo struct {
	gotID string
	saved domain.Playlist
	getErr error
	saveErr error
}

func (f *fakeRepo) GetByID(ctx context.Context, id string) (domain.Playlist, error) {
	f.gotID = id
	if f.getErr != nil {
		return domain.Playlist{}, f.getErr
	}
	return domain.Playlist{ID: id, Name: "test", Tracks: []domain.Track{}}, nil
}

func (f *fakeRepo) Save(ctx context.Context, p domain.Playlist) error {
	f.saved = p
	return f.saveErr
}

func TestHandler_AddTrack(t *testing.T) {
	cases := []struct {
		name       string
		body       string
		setup      func() (*services.Orchestrator, *fakeSpotify, *fakeRepo)
		wantStatus int
		verify     func(t *testing.T, fs *fakeSpotify, fr *fakeRepo)
	}{
		{
			name: "Success: valid JSON returns StatusCreated",
			body: `{"playlist_id":"pl-1","track_id":"isrc-1"}`,
			setup: func() (*services.Orchestrator, *fakeSpotify, *fakeRepo) {
				fs := &fakeSpotify{track: domain.Track{ID: "t1", ISRC: "isrc-1", Title: "Track 1"}}
				fr := &fakeRepo{}
				o := services.NewOrchestrator(fs, fr)
				return o, fs, fr
			},
			wantStatus: http.StatusCreated,
			verify: func(t *testing.T, fs *fakeSpotify, fr *fakeRepo) {
				if fs.calledISRC != "isrc-1" {
					t.Fatalf("expected spotify called with isrc 'isrc-1', got '%s'", fs.calledISRC)
				}
				if fr.gotID != "pl-1" {
					t.Fatalf("expected repo GetByID called with id 'pl-1', got '%s'", fr.gotID)
				}
				if len(fr.saved.Tracks) != 1 || fr.saved.Tracks[0].ISRC != "isrc-1" {
					t.Fatalf("expected saved playlist to contain track with ISRC 'isrc-1', got %+v", fr.saved.Tracks)
				}
			},
		},
		{
			name: "Bad Request: malformed JSON returns StatusBadRequest",
			body: "{",
			setup: func() (*services.Orchestrator, *fakeSpotify, *fakeRepo) {
				// orchestrator not needed because JSON decoding fails before use
				return nil, nil, nil
			},
			wantStatus: http.StatusBadRequest,
			verify: func(t *testing.T, fs *fakeSpotify, fr *fakeRepo) {},
		},
		{
			name: "Service Error: orchestrator returns error -> StatusInternalServerError",
			body: `{"playlist_id":"pl-2","track_id":"isrc-err"}`,
			setup: func() (*services.Orchestrator, *fakeSpotify, *fakeRepo) {
				fs := &fakeSpotify{err: errors.New("spotify not found")}
				fr := &fakeRepo{}
				o := services.NewOrchestrator(fs, fr)
				return o, fs, fr
			},
			wantStatus: http.StatusInternalServerError,
			verify: func(t *testing.T, fs *fakeSpotify, fr *fakeRepo) {
				if fs.calledISRC != "isrc-err" {
					t.Fatalf("expected spotify called with isrc 'isrc-err', got '%s'", fs.calledISRC)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var fs *fakeSpotify
			var fr *fakeRepo
			var o *services.Orchestrator
			if tc.setup != nil {
				o, fs, fr = tc.setup()
			}

			h := NewHandler(o)

			req := httptest.NewRequest("POST", "/playlists/add", strings.NewReader(tc.body))
			rec := httptest.NewRecorder()

			h.AddTrack(rec, req)

			if rec.Result().StatusCode != tc.wantStatus {
				t.Fatalf("expected status %d, got %d, body: %s", tc.wantStatus, rec.Result().StatusCode, rec.Body.String())
			}

			if tc.verify != nil {
				tc.verify(t, fs, fr)
			}
		})
	}
}