package rest

import (
	"encoding/json"
	"net/http"
)

type createPlaylistRequest struct {
	Name string `json:"name"`
}

// CreatePlaylist handles POST /playlists
func (h *Handler) CreatePlaylist(w http.ResponseWriter, r *http.Request) {
	// 1. Decode Request
	var req createPlaylistRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 2. Call Service
	playlist, err := h.svc.CreatePlaylist(r.Context(), req.Name)
	if err != nil {
		// Differentiate errors: Empty name vs DB failure
		if err.Error() == "service: playlist name cannot be empty" {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 3. Respond
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated) // 201 Created
	if err := json.NewEncoder(w).Encode(playlist); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}
