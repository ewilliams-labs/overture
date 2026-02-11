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
	if !isJSONContentType(r) {
		writeError(w, http.StatusUnsupportedMediaType, "Content-Type must be application/json")
		return
	}

	// 1. Decode Request
	var req createPlaylistRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// 2. Call Service
	playlist, err := h.svc.CreatePlaylist(r.Context(), req.Name)
	if err != nil {
		// Differentiate errors: Empty name vs DB failure
		if err.Error() == "service: playlist name cannot be empty" {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// 3. Respond
	w.Header().Set("Location", "/playlists/"+playlist.ID)
	writeJSON(w, http.StatusCreated, playlist)
}

// GetPlaylist handles GET /playlists/{id}
func (h *Handler) GetPlaylist(w http.ResponseWriter, r *http.Request) {
	playlistID := r.PathValue("id")

	playlist, err := h.svc.GetPlaylist(r.Context(), playlistID)
	if err != nil {
		if err.Error() == "service: playlist id cannot be empty" {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, playlist)
}

// GetPlaylistAnalysis handles GET /playlists/{id}/analysis
func (h *Handler) GetPlaylistAnalysis(w http.ResponseWriter, r *http.Request) {
	playlistID := r.PathValue("id")
	if playlistID == "" {
		writeError(w, http.StatusBadRequest, "playlist id is required")
		return
	}

	features, err := h.svc.GetPlaylistAnalysis(r.Context(), playlistID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, features)
}
