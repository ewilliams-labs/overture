package rest

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/ewilliams-labs/overture/backend/internal/core/domain"
	"github.com/ewilliams-labs/overture/backend/internal/core/services"
)

type analyzeIntentRequest struct {
	Message string `json:"message"`
}

// sseStatus represents the status field in SSE events.
type sseStatus struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// sseComplete represents the final SSE event with the IntentObject and summary.
type sseComplete struct {
	Status          string              `json:"status"`
	Data            domain.IntentObject `json:"data"`
	TracksEvaluated int                 `json:"tracks_evaluated"`
	TracksAdded     int                 `json:"tracks_added"`
	Summary         string              `json:"summary"`
}

// sseError represents an error SSE event.
type sseError struct {
	Status string `json:"status"`
	Error  string `json:"error"`
}

// AnalyzeIntent handles POST /playlists/{id}/intent using Server-Sent Events.
func (h *Handler) AnalyzeIntent(w http.ResponseWriter, r *http.Request) {
	if !isJSONContentType(r) {
		writeError(w, http.StatusUnsupportedMediaType, "Content-Type must be application/json")
		return
	}

	playlistID := r.PathValue("id")
	if playlistID == "" {
		writeError(w, http.StatusBadRequest, "playlist id is required")
		return
	}

	if !h.svc.HasIntentCompiler() {
		writeError(w, http.StatusNotImplemented, "intent compiler not configured")
		return
	}

	var req analyzeIntentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if req.Message == "" {
		writeError(w, http.StatusBadRequest, "message is required")
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	rc := http.NewResponseController(w)

	// Send initial "thinking" event
	if err := writeSSEEvent(w, rc, "status", sseStatus{
		Status:  "thinking",
		Message: "Overture is analyzing the vibe...",
	}); err != nil {
		return // Client disconnected
	}

	// Channel to receive the result from ProcessIntent
	type intentResultWrapper struct {
		result services.IntentResult
		err    error
	}
	resultCh := make(chan intentResultWrapper, 1)

	// Create a detached context for background processing.
	// This ensures DB writes and provider operations complete even if the client disconnects.
	// context.WithoutCancel preserves values from the parent context but ignores cancellation.
	detachedCtx := context.WithoutCancel(r.Context())

	// Run ProcessIntent in a goroutine with the detached context
	go func() {
		result, err := h.svc.ProcessIntent(detachedCtx, playlistID, req.Message)
		resultCh <- intentResultWrapper{result: result, err: err}
	}()

	// Heartbeat ticker - every 10 seconds
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	// Wait for result with heartbeats
	for {
		select {
		case <-r.Context().Done():
			// Client disconnected
			return
		case <-ticker.C:
			// Send heartbeat
			if err := writeSSEEvent(w, rc, "status", sseStatus{
				Status: "heartbeat",
			}); err != nil {
				return // Client disconnected
			}
		case wrapper := <-resultCh:
			if wrapper.err != nil {
				// Send error event
				_ = writeSSEEvent(w, rc, "error", sseError{
					Status: "error",
					Error:  wrapper.err.Error(),
				})
				return
			}

			// Send final "complete" event with IntentObject and summary
			_ = writeSSEEvent(w, rc, "complete", sseComplete{
				Status:          "complete",
				Data:            wrapper.result.Intent,
				TracksEvaluated: wrapper.result.TracksEvaluated,
				TracksAdded:     wrapper.result.TracksAdded,
				Summary:         wrapper.result.Summary,
			})
			return
		}
	}
}
