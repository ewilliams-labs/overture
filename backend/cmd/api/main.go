package main

import (
	"log"
	"net/http"
	"os"

	"github.com/ewilliams-labs/overture/backend/internal/adapters/rest"
	"github.com/ewilliams-labs/overture/backend/internal/adapters/spotify"
	"github.com/ewilliams-labs/overture/backend/internal/adapters/sqlite"
	"github.com/ewilliams-labs/overture/backend/internal/core/services"
)

func main() {
	// 1. Configuration (Environment Variables)
	// It's best practice to crash early if required config is missing.
	clientID := os.Getenv("SPOTIFY_CLIENT_ID")
	clientSecret := os.Getenv("SPOTIFY_CLIENT_SECRET")
	if clientID == "" || clientSecret == "" {
		log.Fatal("FATAL: SPOTIFY_CLIENT_ID and SPOTIFY_CLIENT_SECRET environment variables are required")
	}

	// 2. Initialize "Driven" Adapters (The Tools)
	// -- Database Adapter
	// We create the concrete struct here so we can defer Close()
	dbAdapter, err := sqlite.NewAdapter("overture.db")
	if err != nil {
		log.Fatalf("FATAL: Failed to initialize database: %v", err)
	}
	// Ensure the connection closes when the server stops
	defer dbAdapter.Close()

	// -- Spotify Adapter
	spotifyClient := spotify.NewClient(clientID, clientSecret)

	// 3. Initialize Core Logic (The Driver)
	// This is Dependency Injection in action.
	// We inject the specific adapters into the agnostic service.
	// The compiler guarantees that dbAdapter implements ports.PlaylistRepository
	// and spotifyClient implements ports.SpotifyClient.
	svc := services.NewOrchestrator(spotifyClient, dbAdapter)

	// 4. Initialize "Driving" Adapter (The Interface)
	// The HTTP handler talks to the Service.
	handler := rest.NewHandler(svc)

	// 5. Start the Server
	log.Println("------------------------------------------------")
	log.Println("🎶 Overture API is running on http://localhost:8080")
	log.Println("------------------------------------------------")

	if err := http.ListenAndServe(":8080", handler); err != nil {
		log.Fatal(err)
	}
}
