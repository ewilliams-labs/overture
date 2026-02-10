package rest

import (
	"encoding/json"
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
	h.router.HandleFunc("POST /tracks", h.AddTrack)
	h.router.HandleFunc("POST /playlists", h.CreatePlaylist)
}

// HealthCheck is a simple endpoint to verify the API is running.
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "message": "Overture is live 🎶"})
}
