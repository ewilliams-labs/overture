package spotify

import "strings"

var searchSuffixTokens = map[string]struct{}{
	"clean":      {},
	"deluxe":     {},
	"edition":    {},
	"edit":       {},
	"explicit":   {},
	"feat":       {},
	"featuring":  {},
	"ft":         {},
	"live":       {},
	"mix":        {},
	"mono":       {},
	"radio":      {},
	"remaster":   {},
	"remastered": {},
	"stereo":     {},
	"version":    {},
}

// Normalize cleans a search string for comparison.
func Normalize(input string) string {
	if strings.TrimSpace(input) == "" {
		return ""
	}

	lowered := strings.ToLower(strings.TrimSpace(input))
	trimmed := stripCommonSuffixes(lowered)
	cleaned := cleanSeparators(trimmed)

	return strings.Join(strings.Fields(cleaned), " ")
}

// ScoreResult returns a similarity score between two artist+title pairs.
func ScoreResult(targetArtist string, targetTitle string, actualArtist string, actualTitle string) float64 {
	target := Normalize(strings.TrimSpace(targetArtist + " " + targetTitle))
	actual := Normalize(strings.TrimSpace(actualArtist + " " + actualTitle))
	if target == "" || actual == "" {
		return 0
	}

	return similarity(target, actual)
}

func stripCommonSuffixes(input string) string {
	trimmed := strings.TrimSpace(input)
	for {
		next := trimBracketedSuffix(trimmed)
		next = trimDashSuffix(next)
		if next == trimmed {
			return trimmed
		}
		trimmed = strings.TrimSpace(next)
	}
}

func trimBracketedSuffix(input string) string {
	trimmed := strings.TrimSpace(input)
	if strings.HasSuffix(trimmed, ")") {
		if idx := strings.LastIndex(trimmed, "("); idx != -1 && idx < len(trimmed)-1 {
			suffix := trimmed[idx+1 : len(trimmed)-1]
			if suffixHasToken(suffix) {
				return strings.TrimSpace(trimmed[:idx])
			}
		}
	}

	if strings.HasSuffix(trimmed, "]") {
		if idx := strings.LastIndex(trimmed, "["); idx != -1 && idx < len(trimmed)-1 {
			suffix := trimmed[idx+1 : len(trimmed)-1]
			if suffixHasToken(suffix) {
				return strings.TrimSpace(trimmed[:idx])
			}
		}
	}

	return input
}

func trimDashSuffix(input string) string {
	trimmed := strings.TrimSpace(input)
	idx := strings.LastIndex(trimmed, " - ")
	if idx == -1 {
		return input
	}

	suffix := strings.TrimSpace(trimmed[idx+3:])
	if suffixHasToken(suffix) {
		return strings.TrimSpace(trimmed[:idx])
	}

	return input
}

func suffixHasToken(input string) bool {
	if strings.TrimSpace(input) == "" {
		return false
	}

	cleaned := cleanSeparators(strings.ToLower(input))
	for _, token := range strings.Fields(cleaned) {
		if _, ok := searchSuffixTokens[token]; ok {
			return true
		}
	}

	return false
}

func similarity(a string, b string) float64 {
	if a == b {
		return 1.0
	}
	maxLen := maxInt(len([]rune(a)), len([]rune(b)))
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
			curr[j] = min3(
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

func min3(a int, b int, c int) int {
	m := a
	if b < m {
		m = b
	}
	if c < m {
		m = c
	}
	return m
}

func maxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
