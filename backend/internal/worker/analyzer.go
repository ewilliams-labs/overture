package worker

import (
	"fmt"
	"io"
	"math"
	"net/http"
	"time"

	"github.com/hajimehoshi/go-mp3"
)

var previewClient = &http.Client{Timeout: 15 * time.Second}

func analyzePreview(url string) (float64, error) {
	// #nosec G107 -- URL is a validated Spotify preview URL from trusted API response
	resp, err := previewClient.Get(url)
	if err != nil {
		return 0, fmt.Errorf("preview fetch failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("preview fetch status %d", resp.StatusCode)
	}

	decoder, err := mp3.NewDecoder(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("preview decode failed: %w", err)
	}

	buf := make([]byte, 4096)
	var sumSquares float64
	var count float64

	for {
		n, err := decoder.Read(buf)
		if n > 0 {
			for i := 0; i+1 < n; i += 2 {
				sample := int16(buf[i]) | int16(buf[i+1])<<8
				val := float64(sample)
				sumSquares += val * val
				count++
			}
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return 0, fmt.Errorf("preview read failed: %w", err)
		}
	}

	if count == 0 {
		return 0, fmt.Errorf("preview contains no samples")
	}

	rms := math.Sqrt(sumSquares / count)
	energy := rms / 32768.0
	if energy < 0 {
		energy = 0
	}
	if energy > 1 {
		energy = 1
	}

	return energy, nil
}

// AnalyzePreviewFunc allows tests to override the analyzer implementation.
var AnalyzePreviewFunc = analyzePreview
