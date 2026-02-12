package spotify

import (
	"math"
	"testing"
)

func TestNormalize(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "exact",
			input: "Hello",
			want:  "hello",
		},
		{
			name:  "dash suffix",
			input: "Track - Remastered 2011",
			want:  "track",
		},
		{
			name:  "bracket suffix",
			input: "Song (Live)",
			want:  "song",
		},
		{
			name:  "punctuation",
			input: "AC/DC",
			want:  "ac dc",
		},
		{
			name:  "not suffix",
			input: "Live Forever",
			want:  "live forever",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Normalize(tt.input)
			if got != tt.want {
				t.Fatalf("Normalize(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

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

func TestScoreResult(t *testing.T) {
	tests := []struct {
		name          string
		targetArtist  string
		targetTitle   string
		actualArtist  string
		actualTitle   string
		want          float64
		wantTolerance float64
		min           float64
		max           float64
	}{
		{
			name:          "exact match",
			targetArtist:  "Radiohead",
			targetTitle:   "Creep",
			actualArtist:  "Radiohead",
			actualTitle:   "Creep",
			want:          1.0,
			wantTolerance: 0.0001,
		},
		{
			name:          "case mismatch",
			targetArtist:  "RADIOHEAD",
			targetTitle:   "CREEP",
			actualArtist:  "radiohead",
			actualTitle:   "creep",
			want:          1.0,
			wantTolerance: 0.0001,
		},
		{
			name:         "major mismatch",
			targetArtist: "Radiohead",
			targetTitle:  "Creep",
			actualArtist: "Taylor Swift",
			actualTitle:  "Love Story",
			max:          0.4,
		},
		{
			name:         "suffix mismatch",
			targetArtist: "Radiohead",
			targetTitle:  "Creep",
			actualArtist: "Radiohead",
			actualTitle:  "Creep - Remastered 2009",
			min:          0.9,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ScoreResult(tt.targetArtist, tt.targetTitle, tt.actualArtist, tt.actualTitle)
			if tt.wantTolerance > 0 {
				if math.Abs(got-tt.want) > tt.wantTolerance {
					t.Fatalf("ScoreResult() = %0.4f, want %0.4f", got, tt.want)
				}
				return
			}
			if tt.min > 0 && got < tt.min {
				t.Fatalf("ScoreResult() = %0.4f, want >= %0.4f", got, tt.min)
			}
			if tt.max > 0 && got > tt.max {
				t.Fatalf("ScoreResult() = %0.4f, want <= %0.4f", got, tt.max)
			}
		})
	}
}
