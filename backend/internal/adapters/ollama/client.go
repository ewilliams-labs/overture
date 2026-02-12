// Package ollama provides an adapter for the Ollama LLM service.
// It implements intent analysis by sending user messages to a local Ollama instance
// and parsing the structured JSON response into domain IntentObjects.
package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ewilliams-labs/overture/backend/internal/core/domain"
)

const defaultBaseURL = "http://localhost:11434"

const systemPrompt = "You are the Overture Music Intent Engine. Your goal is to translate abstract human desires into a structured JSON 'IntentObject'.\n\nRules:\nReasoning: Use your internal logic to map stylistic requests (e.g., 'no auto-tune') to technical constraints (e.g., 'acousticness.min: 0.8').\nEntities: Extract specific artists or genres mentioned.\nOutput: Return ONLY a valid JSON object. No conversational text.\nVibe Scaling: Energy and Valence are 0.0 to 1.0.\nExample Mapping: 'I want a sad acoustic set' -> { 'vibe_constraints': { 'valence': {'target': 0.2}, 'acousticness': {'min': 0.7} } }"

type Client struct {
	baseURL    string
	httpClient *http.Client
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
	Format   string        `json:"format,omitempty"`
}

type chatResponse struct {
	Message chatMessage `json:"message"`
	Error   string      `json:"error,omitempty"`
}

func NewClient(baseURL string) *Client {
	baseURL = strings.TrimRight(baseURL, "/")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) AnalyzeIntent(ctx context.Context, message string) (domain.IntentObject, error) {
	payload := chatRequest{
		Model:  "deepseek-r1:8b",
		Stream: false,
		Format: "json",
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: message},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return domain.IntentObject{}, fmt.Errorf("ollama: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return domain.IntentObject{}, fmt.Errorf("ollama: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return domain.IntentObject{}, fmt.Errorf("ollama: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return domain.IntentObject{}, fmt.Errorf("ollama: unexpected status %d", resp.StatusCode)
	}

	var parsed chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return domain.IntentObject{}, fmt.Errorf("ollama: decode response: %w", err)
	}
	if parsed.Error != "" {
		return domain.IntentObject{}, fmt.Errorf("ollama: %s", parsed.Error)
	}

	if strings.TrimSpace(parsed.Message.Content) == "" {
		return domain.IntentObject{}, fmt.Errorf("ollama: empty response")
	}

	var intent domain.IntentObject
	if err := json.Unmarshal([]byte(parsed.Message.Content), &intent); err != nil {
		return domain.IntentObject{}, fmt.Errorf("ollama: decode intent: %w", err)
	}

	return intent, nil
}
