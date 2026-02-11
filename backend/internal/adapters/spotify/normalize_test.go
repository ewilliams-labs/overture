package spotify

import "testing"

func TestNormalizeSearchInput(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "strips remastered and punctuation",
			input: "Blinding Lights (Remastered 2020)",
			want:  "blinding lights",
		},
		{
			name:  "strips live suffix",
			input: "Song Title - Live",
			want:  "song title",
		},
		{
			name:  "keeps digits",
			input: "Symphony No. 5",
			want:  "symphony no 5",
		},
		{
			name:  "removes feat tokens",
			input: "Artist feat. Someone",
			want:  "artist someone",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeSearchInput(tt.input)
			if got != tt.want {
				t.Fatalf("normalizeSearchInput: got %q, want %q", got, tt.want)
			}
		})
	}
}
