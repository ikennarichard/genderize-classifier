package main

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/ikennarichard/genderize-classifier/internal/config"
	handler "github.com/ikennarichard/genderize-classifier/internal/handler/http"
	"github.com/ikennarichard/genderize-classifier/internal/repository"
)

func main() {
	pool, ctx := config.Load()
	defer pool.Close()
	
	profileRepo := repository.NewPostgresProfileRepository(pool)
	profileHandler := handler.New(profileRepo)

	err := profileRepo.SeedFromJSON(ctx, "seed_profiles.json")
	if err != nil {
			fmt.Println("seeding failed", "error", err)
	}

	mux := http.NewServeMux()
	profileHandler.RegisterProfileRoutes(mux)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      withCORS(loggingMiddleware(mux)),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("Server listening on :%s", port)
	log.Fatal(server.ListenAndServe())
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		next.ServeHTTP(w, r)

		slog.Info("request completed",
			"method", r.Method,
			"path", r.URL.Path,
			"duration", time.Since(start),
			"remote_addr", r.RemoteAddr,
		)
	})
}