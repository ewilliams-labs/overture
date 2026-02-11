package spotify

import "strings"

const (
	minTitleSimilarity   = 0.65
	minArtistSimilarity  = 0.55
	minOverallSimilarity = 0.70
)

func trackMatchScore(requestTitle string, requestArtist string, candidate spotifyTrack) (float64, bool) {
	normalizedTitle := normalizeSearchInput(requestTitle)
	normalizedArtist := normalizeSearchInput(requestArtist)
	candidateTitle := normalizeSearchInput(candidate.Name)
	candidateArtist := normalizeSearchInput(joinArtistNames(candidate))

	if normalizedTitle == "" || normalizedArtist == "" || candidateTitle == "" || candidateArtist == "" {
		return 0, false
	}

	titleSim := similarity(normalizedTitle, candidateTitle)
	artistSim := similarity(normalizedArtist, candidateArtist)
	score := 0.7*titleSim + 0.3*artistSim

	if titleSim < minTitleSimilarity || artistSim < minArtistSimilarity || score < minOverallSimilarity {
		return score, false
	}

	return score, true
}

func similarity(a string, b string) float64 {
	if a == b {
		return 1.0
	}
	maxLen := max(len([]rune(a)), len([]rune(b)))
	if maxLen == 0 {
		return 1.0
	}

	distance := levenshteinDistance(a, b)
	return 1.0 - float64(distance)/float64(maxLen)
}

func levenshteinDistance(a string, b string) int {
	ra := []rune(a)
	rb := []rune(b)
	if len(ra) == 0 {
		return len(rb)
	}
	if len(rb) == 0 {
		return len(ra)
	}

	prev := make([]int, len(rb)+1)
	curr := make([]int, len(rb)+1)
	for j := 0; j <= len(rb); j++ {
		prev[j] = j
	}

	for i := 1; i <= len(ra); i++ {
		curr[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 0
			if ra[i-1] != rb[j-1] {
				cost = 1
			}
			curr[j] = min(
				prev[j]+1,
				curr[j-1]+1,
				prev[j-1]+cost,
			)
		}
		copy(prev, curr)
	}

	return prev[len(rb)]
}

func joinArtistNames(track spotifyTrack) string {
	if len(track.Artists) == 0 {
		return ""
	}
	parts := make([]string, 0, len(track.Artists))
	for _, artist := range track.Artists {
		parts = append(parts, artist.Name)
	}
	return strings.Join(parts, " ")
}

func min(a int, b int, c int) int {
	m := a
	if b < m {
		m = b
	}
	if c < m {
		m = c
	}
	return m
}

func max(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
