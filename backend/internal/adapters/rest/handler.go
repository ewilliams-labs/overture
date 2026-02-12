// Package rest provides HTTP adapters for the Overture application.
package rest

import (
	"encoding/json"
	"mime"
	"net/http"

	"github.com/ewilliams-labs/overture/backend/internal/core/services"
)

// Handler manages the HTTP interface for our application.
type Handler struct {
	svc    *services.Orchestrator // Dependency on the Core Service
	router *http.ServeMux         // Standard library router
}

// NewHandler initializes the HTTP adapter and sets up routes.
func NewHandler(svc *services.Orchestrator) *Handler {
	h := &Handler{
		svc:    svc,
		router: http.NewServeMux(),
	}

	// Register Routes
	h.routes()

	return h
}

// ServeHTTP satisfies the http.Handler interface.
// It acts as a proxy, passing the request to our internal router.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.router.ServeHTTP(w, r)
}

// routes defines the mapping between URLs and methods.
func (h *Handler) routes() {
	// Health Check
	h.router.HandleFunc("GET /health", h.HealthCheck)
	// Playlist Management
	h.router.HandleFunc("POST /playlists", h.CreatePlaylist)
	h.router.HandleFunc("GET /playlists/{id}", h.GetPlaylist)
	h.router.HandleFunc("POST /playlists/{id}/tracks", h.AddTrack)
	h.router.HandleFunc("GET /playlists/{id}/analysis", h.GetPlaylistAnalysis)
}

// HealthCheck is a simple endpoint to verify the API is running.
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "message": "Overture is live 🎶"})
}

type errorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code,omitempty"`
}

func isJSONContentType(r *http.Request) bool {
	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		return false
	}
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return false
	}
	return mediaType == "application/json"
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errorResponse{Error: msg})
}

func writeErrorWithCode(w http.ResponseWriter, status int, msg string, code string) {
	writeJSON(w, status, errorResponse{Error: msg, Code: code})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}
