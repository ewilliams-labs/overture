package spotify_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ewilliams-labs/overture/backend/internal/adapters/spotify"
	"github.com/ewilliams-labs/overture/backend/internal/core/domain"
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
	if got.ISRC != want.ISRC {
		t.Errorf("ISRC: got %v, want %v", got.ISRC, want.ISRC)
	}
	// Check new fields
	if got.DurationMs != want.DurationMs {
		t.Errorf("DurationMs: got %v, want %v", got.DurationMs, want.DurationMs)
	}
	// Note: We don't strictly compare CoverURL in these basic tests unless explicitly set

	compareFeatures(t, got.Features, want.Features)
}

func compareFeatures(t *testing.T, got, want domain.AudioFeatures) {
	t.Helper()
	// Compare floating point values with a small epsilon if needed,
	// but direct comparison is usually fine for test constants.
	if got.Energy != want.Energy {
		t.Errorf("Features.Energy: got %v, want %v", got.Energy, want.Energy)
	}
	if got.Valence != want.Valence {
		t.Errorf("Features.Valence: got %v, want %v", got.Valence, want.Valence)
	}
	if got.Danceability != want.Danceability {
		t.Errorf("Features.Danceability: got %v, want %v", got.Danceability, want.Danceability)
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

// --- Tests ---

func TestGetTrackByISRC(t *testing.T) {
	tests := []struct {
		name          string
		isrc          string
		response      string
		statusCode    int
		expectedTrack domain.Track
		expectErr     bool
	}{
		{
			name:       "successful track retrieval",
			isrc:       "US1234567890",
			statusCode: http.StatusOK,
			// MOCK: Search API Structure (Wrapper -> Items -> Track -> Nested Fields)
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
							},
							"external_ids": { "isrc": "US1234567890" }
						}
					]
				}
			}`,
			expectedTrack: domain.Track{
				ID:         "1",
				Title:      "Test Track",
				Artist:     "Test Artist", // Flattened
				Album:      "Test Album",  // Flattened
				CoverURL:   "http://img.com/1.jpg",
				DurationMs: 200000,
				ISRC:       "US1234567890",
				// Features are nil/empty because GetTrackByISRC (Search) doesn't return them
				Features: domain.AudioFeatures{},
			},
			expectErr: false,
		},
		{
			name:       "not found (empty items list)",
			isrc:       "INVALID",
			statusCode: http.StatusOK, // Search returns 200 OK with empty list
			response:   `{ "tracks": { "items": [] } }`,
			expectErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify we are calling the Search endpoint
				if r.URL.Path != "/search" {
					t.Errorf("Expected URL path /search, got %s", r.URL.Path)
				}
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.response))
			}))
			defer ts.Close()

			client := spotify.NewClient("test-id", "test-secret")
			// Inject the test server URL using a setter or by modifying the client struct directly if public.
			// Ideally, NewClient should accept a base URL option, OR we modify it for the test.
			// For this specific test setup to work with your current code, you might need:
			// client.SetBaseURL(ts.URL) -> (If you implemented this method)
			// OR use the constructor if it supports it.
			// Assuming you have: NewClient(httpClient, baseURL) as per previous context:
			client = spotify.NewClientWithBaseURL(http.DefaultClient, ts.URL)

			track, err := client.GetTrackByISRC(context.Background(), tt.isrc)

			if (err != nil) != tt.expectErr {
				t.Errorf("expected error: %v, got: %v", tt.expectErr, err)
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
			statusCode: http.StatusOK, // or 201 Created depending on API
			// MOCK: Playlist Structure (Tracks wrapped in Paging Object -> Items -> 'track' wrapper)
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
								"album": { "name": "Test Album", "images": [] },
								"external_ids": { "isrc": "US123" }
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
						ISRC:   "US123",
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

			// Assuming NewClientWithBaseURL exists for testing, or standard NewClient
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
		isrc           string
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
			isrc:         "BAD",
			searchStatus: http.StatusOK,
			searchBody:   `{ "tracks": { "items": [] } }`,
			expectErr:    true,
		},
		{
			name:         "success with features",
			isrc:         "US1234567890",
			searchStatus: http.StatusOK,
			searchBody: `{
				"tracks": {
					"items": [
						{
							"id": "track-1",
							"name": "Test Track",
							"duration_ms": 210000,
							"artists": [ { "name": "Test Artist" } ],
							"album": { "name": "Test Album", "images": [ { "url": "http://img.com/cover.jpg" } ] },
							"external_ids": { "isrc": "US1234567890" }
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
				ISRC:       "US1234567890",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			featuresCalled := false
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case r.URL.Path == "/search":
					query := r.URL.Query()
					if query.Get("q") != "isrc:"+tt.isrc {
						t.Errorf("q param: got %q, want %q", query.Get("q"), "isrc:"+tt.isrc)
					}
					if query.Get("type") != "track" {
						t.Errorf("type param: got %q, want %q", query.Get("type"), "track")
					}
					if query.Get("limit") != "1" {
						t.Errorf("limit param: got %q, want %q", query.Get("limit"), "1")
					}
					w.WriteHeader(tt.searchStatus)
					w.Write([]byte(tt.searchBody))
				case r.URL.Path == "/audio-features/track-1":
					featuresCalled = true
					w.WriteHeader(tt.featuresStatus)
					w.Write([]byte(tt.featuresBody))
				default:
					t.Fatalf("unexpected path: %s", r.URL.Path)
				}
			}))
			defer ts.Close()

			client := spotify.NewClientWithBaseURL(http.DefaultClient, ts.URL)

			track, err := client.GetTrack(context.Background(), tt.isrc)
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
