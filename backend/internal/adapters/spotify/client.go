// Package spotify provides the Spotify API adapter implementing the ports.TrackSearcher interface.
package spotify

import (
	"context"
	"net/http"
	"time"

	"golang.org/x/oauth2/clientcredentials"
)

const (
	// BaseURL is the production Spotify API endpoint
	BaseURL = "https://api.spotify.com/v1"
)

// Client adapts the Spotify API to our Domain interface
type Client struct {
	httpClient  *http.Client
	baseURL     string
	maxRetries  int
	baseBackoff time.Duration
}

// NewClient creates a standard Spotify client.
func NewClient(clientID, clientSecret string) *Client {
	config := &clientcredentials.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     "https://accounts.spotify.com/api/token", // Real Spotify Auth URL
	}

	httpClient := config.Client(context.Background())
	maxRetries, baseBackoff := getRetryConfig()

	return &Client{
		httpClient:  httpClient,
		baseURL:     BaseURL,
		maxRetries:  maxRetries,
		baseBackoff: baseBackoff,
	}
}

// NewClientWithBaseURL creates a client with a custom base URL.
// This is strictly for TESTS (injecting the mock server URL).
func NewClientWithBaseURL(httpClient *http.Client, baseURL string) *Client {
	maxRetries, baseBackoff := getRetryConfig()

	return &Client{
		httpClient:  httpClient,
		baseURL:     baseURL,
		maxRetries:  maxRetries,
		baseBackoff: baseBackoff,
	}
}
