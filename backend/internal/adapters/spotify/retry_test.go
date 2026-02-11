package spotify

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClientDoRequestWithRetry(t *testing.T) {
	tests := []struct {
		name             string
		statuses         []int
		maxRetries       int
		expectedStatus   int
		expectedAttempts int
		expectErr        bool
	}{
		{
			name:             "retries on 503 then succeeds",
			statuses:         []int{http.StatusServiceUnavailable, http.StatusServiceUnavailable, http.StatusOK},
			maxRetries:       3,
			expectedStatus:   http.StatusOK,
			expectedAttempts: 3,
			expectErr:        false,
		},
		{
			name:             "exhausts retries on 429",
			statuses:         []int{http.StatusTooManyRequests},
			maxRetries:       2,
			expectedStatus:   0,
			expectedAttempts: 2,
			expectErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attempts := 0
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				attempts++
				status := tt.statuses[len(tt.statuses)-1]
				if attempts <= len(tt.statuses) {
					status = tt.statuses[attempts-1]
				}
				w.WriteHeader(status)
			}))
			defer ts.Close()

			client := &Client{
				httpClient:  http.DefaultClient,
				baseURL:     ts.URL,
				maxRetries:  tt.maxRetries,
				baseBackoff: time.Millisecond,
			}

			req, err := http.NewRequest(http.MethodGet, ts.URL, nil)
			if err != nil {
				t.Fatalf("create request: %v", err)
			}

			resp, err := client.doRequestWithRetry(req)
			if (err != nil) != tt.expectErr {
				t.Fatalf("expected error: %v, got: %v", tt.expectErr, err)
			}
			if resp != nil {
				defer resp.Body.Close()
				if resp.StatusCode != tt.expectedStatus {
					t.Fatalf("status: got %d, want %d", resp.StatusCode, tt.expectedStatus)
				}
			}
			if attempts != tt.expectedAttempts {
				t.Fatalf("attempts: got %d, want %d", attempts, tt.expectedAttempts)
			}
		})
	}
}
