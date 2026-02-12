package ollama

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClient_AnalyzeIntent(t *testing.T) {
	tests := []struct {
		name         string
		status       int
		responseBody string
		wantErr      bool
	}{
		{
			name:         "Success",
			status:       http.StatusOK,
			responseBody: `{"message":{"role":"assistant","content":"{\"intent_type\":\"CREATE\",\"entities\":{\"artists\":[\"Willie Nelson\"],\"genres\":[]},\"vibe_constraints\":{\"acousticness\":{\"min\":0.8,\"weight\":\"HIGH\"}},\"sequence\":{\"pattern\":\"LINEAR\",\"description\":\"steady\"},\"explanation\":\"Test\"}"}}`,
			wantErr:      false,
		},
		{
			name:         "Server error",
			status:       http.StatusInternalServerError,
			responseBody: `{"error":"bad"}`,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			var gotRequest chatRequest
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/chat" {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				if r.Method != http.MethodPost {
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}
				if err := json.NewDecoder(r.Body).Decode(&gotRequest); err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer srv.Close()

			client := NewClient(srv.URL)
			intent, err := client.AnalyzeIntent(context.Background(), "test message")

			if (err != nil) != tt.wantErr {
				t.Fatalf("expected err=%v, got %v", tt.wantErr, err)
			}
			if tt.wantErr {
				return
			}
			if gotRequest.Model != "deepseek-r1:8b" {
				t.Fatalf("expected model deepseek-r1:8b, got %q", gotRequest.Model)
			}
			if gotRequest.Format != "json" {
				t.Fatalf("expected format json, got %q", gotRequest.Format)
			}
			if len(gotRequest.Messages) != 2 {
				t.Fatalf("expected 2 messages, got %d", len(gotRequest.Messages))
			}
			if gotRequest.Messages[0].Role != "system" || gotRequest.Messages[0].Content != systemPrompt {
				t.Fatalf("system prompt mismatch")
			}
			if gotRequest.Messages[1].Role != "user" || gotRequest.Messages[1].Content != "test message" {
				t.Fatalf("user message mismatch")
			}
			if intent.Explanation == "" {
				t.Fatalf("expected explanation in intent")
			}
		})
	}
}
