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

// GetPlaylist handles GET /playlists/{id}
func (h *Handler) GetPlaylist(w http.ResponseWriter, r *http.Request) {
	playlistID := r.PathValue("id")

	playlist, err := h.svc.GetPlaylist(r.Context(), playlistID)
	if err != nil {
		if err.Error() == "service: playlist id cannot be empty" {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(playlist); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// GetPlaylistAnalysis handles GET /playlists/{id}/analysis
func (h *Handler) GetPlaylistAnalysis(w http.ResponseWriter, r *http.Request) {
	playlistID := r.PathValue("id")
	if playlistID == "" {
		http.Error(w, "playlist id is required", http.StatusBadRequest)
		return
	}

	features, err := h.svc.GetPlaylistAnalysis(r.Context(), playlistID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(features); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}
