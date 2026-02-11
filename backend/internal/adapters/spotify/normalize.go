package spotify

import (
	"strings"
	"unicode"
)

var noiseTokens = map[string]struct{}{
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

func normalizeTitleArtist(title string, artist string) (string, string) {
	return normalizeSearchInput(title), normalizeSearchInput(artist)
}

func normalizeSearchInput(input string) string {
	if input == "" {
		return ""
	}

	lower := strings.ToLower(input)
	filtered := stripBracketedSegments(lower)
	tokens := strings.Fields(cleanSeparators(filtered))

	cleaned := make([]string, 0, len(tokens))
	for _, token := range tokens {
		if _, drop := noiseTokens[token]; drop {
			continue
		}
		cleaned = append(cleaned, token)
	}

	return strings.Join(cleaned, " ")
}

func stripBracketedSegments(input string) string {
	var out strings.Builder
	depth := 0
	for _, r := range input {
		switch r {
		case '(', '[':
			depth++
		case ')', ']':
			if depth > 0 {
				depth--
			}
		default:
			if depth == 0 {
				out.WriteRune(r)
			}
		}
	}

	return out.String()
}

func cleanSeparators(input string) string {
	var out strings.Builder
	lastSpace := false
	for _, r := range input {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			out.WriteRune(r)
			lastSpace = false
			continue
		}
		if !lastSpace {
			out.WriteRune(' ')
			lastSpace = true
		}
	}

	return out.String()
}

func fallbackIfEmpty(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}

	return value
}
