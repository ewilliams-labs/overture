package spotify

import (
	"hash/fnv"
	"math/rand"

	"github.com/ewilliams-labs/overture/backend/internal/core/domain"
)

func generateDeterministicFeatures(trackID string) domain.AudioFeatures {
	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(trackID))
	seed := int64(hasher.Sum32())
	// #nosec G404 -- Deterministic RNG for reproducible audio features, not security-sensitive
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

func allFeaturesZero(features spotifyAudioFeatures) bool {
	return features.Danceability == 0 &&
		features.Energy == 0 &&
		features.Valence == 0 &&
		features.Tempo == 0 &&
		features.Instrumentalness == 0 &&
		features.Acousticness == 0
}
