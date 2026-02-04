// Package rest provides HTTP handlers for the Overture API.
package rest

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ewilliams-labs/overture/backend/internal/core/services"
)

type Handler struct {
	orchestrator *services.Orchestrator
}

// NewHandler creates and returns a new Handler instance initialized with the provided Orchestrator service.
// The Handler is responsible for managing HTTP request handling operations that depend on the orchestrator.
func NewHandler(orchestrator *services.Orchestrator) *Handler {
	return &Handler{
		orchestrator: orchestrator,
	}
}

type addTrackRequest struct {
	PlaylistID string `json:"playlist_id"`
	TrackID    string `json:"track_id"`
}

func (h *Handler) AddTrack(w http.ResponseWriter, r *http.Request) {
	var req addTrackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	if req.PlaylistID == "" || req.TrackID == "" {
		http.Error(w, "playlist_id and track_id are required", http.StatusBadRequest)
		return
	}

	if err := h.orchestrator.AddTrackToPlaylist(r.Context(), req.PlaylistID, req.TrackID); err != nil {
		http.Error(w, fmt.Sprintf("failed to add track: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}
