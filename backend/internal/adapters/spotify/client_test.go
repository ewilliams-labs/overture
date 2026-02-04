package spotify_test

import (
    "context"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/ewilliams-labs/overture/backend/internal/adapters/spotify"
    "github.com/ewilliams-labs/overture/backend/internal/core/domain"
)

func compareTracks(t *testing.T, got, want domain.Track) {
    t.Helper()

    if got.ID != want.ID {
        t.Errorf("field ID: got %v, want %v", got.ID, want.ID)
    }
    if got.Title != want.Title {
        t.Errorf("field Title: got %v, want %v", got.Title, want.Title)
    }
    if got.Artist != want.Artist {
        t.Errorf("field Artist: got %v, want %v", got.Artist, want.Artist)
    }
    if got.Album != want.Album {
        t.Errorf("field Album: got %v, want %v", got.Album, want.Album)
    }
    if got.ISRC != want.ISRC {
        t.Errorf("field ISRC: got %v, want %v", got.ISRC, want.ISRC)
    }

    compareVibes(t, got.Vibe, want.Vibe)
}

func compareVibes(t *testing.T, got, want map[string]float64) {
    t.Helper()

    if len(got) != len(want) {
        t.Errorf("field Vibe: got %d entries, want %d entries", len(got), len(want))
        return
    }

    for key, wantValue := range want {
        gotValue, exists := got[key]
        if !exists {
            t.Errorf("field Vibe: missing key %q", key)
            continue
        }
        if gotValue != wantValue {
            t.Errorf("field Vibe[%q]: got %v, want %v", key, gotValue, wantValue)
        }
    }

    for key := range got {
        if _, exists := want[key]; !exists {
            t.Errorf("field Vibe: unexpected key %q", key)
        }
    }
}

func comparePlaylists(t *testing.T, got, want domain.Playlist) {
    t.Helper()

    if got.ID != want.ID {
        t.Errorf("field ID: got %v, want %v", got.ID, want.ID)
    }
    if got.Name != want.Name {
        t.Errorf("field Name: got %v, want %v", got.Name, want.Name)
    }

    if len(got.Tracks) != len(want.Tracks) {
        t.Errorf("field Tracks: got %d tracks, want %d tracks", len(got.Tracks), len(want.Tracks))
        return
    }

    for i := range want.Tracks {
        t.Run("track_"+string(rune(i)), func(t *testing.T) {
            compareTracks(t, got.Tracks[i], want.Tracks[i])
        })
    }
}

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
            response:   `{"id":"1","title":"Test Track","artist":"Test Artist","isrc":"US1234567890"}`,
            expectedTrack: domain.Track{
                ID:     "1",
                Title:  "Test Track",
                Artist: "Test Artist",
                ISRC:   "US1234567890",
            },
            expectErr: false,
        },
        {
            name:       "not found",
            isrc:       "INVALID",
            statusCode: http.StatusNotFound,
            response:   `{}`,
            expectErr:  true,
        },
        {
            name:       "track with vibe metadata",
            isrc:       "US9876543210",
            statusCode: http.StatusOK,
            response:   `{"id":"2","title":"Vibed Track","artist":"Vibe Artist","isrc":"US9876543210","vibe":{"energy":0.8,"danceability":0.75}}`,
            expectedTrack: domain.Track{
                ID:     "2",
                Title:  "Vibed Track",
                Artist: "Vibe Artist",
                ISRC:   "US9876543210",
                Vibe: map[string]float64{
                    "energy":       0.8,
                    "danceability": 0.75,
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

            client := spotify.NewClient(http.DefaultClient, ts.URL)
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
            playlistID: "playlist1",
            trackID:    "track1",
            statusCode: http.StatusOK,
            response:   `{"id":"playlist1","name":"Test Playlist","tracks":[{"id":"track1","title":"Test Track","artist":"Test Artist","isrc":"US1234567890"}]}`,
            expectedPlaylist: domain.Playlist{
                ID:   "playlist1",
                Name: "Test Playlist",
                Tracks: []domain.Track{
                    {
                        ID:     "track1",
                        Title:  "Test Track",
                        Artist: "Test Artist",
                        ISRC:   "US1234567890",
                    },
                },
            },
            expectErr: false,
        },
        {
            name:       "playlist not found",
            playlistID: "invalid",
            trackID:    "track1",
            statusCode: http.StatusNotFound,
            response:   `{}`,
            expectErr:  true,
        },
        {
            name:       "playlist with multiple tracks",
            playlistID: "playlist2",
            trackID:    "track3",
            statusCode: http.StatusCreated,
            response:   `{"id":"playlist2","name":"Multi Track Playlist","tracks":[{"id":"track1","title":"Track One","artist":"Artist One","isrc":"US1111111111"},{"id":"track2","title":"Track Two","artist":"Artist Two","isrc":"US2222222222"},{"id":"track3","title":"Track Three","artist":"Artist Three","isrc":"US3333333333"}]}`,
            expectedPlaylist: domain.Playlist{
                ID:   "playlist2",
                Name: "Multi Track Playlist",
                Tracks: []domain.Track{
                    {
                        ID:     "track1",
                        Title:  "Track One",
                        Artist: "Artist One",
                        ISRC:   "US1111111111",
                    },
                    {
                        ID:     "track2",
                        Title:  "Track Two",
                        Artist: "Artist Two",
                        ISRC:   "US2222222222",
                    },
                    {
                        ID:     "track3",
                        Title:  "Track Three",
                        Artist: "Artist Three",
                        ISRC:   "US3333333333",
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

            client := spotify.NewClient(http.DefaultClient, ts.URL)
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