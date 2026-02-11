package spotify

import "testing"

func TestLevenshteinDistance(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
		want int
	}{
		{
			name: "kitten sitting",
			a:    "kitten",
			b:    "sitting",
			want: 3,
		},
		{
			name: "empty to word",
			a:    "",
			b:    "sound",
			want: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := levenshteinDistance(tt.a, tt.b)
			if got != tt.want {
				t.Fatalf("distance: got %d, want %d", got, tt.want)
			}
		})
	}
}

func TestTrackMatchScore(t *testing.T) {
	tests := []struct {
		name   string
		title  string
		artist string
		track  spotifyTrack
		wantOK bool
	}{
		{
			name:   "matches remastered title",
			title:  "Happy",
			artist: "Pharrell Williams",
			track: spotifyTrack{
				Name: "Happy (Remastered 2014)",
				Artists: []struct {
					Name string `json:"name"`
				}{
					{Name: "Pharrell Williams"},
				},
			},
			wantOK: true,
		},
		{
			name:   "rejects different track",
			title:  "Happy",
			artist: "Pharrell Williams",
			track: spotifyTrack{
				Name: "Sad Song",
				Artists: []struct {
					Name string `json:"name"`
				}{
					{Name: "Other Artist"},
				},
			},
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, got := trackMatchScore(tt.title, tt.artist, tt.track)
			if got != tt.wantOK {
				t.Fatalf("match: got %v, want %v", got, tt.wantOK)
			}
		})
	}
}
