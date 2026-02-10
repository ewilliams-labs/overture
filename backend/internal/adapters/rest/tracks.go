package rest

import (
	"encoding/json"
	"net/http"
)

// addTrackRequest defines what the client sends us
type addTrackRequest struct {
	PlaylistID string `json:"playlist_id"`
	ISRC       string `json:"isrc"`
}

// AddTrack handles POST /tracks
func (h *Handler) AddTrack(w http.ResponseWriter, r *http.Request) {
	// 1. Decode the Request Body
	var req addTrackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 2. Validate Input
	if req.PlaylistID == "" || req.ISRC == "" {
		http.Error(w, "playlist_id and isrc are required", http.StatusBadRequest)
		return
	}

	// 3. Call the Service (The Core Logic)
	// We pass the Context so the service can cancel long-running tasks if the user disconnects
	playlist, err := h.svc.AddTrackToPlaylist(r.Context(), req.PlaylistID, req.ISRC)
	if err != nil {
		// In a real app, you'd check the error type to decide between 400 vs 500
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 4. Return the Response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated) // 201 Created
	if err := json.NewEncoder(w).Encode(playlist); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}
