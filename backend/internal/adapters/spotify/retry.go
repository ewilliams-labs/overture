package spotify

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

const (
	defaultMaxRetries = 3
	defaultBackoffMs  = 500
)

func getRetryConfig() (int, time.Duration) {
	maxRetries := defaultMaxRetries
	if raw := os.Getenv("SPOTIFY_MAX_RETRIES"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			maxRetries = parsed
		}
	}

	backoffMs := defaultBackoffMs
	if raw := os.Getenv("SPOTIFY_RETRY_BACKOFF_MS"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			backoffMs = parsed
		}
	}

	return maxRetries, time.Duration(backoffMs) * time.Millisecond
}

func (c *Client) doRequestWithRetry(req *http.Request) (*http.Response, error) {
	maxRetries := c.maxRetries
	if maxRetries <= 0 {
		maxRetries = defaultMaxRetries
	}

	baseBackoff := c.baseBackoff
	if baseBackoff <= 0 {
		baseBackoff = time.Duration(defaultBackoffMs) * time.Millisecond
	}

	if req.Body != nil && req.GetBody == nil {
		bodyBytes, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, fmt.Errorf("spotify adapter: read request body: %w", err)
		}
		_ = req.Body.Close()
		req.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(bodyBytes)), nil
		}
	}

	ctx := req.Context()
	for attempt := 0; attempt < maxRetries; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("spotify adapter: request canceled: %w", err)
		}

		if req.GetBody != nil {
			body, err := req.GetBody()
			if err != nil {
				return nil, fmt.Errorf("spotify adapter: reset request body: %w", err)
			}
			req.Body = body
		}

		// #nosec G107 -- URL constructed from trusted Spotify API baseURL constant
		resp, err := c.httpClient.Do(req)
		retryAfter, retry := shouldRetry(resp, err)
		if !retry {
			return resp, err
		}

		attemptNum := attempt + 1
		if err != nil {
			log.Printf("WARN spotify adapter: retry attempt %d/%d after error: %v", attemptNum, maxRetries, err) // #nosec G706 -- error value is from trusted internal HTTP operation
		} else if resp != nil {
			log.Printf("WARN spotify adapter: retry attempt %d/%d after status %d", attemptNum, maxRetries, resp.StatusCode) // #nosec G706 -- status code is numeric from trusted HTTP response
			_ = resp.Body.Close()
		}

		if attempt == maxRetries-1 {
			if err != nil {
				return nil, fmt.Errorf("spotify adapter: request failed after %d attempts: %w", maxRetries, err)
			}
			if resp != nil {
				_ = resp.Body.Close()
				return nil, fmt.Errorf("spotify adapter: request failed after %d attempts: status %d", maxRetries, resp.StatusCode)
			}
			return nil, fmt.Errorf("spotify adapter: request failed after %d attempts", maxRetries)
		}

		backoff := baseBackoff * time.Duration(1<<attempt)
		if retryAfter > 0 {
			backoff = retryAfter
		}

		if err := sleepWithContext(ctx, backoff); err != nil {
			return nil, err
		}
	}

	return nil, fmt.Errorf("spotify adapter: request failed after %d attempts", maxRetries)
}

func shouldRetry(resp *http.Response, err error) (time.Duration, bool) {
	if err != nil {
		return 0, true
	}
	if resp == nil {
		return 0, false
	}

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= http.StatusInternalServerError {
		return parseRetryAfter(resp), true
	}

	return 0, false
}

func parseRetryAfter(resp *http.Response) time.Duration {
	if resp == nil {
		return 0
	}

	retryAfter := resp.Header.Get("Retry-After")
	if retryAfter == "" {
		return 0
	}

	if seconds, err := strconv.Atoi(retryAfter); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}

	if when, err := http.ParseTime(retryAfter); err == nil {
		until := time.Until(when)
		if until > 0 {
			return until
		}
	}

	return 0
}

func sleepWithContext(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return fmt.Errorf("spotify adapter: request canceled: %w", ctx.Err())
	case <-timer.C:
		return nil
	}
}
