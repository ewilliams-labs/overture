package ollama

import (
	"context"
	"os"
	"testing"
)

// TestClient_AnalyzeIntent_Integration tests against a live Ollama instance.
// This test is skipped unless RUN_AI_TESTS=true is set.
func TestClient_AnalyzeIntent_Integration(t *testing.T) {
	if os.Getenv("RUN_AI_TESTS") != "true" {
		t.Skip("Skipping AI-dependent test (set RUN_AI_TESTS=true to enable)")
	}

	ollamaHost := os.Getenv("OLLAMA_HOST")
	if ollamaHost == "" {
		ollamaHost = "http://localhost:11434"
	}

	client := NewClient(ollamaHost)

	tests := []struct {
		name    string
		message string
	}{
		{
			name:    "Simple artist request",
			message: "I want some tracks by Willie Nelson",
		},
		{
			name:    "Complex vibe request",
			message: "Give me a chill acoustic set with low energy, nothing too upbeat",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			intent, err := client.AnalyzeIntent(context.Background(), tt.message)
			if err != nil {
				t.Fatalf("AnalyzeIntent() error = %v", err)
			}

			// Basic validation that we got a response
			if intent.IntentType == "" {
				t.Error("expected non-empty intent_type")
			}
			t.Logf("Intent: %+v", intent)
		})
	}
}
