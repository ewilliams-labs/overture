package spotify

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/ewilliams-labs/overture/backend/internal/core/domain"
	"github.com/ewilliams-labs/overture/backend/internal/core/ports"
)

const defaultSearchMatchThreshold = 0.5

// GetTrackByMetadata searches for a track using title and artist metadata.
func (c *Client) GetTrackByMetadata(ctx context.Context, title string, artist string) (domain.Track, error) {
	track, err := c.searchTrack(ctx, title, artist)
	if err != nil {
		return domain.Track{}, err
	}

	return mapTrackToDomain(track, nil), nil
}

func (c *Client) searchTrack(ctx context.Context, title string, artist string) (spotifyTrack, error) {
	searchURL, err := url.Parse(fmt.Sprintf("%s/search", c.baseURL))
	if err != nil {
		return spotifyTrack{}, fmt.Errorf("spotify adapter: invalid search url: %w", err)
	}

	normalizedTitle, normalizedArtist := normalizeTitleArtist(title, artist)
	queryTitle := fallbackIfEmpty(normalizedTitle, title)
	queryArtist := fallbackIfEmpty(normalizedArtist, artist)

	query := searchURL.Query()
	query.Set("q", fmt.Sprintf("track:%s artist:%s", queryTitle, queryArtist))
	query.Set("type", "track")
	query.Set("limit", "5")
	query.Set("market", "US")
	searchURL.RawQuery = query.Encode()

	log.Printf("DEBUG spotify adapter: search request URL: %s", searchURL.String()) // #nosec G706 -- URL is internally constructed from trusted baseURL constant

	searchReq, err := http.NewRequestWithContext(ctx, http.MethodGet, searchURL.String(), nil)
	if err != nil {
		return spotifyTrack{}, fmt.Errorf("spotify adapter: failed to create search request: %w", err)
	}

	searchResp, err := c.doRequestWithRetry(searchReq)
	if err != nil {
		return spotifyTrack{}, fmt.Errorf("spotify adapter: search request failed: %w", err)
	}
	defer searchResp.Body.Close()

	if searchResp.StatusCode != http.StatusOK {
		return spotifyTrack{}, fmt.Errorf("spotify adapter: search status %d", searchResp.StatusCode)
	}

	var searchBody struct {
		Tracks struct {
			Items []spotifyTrack `json:"items"`
		} `json:"tracks"`
	}

	if err := json.NewDecoder(searchResp.Body).Decode(&searchBody); err != nil {
		return spotifyTrack{}, fmt.Errorf("spotify adapter: search decode error: %w", err)
	}

	if len(searchBody.Tracks.Items) == 0 {
		return spotifyTrack{}, fmt.Errorf("spotify adapter: %w", &ports.NoConfidentMatchError{Title: title, Artist: artist})
	}

	maxItems := len(searchBody.Tracks.Items)
	if maxItems > 5 {
		maxItems = 5
	}
	minConfidence := getMinConfidence()
	bestScore := 0.0
	bestIndex := -1
	bestExactArtist := false
	bestTitleMatch := false
	for i := 0; i < maxItems; i++ {
		candidate := searchBody.Tracks.Items[i]
		candidateArtist := joinArtistNames(candidate)
		score := ScoreResult(artist, title, candidateArtist, candidate.Name)
		exactArtist := artistExactMatch(candidate, artist)
		if exactArtist {
			score += 0.4
		}
		titleMatch := titleSubstringMatch(candidate.Name, title)
		if titleMatch {
			score += 0.3
		}
		if score > 1.0 {
			score = 1.0
		}
		log.Printf("DEBUG spotify adapter: Spotify Match: %s - %s (Score: %.2f)", candidateArtist, candidate.Name, score)
		if score >= minConfidence && (score > bestScore || (score == bestScore && (exactArtist && !bestExactArtist || (exactArtist == bestExactArtist && titleMatch && !bestTitleMatch)))) {
			bestScore = score
			bestIndex = i
			bestExactArtist = exactArtist
			bestTitleMatch = titleMatch
		}
	}

	if bestIndex == -1 {
		return spotifyTrack{}, fmt.Errorf("spotify adapter: %w", &ports.NoConfidentMatchError{Title: title, Artist: artist})
	}

	return searchBody.Tracks.Items[bestIndex], nil
}

func getMinConfidence() float64 {
	value := strings.TrimSpace(os.Getenv("SPOTIFY_MIN_CONFIDENCE"))
	if value == "" {
		return defaultSearchMatchThreshold
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		log.Printf("WARN spotify adapter: invalid SPOTIFY_MIN_CONFIDENCE %q", value)
		return defaultSearchMatchThreshold
	}
	if parsed < 0 {
		return 0
	}
	if parsed > 1 {
		return 1
	}
	return parsed
}

func artistExactMatch(candidate spotifyTrack, target string) bool {
	target = strings.TrimSpace(target)
	if target == "" {
		return false
	}
	for _, artist := range candidate.Artists {
		if strings.EqualFold(strings.TrimSpace(artist.Name), target) {
			return true
		}
	}
	return false
}

func titleSubstringMatch(candidateTitle string, targetTitle string) bool {
	ct := strings.ToLower(strings.TrimSpace(candidateTitle))
	tt := strings.ToLower(strings.TrimSpace(targetTitle))
	if ct == "" || tt == "" {
		return false
	}
	return strings.Contains(ct, tt)
}
