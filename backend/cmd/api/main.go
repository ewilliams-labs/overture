package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ewilliams-labs/overture/backend/internal/adapters/ollama"
	"github.com/ewilliams-labs/overture/backend/internal/adapters/rest"
	"github.com/ewilliams-labs/overture/backend/internal/adapters/spotify"
	"github.com/ewilliams-labs/overture/backend/internal/adapters/sqlite"
	"github.com/ewilliams-labs/overture/backend/internal/core/ports"
	"github.com/ewilliams-labs/overture/backend/internal/core/services"
	"github.com/ewilliams-labs/overture/backend/internal/worker"
)

func main() {
	// 1. Configuration (Environment Variables)
	// It's best practice to crash early if required config is missing.
	clientID := os.Getenv("SPOTIFY_CLIENT_ID")
	clientSecret := os.Getenv("SPOTIFY_CLIENT_SECRET")
	fmt.Printf("DEBUG: Client ID length: %d\n", len(clientID))
	fmt.Printf("DEBUG: Client Secret length: %d\n", len(clientSecret))
	if clientID == "" || clientSecret == "" {
		log.Fatal("FATAL: SPOTIFY_CLIENT_ID and SPOTIFY_CLIENT_SECRET environment variables are required")
	}

	// 2. Initialize "Driven" Adapters (The Tools)
	// -- Database Adapter
	storageDriver := os.Getenv("STORAGE_DRIVER")
	if storageDriver == "" {
		storageDriver = "sqlite"
	}

	var repo ports.PlaylistRepository
	var repoCloser func() error

	switch storageDriver {
	case "sqlite":
		dbAdapter, err := sqlite.NewAdapter("overture.db")
		if err != nil {
			log.Fatalf("FATAL: Failed to initialize database: %v", err)
		}
		repo = dbAdapter
		repoCloser = dbAdapter.Close
	case "postgres":
		log.Fatal("Postgres driver not yet implemented")
	default:
		log.Fatalf("Unknown storage driver: %s", storageDriver)
	}
	defer repoCloser()

	// -- Spotify Adapter
	spotifyClient := spotify.NewClient(clientID, clientSecret)

	// 3. Initialize Core Logic (The Driver)
	// This is Dependency Injection in action.
	// We inject the specific adapters into the agnostic service.
	// The compiler guarantees that dbAdapter implements ports.PlaylistRepository
	// and spotifyClient implements ports.SpotifyClient.
	intentCompiler := ollama.NewClient(os.Getenv("OLLAMA_HOST"))
	svc := services.NewOrchestrator(spotifyClient, repo, intentCompiler)

	// 4. Initialize "Driving" Adapter (The Interface)
	// The HTTP handler talks to the Service.
	pool := worker.NewPool(repo, 2, 100)
	pool.Start(2)
	defer pool.Stop()

	handler := rest.NewHandler(svc, pool)

	// 5. Start the Server
	log.Println("------------------------------------------------")
	log.Println("🎶 Overture API is running on http://localhost:8080")
	log.Println("------------------------------------------------")

	srv := &http.Server{
		Addr:              ":8080",
		Handler:           handler,
		ReadHeaderTimeout: 15 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		err := srv.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			serverErr <- err
			return
		}
		serverErr <- nil
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-serverErr:
		if err != nil {
			log.Fatal(err)
		}
	case <-ctx.Done():
		log.Println("Shutting down server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("shutdown error: %v", err)
		}
	}
}
