package spotify_test

import (
	"context"
	"errors"
	"hash/fnv"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ewilliams-labs/overture/backend/internal/adapters/spotify"
	"github.com/ewilliams-labs/overture/backend/internal/core/domain"
	"github.com/ewilliams-labs/overture/backend/internal/core/ports"
)

// --- Helpers ---

func compareTracks(t *testing.T, got, want domain.Track) {
	t.Helper()

	if got.ID != want.ID {
		t.Errorf("ID: got %v, want %v", got.ID, want.ID)
	}
	if got.Title != want.Title {
		t.Errorf("Title: got %v, want %v", got.Title, want.Title)
	}
	if got.Artist != want.Artist {
		t.Errorf("Artist: got %v, want %v", got.Artist, want.Artist)
	}
	if got.CoverURL != want.CoverURL {
		t.Errorf("CoverURL: got %v, want %v", got.CoverURL, want.CoverURL)
	}
	if got.PreviewURL != want.PreviewURL {
		t.Errorf("PreviewURL: got %v, want %v", got.PreviewURL, want.PreviewURL)
	}
	if got.ISRC != want.ISRC {
		t.Errorf("ISRC: got %v, want %v", got.ISRC, want.ISRC)
	}
	if got.DurationMs != want.DurationMs {
		t.Errorf("DurationMs: got %v, want %v", got.DurationMs, want.DurationMs)
	}

	compareFeatures(t, got.Features, want.Features)
}

func compareFeatures(t *testing.T, got, want domain.AudioFeatures) {
	t.Helper()

	if got.Energy != want.Energy {
		t.Errorf("Features.Energy: got %v, want %v", got.Energy, want.Energy)
	}
	if got.Valence != want.Valence {
		t.Errorf("Features.Valence: got %v, want %v", got.Valence, want.Valence)
	}
	if got.Danceability != want.Danceability {
		t.Errorf("Features.Danceability: got %v, want %v", got.Danceability, want.Danceability)
	}
	if got.Acousticness != want.Acousticness {
		t.Errorf("Features.Acousticness: got %v, want %v", got.Acousticness, want.Acousticness)
	}
	if got.Instrumentalness != want.Instrumentalness {
		t.Errorf("Features.Instrumentalness: got %v, want %v", got.Instrumentalness, want.Instrumentalness)
	}
	if got.Tempo != want.Tempo {
		t.Errorf("Features.Tempo: got %v, want %v", got.Tempo, want.Tempo)
	}
}

func comparePlaylists(t *testing.T, got, want domain.Playlist) {
	t.Helper()

	if got.ID != want.ID {
		t.Errorf("Playlist ID: got %v, want %v", got.ID, want.ID)
	}
	if got.Name != want.Name {
		t.Errorf("Playlist Name: got %v, want %v", got.Name, want.Name)
	}

	if len(got.Tracks) != len(want.Tracks) {
		t.Fatalf("Playlist Tracks: got %d tracks, want %d", len(got.Tracks), len(want.Tracks))
	}

	for i := range want.Tracks {
		t.Run("track_"+want.Tracks[i].ID, func(t *testing.T) {
			compareTracks(t, got.Tracks[i], want.Tracks[i])
		})
	}
}

func deterministicFeatures(trackID string) domain.AudioFeatures {
	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(trackID))
	seed := int64(hasher.Sum32())
	rng := rand.New(rand.NewSource(seed))

	between := func(min, max float64) float64 {
		return min + rng.Float64()*(max-min)
	}

	return domain.AudioFeatures{
		Energy:           between(0.1, 0.9),
		Valence:          between(0.1, 0.9),
		Danceability:     between(0.1, 0.9),
		Acousticness:     between(0.1, 0.9),
		Instrumentalness: between(0.1, 0.9),
		Tempo:            between(60.0, 180.0),
	}
}

// --- Tests ---

func TestGetTrackByMetadata(t *testing.T) {
	tests := []struct {
		name          string
		title         string
		artist        string
		response      string
		statusCode    int
		expectedTrack domain.Track
		expectErr     bool
		expectErrIs   error
	}{
		{
			name:       "successful track retrieval",
			title:      "Test Track",
			artist:     "Test Artist",
			statusCode: http.StatusOK,
			response: `{
				"tracks": {
					"items": [
						{
							"id": "1",
							"name": "Test Track",
							"duration_ms": 200000,
							"artists": [ { "name": "Test Artist" } ],
							"album": {
								"name": "Test Album",
								"images": [ { "url": "http://img.com/1.jpg" } ]
							}
						}
					]
				}
			}`,
			expectedTrack: domain.Track{
				ID:         "1",
				Title:      "Test Track",
				Artist:     "Test Artist",
				Album:      "Test Album",
				CoverURL:   "http://img.com/1.jpg",
				DurationMs: 200000,
				ISRC:       "",
				Features:   domain.AudioFeatures{},
			},
			expectErr: false,
		},
		{
			name:       "best match wins",
			title:      "Test Track",
			artist:     "Test Artist",
			statusCode: http.StatusOK,
			response: `{
				"tracks": {
					"items": [
						{
							"id": "1",
							"name": "Test Track",
							"duration_ms": 200000,
							"artists": [ { "name": "Test Artist X" } ],
							"album": {
								"name": "Test Album",
								"images": [ { "url": "http://img.com/1.jpg" } ]
							}
						},
						{
							"id": "2",
							"name": "Test Track",
							"duration_ms": 200000,
							"artists": [ { "name": "Test Artist" } ],
							"album": {
								"name": "Test Album",
								"images": [ { "url": "http://img.com/2.jpg" } ]
							}
						}
					]
				}
			}`,
			expectedTrack: domain.Track{
				ID:         "2",
				Title:      "Test Track",
				Artist:     "Test Artist",
				Album:      "Test Album",
				CoverURL:   "http://img.com/2.jpg",
				DurationMs: 200000,
				ISRC:       "",
				Features:   domain.AudioFeatures{},
			},
			expectErr: false,
		},
		{
			name:        "not found (empty items list)",
			title:       "Missing Track",
			artist:      "Missing Artist",
			statusCode:  http.StatusOK,
			response:    `{ "tracks": { "items": [] } }`,
			expectErr:   true,
			expectErrIs: ports.ErrNoConfidentMatch,
		},
		{
			name:       "top result does not match",
			title:      "Desired Track",
			artist:     "Desired Artist",
			statusCode: http.StatusOK,
			response: `{
				"tracks": {
					"items": [
						{
							"id": "2",
							"name": "Different Track",
							"duration_ms": 200000,
							"artists": [ { "name": "Other Artist" } ],
							"album": {
								"name": "Other Album",
								"images": [ { "url": "http://img.com/2.jpg" } ]
							}
						}
					]
				}
			}`,
			expectErr:   true,
			expectErrIs: ports.ErrNoConfidentMatch,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/search" {
					t.Errorf("Expected URL path /search, got %s", r.URL.Path)
				}
				query := r.URL.Query()
				expectedQuery := "track:" + strings.ToLower(tt.title) + " artist:" + strings.ToLower(tt.artist)
				if query.Get("q") != expectedQuery {
					t.Errorf("q param: got %q, want %q", query.Get("q"), expectedQuery)
				}
				if query.Get("type") != "track" {
					t.Errorf("type param: got %q, want %q", query.Get("type"), "track")
				}
				if query.Get("limit") != "5" {
					t.Errorf("limit param: got %q, want %q", query.Get("limit"), "5")
				}
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.response))
			}))
			defer ts.Close()

			client := spotify.NewClientWithBaseURL(http.DefaultClient, ts.URL)

			track, err := client.GetTrackByMetadata(context.Background(), tt.title, tt.artist)
			if (err != nil) != tt.expectErr {
				t.Errorf("expected error: %v, got: %v", tt.expectErr, err)
			}
			if tt.expectErrIs != nil && !errors.Is(err, tt.expectErrIs) {
				t.Errorf("expected error %v, got %v", tt.expectErrIs, err)
			}

			if !tt.expectErr {
				compareTracks(t, track, tt.expectedTrack)
			}
		})
	}
}

func TestAddTrackToPlaylist(t *testing.T) {
	tests := []struct {
		name             string
		playlistID       string
		trackID          string
		response         string
		statusCode       int
		expectedPlaylist domain.Playlist
		expectErr        bool
	}{
		{
			name:       "successful track addition",
			playlistID: "p1",
			trackID:    "t1",
			statusCode: http.StatusOK,
			response: `{
				"id": "p1",
				"name": "Test Playlist",
				"tracks": {
					"items": [
						{
							"track": {
								"id": "t1",
								"name": "Test Track",
								"artists": [ { "name": "Test Artist" } ],
								"album": { "name": "Test Album", "images": [] }
							}
						}
					]
				}
			}`,
			expectedPlaylist: domain.Playlist{
				ID:   "p1",
				Name: "Test Playlist",
				Tracks: []domain.Track{
					{
						ID:     "t1",
						Title:  "Test Track",
						Artist: "Test Artist",
						Album:  "Test Album",
						ISRC:   "",
					},
				},
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.response))
			}))
			defer ts.Close()

			client := spotify.NewClientWithBaseURL(http.DefaultClient, ts.URL)

			playlist, err := client.AddTrackToPlaylist(context.Background(), tt.playlistID, tt.trackID)
			if (err != nil) != tt.expectErr {
				t.Errorf("expected error: %v, got: %v", tt.expectErr, err)
			}

			if !tt.expectErr {
				comparePlaylists(t, playlist, tt.expectedPlaylist)
			}
		})
	}
}

func TestGetTrack(t *testing.T) {
	tests := []struct {
		name           string
		title          string
		artist         string
		searchStatus   int
		searchBody     string
		featuresStatus int
		featuresBody   string
		expectErr      bool
		want           domain.Track
		wantFeatures   domain.AudioFeatures
	}{
		{
			name:         "not found",
			title:        "Missing Track",
			artist:       "Missing Artist",
			searchStatus: http.StatusOK,
			searchBody:   `{ "tracks": { "items": [] } }`,
			expectErr:    true,
		},
		{
			name:         "success with features",
			title:        "Test Track",
			artist:       "Test Artist",
			searchStatus: http.StatusOK,
			searchBody: `{
				"tracks": {
					"items": [
						{
							"id": "track-1",
							"name": "Test Track",
							"duration_ms": 210000,
							"artists": [ { "name": "Test Artist" } ],
							"album": { "name": "Test Album", "images": [ { "url": "http://img.com/cover.jpg" } ] }
						}
					]
				}
			}`,
			featuresStatus: http.StatusOK,
			featuresBody: `{
				"danceability": 0.5,
				"energy": 0.7,
				"valence": 0.3,
				"tempo": 120,
				"instrumentalness": 0.2,
				"acousticness": 0.4
			}`,
			expectErr: false,
			want: domain.Track{
				ID:         "track-1",
				Title:      "Test Track",
				Artist:     "Test Artist",
				Album:      "Test Album",
				CoverURL:   "http://img.com/cover.jpg",
				DurationMs: 210000,
				ISRC:       "",
			},
			wantFeatures: domain.AudioFeatures{
				Danceability:     0.5,
				Energy:           0.7,
				Valence:          0.3,
				Tempo:            120,
				Instrumentalness: 0.2,
				Acousticness:     0.4,
			},
		},
		{
			name:         "features restricted falls back to deterministic",
			title:        "Restricted Track",
			artist:       "Test Artist",
			searchStatus: http.StatusOK,
			searchBody: `{
				"tracks": {
					"items": [
						{
							"id": "track-2",
							"name": "Restricted Track",
							"duration_ms": 180000,
							"artists": [ { "name": "Test Artist" } ],
							"album": { "name": "Test Album", "images": [] }
						}
					]
				}
			}`,
			featuresStatus: http.StatusForbidden,
			featuresBody:   `{ "error": "restricted" }`,
			expectErr:      false,
			want: domain.Track{
				ID:         "track-2",
				Title:      "Restricted Track",
				Artist:     "Test Artist",
				Album:      "Test Album",
				DurationMs: 180000,
				ISRC:       "",
			},
			wantFeatures: deterministicFeatures("track-2"),
		},
		{
			name:         "zero features fall back to deterministic",
			title:        "Zero Features",
			artist:       "Test Artist",
			searchStatus: http.StatusOK,
			searchBody: `{
				"tracks": {
					"items": [
						{
							"id": "track-3",
							"name": "Zero Features",
							"duration_ms": 150000,
							"artists": [ { "name": "Test Artist" } ],
							"album": { "name": "Test Album", "images": [] }
						}
					]
				}
			}`,
			featuresStatus: http.StatusOK,
			featuresBody: `{
				"danceability": 0,
				"energy": 0,
				"valence": 0,
				"tempo": 0,
				"instrumentalness": 0,
				"acousticness": 0
			}`,
			expectErr: false,
			want: domain.Track{
				ID:         "track-3",
				Title:      "Zero Features",
				Artist:     "Test Artist",
				Album:      "Test Album",
				DurationMs: 150000,
				ISRC:       "",
			},
			wantFeatures: deterministicFeatures("track-3"),
		},
		{
			name:         "empty energy falls back to deterministic",
			title:        "Empty Energy",
			artist:       "Test Artist",
			searchStatus: http.StatusOK,
			searchBody: `{
				"tracks": {
					"items": [
						{
							"id": "track-4",
							"name": "Empty Energy",
							"duration_ms": 160000,
							"artists": [ { "name": "Test Artist" } ],
							"album": { "name": "Test Album", "images": [] }
						}
					]
				}
			}`,
			featuresStatus: http.StatusOK,
			featuresBody: `{
				"danceability": 0.5,
				"energy": 0.0,
				"valence": 0.4,
				"tempo": 110,
				"instrumentalness": 0.2,
				"acousticness": 0.3
			}`,
			expectErr: false,
			want: domain.Track{
				ID:         "track-4",
				Title:      "Empty Energy",
				Artist:     "Test Artist",
				Album:      "Test Album",
				DurationMs: 160000,
				ISRC:       "",
			},
			wantFeatures: deterministicFeatures("track-4"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			featuresCalled := false
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case r.URL.Path == "/search":
					query := r.URL.Query()
					expectedQuery := "track:" + strings.ToLower(tt.title) + " artist:" + strings.ToLower(tt.artist)
					if query.Get("q") != expectedQuery {
						t.Errorf("q param: got %q, want %q", query.Get("q"), expectedQuery)
					}
					if query.Get("type") != "track" {
						t.Errorf("type param: got %q, want %q", query.Get("type"), "track")
					}
					if query.Get("limit") != "5" {
						t.Errorf("limit param: got %q, want %q", query.Get("limit"), "5")
					}
					w.WriteHeader(tt.searchStatus)
					w.Write([]byte(tt.searchBody))
				case r.URL.Path == "/audio-features/track-1":
					featuresCalled = true
					w.WriteHeader(tt.featuresStatus)
					w.Write([]byte(tt.featuresBody))
				case r.URL.Path == "/audio-features/track-2":
					featuresCalled = true
					w.WriteHeader(tt.featuresStatus)
					w.Write([]byte(tt.featuresBody))
				case r.URL.Path == "/audio-features/track-3":
					featuresCalled = true
					w.WriteHeader(tt.featuresStatus)
					w.Write([]byte(tt.featuresBody))
				case r.URL.Path == "/audio-features/track-4":
					featuresCalled = true
					w.WriteHeader(tt.featuresStatus)
					w.Write([]byte(tt.featuresBody))
				default:
					t.Fatalf("unexpected path: %s", r.URL.Path)
				}
			}))
			defer ts.Close()

			client := spotify.NewClientWithBaseURL(http.DefaultClient, ts.URL)

			track, err := client.GetTrack(context.Background(), tt.title, tt.artist)
			if (err != nil) != tt.expectErr {
				t.Errorf("expected error: %v, got: %v", tt.expectErr, err)
			}

			if tt.expectErr {
				if featuresCalled {
					t.Error("features endpoint should not be called on search miss")
				}
				return
			}

			want := tt.want
			want.Features = tt.wantFeatures
			compareTracks(t, track, want)
			if track.Features != tt.wantFeatures {
				t.Errorf("Features: got %+v, want %+v", track.Features, tt.wantFeatures)
			}
		})
	}
}
