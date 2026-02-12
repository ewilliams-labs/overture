package rest

import (
	"encoding/json"
	"net/http"
	"reflect"
)

type analyzeIntentRequest struct {
	Message string `json:"message"`
}

// AnalyzeIntent handles POST /playlists/{id}/intent
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

	if h.intent == nil {
		writeError(w, http.StatusNotImplemented, "intent compiler not configured")
		return
	}
	intentVal := reflect.ValueOf(h.intent)
	if intentVal.Kind() == reflect.Ptr && intentVal.IsNil() {
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

	intent, err := h.intent.AnalyzeIntent(r.Context(), req.Message)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, intent)
}
