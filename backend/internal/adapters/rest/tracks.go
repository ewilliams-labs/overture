package rest

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/ewilliams-labs/overture/backend/internal/core/ports"
)

const errCodeNoConfidentMatch = "NO_CONFIDENT_MATCH"

// addTrackRequest defines what the client sends us
type addTrackRequest struct {
	Title  string `json:"title"`
	Artist string `json:"artist"`
}

type addTrackResponse struct {
	ID string `json:"id"`
}

// AddTrack handles POST /playlists/{id}/tracks
func (h *Handler) AddTrack(w http.ResponseWriter, r *http.Request) {
	if !isJSONContentType(r) {
		writeError(w, http.StatusUnsupportedMediaType, "Content-Type must be application/json")
		return
	}

	playlistID := r.PathValue("id")
	if playlistID == "" {
		writeError(w, http.StatusBadRequest, "playlist id is required")
		return
	}

	// 1. Decode the Request Body
	var req addTrackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// 2. Validate Input
	if req.Title == "" || req.Artist == "" {
		writeError(w, http.StatusBadRequest, "title and artist are required")
		return
	}

	// 3. Call the Service (The Core Logic)
	// We pass the Context so the service can cancel long-running tasks if the user disconnects
	playlistIDResult, err := h.svc.AddTrackToPlaylist(r.Context(), playlistID, req.Title, req.Artist)
	if err != nil {
		var matchErr *ports.NoConfidentMatchError
		if errors.As(err, &matchErr) {
			writeErrorWithCode(w, http.StatusUnprocessableEntity, matchErr.Error(), errCodeNoConfidentMatch)
			return
		}
		// In a real app, you'd check the error type to decide between 400 vs 500
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// 4. Return the Response
	w.Header().Set("Location", "/playlists/"+playlistIDResult)
	writeJSON(w, http.StatusCreated, addTrackResponse{ID: playlistIDResult})
}
