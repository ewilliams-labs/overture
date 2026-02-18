// Package main provides the BFF (Backend-for-Frontend) service for Overture.
// The BFF acts as an API gateway optimized for React/Generative UI consumption,
// handling OAuth sessions, SSE stream shaping, and multi-provider aggregation.
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
)

func main() {
	backendURL := getEnv("BACKEND_URL", "http://backend:8080")
	port := getEnv("PORT", "3000")

	log.Println("================================================")
	log.Println("🎭 Overture BFF starting...")
	log.Printf("   Backend URL: %s", backendURL)
	log.Printf("   Listening on: :%s", port)
	log.Println("================================================")

	// Verify backend connectivity on startup
	if err := waitForBackend(backendURL, 30*time.Second); err != nil {
		log.Printf("⚠️  Backend not reachable: %v (continuing anyway)", err)
	} else {
		log.Println("✅ Backend health check passed")
	}

	// Set up routes
	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		readyHandler(w, r, backendURL)
	})
	mux.HandleFunc("/", rootHandler)

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("🎭 BFF is running on http://localhost:%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Shutting down BFF...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Shutdown error: %v", err)
	}
	log.Println("👋 BFF stopped")
}

// healthHandler returns the BFF's own health status
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"status":"healthy","service":"bff"}`)
}

// readyHandler checks if the BFF can reach the backend
func readyHandler(w http.ResponseWriter, r *http.Request, backendURL string) {
	w.Header().Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(backendURL + "/health")
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(w, `{"status":"not_ready","error":"%s"}`, err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(w, `{"status":"not_ready","backend_status":%d}`, resp.StatusCode)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"status":"ready","backend":"connected"}`)
}

// rootHandler provides basic service info
func rootHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, `{"service":"overture-bff","version":"0.1.0","description":"Backend-for-Frontend API Gateway"}`)
}

// waitForBackend polls the backend health endpoint until it responds or times out
func waitForBackend(backendURL string, timeout time.Duration) error {
	client := &http.Client{Timeout: 2 * time.Second}
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		resp, err := client.Get(backendURL + "/health")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("backend not available after %v", timeout)
}

// getEnv returns environment variable value or default
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
