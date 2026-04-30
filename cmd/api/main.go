package main

import (
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/ikennarichard/genderize-classifier/internal/config"
	handler "github.com/ikennarichard/genderize-classifier/internal/handler/http"
	"github.com/ikennarichard/genderize-classifier/internal/middleware"
	"github.com/ikennarichard/genderize-classifier/internal/repository"
	"github.com/ikennarichard/genderize-classifier/internal/service"
	"github.com/ikennarichard/genderize-classifier/internal/utils"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

func main() {
	config.InitLogger()
	pool, ctx := config.Load()
	defer pool.Close()

	r := chi.NewRouter()

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5500", "http://127.0.0.1:5500", "https://ikennarichard.github.io/insighta-web"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "X-API-Version", "X-CSRF-Token", "Authorization"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	profileRepo := repository.NewPostgresProfileRepository(pool)
	userRepo := repository.NewPostgresUserRepository(pool)
	sessionRepo := repository.NewPostgresSessionRepository(pool)
	tokenService := service.NewTokenService(os.Getenv("JWT_SECRET"), userRepo, sessionRepo)

	// oauth config
	oauthConfig := &oauth2.Config{
		ClientID:     os.Getenv("GITHUB_CLIENT_ID"),
		ClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("GITHUB_REDIRECT_URL"),
		 Scopes:       []string{"read:user", "user:email"},
		Endpoint:     github.Endpoint,
	}

	// handlers
	profileHandler := handler.New(profileRepo)
	authHandler := handler.NewAuthHandler(oauthConfig, tokenService, userRepo, sessionRepo)

	m := middleware.NewMiddleware(tokenService, userRepo, sessionRepo)
	router := RegisterRoutes(r, profileHandler, authHandler, m)

	// if os.Getenv("ENV") != "production" {
    r.Get("/dev/analyst-token", func(w http.ResponseWriter, r *http.Request) {
        analyst, err := userRepo.FindByUsername(r.Context(), "insighta_analyst")
        if err != nil {
            utils.RespondError(w, http.StatusNotFound, "Analyst user not seeded")
            return
        }
        access, _, err := tokenService.GenerateTokenPair(r.Context(), analyst)
        if err != nil {
            utils.RespondError(w, http.StatusInternalServerError, "Failed to generate token")
            return
        }
        utils.Respond(w, http.StatusOK, map[string]string{
            "analyst_token": access,
        })
    })
// }

	// seed data
	if err := repository.SeedFromJSON(pool, ctx, "seed_profiles.json"); err != nil {
		log.Printf("seeding failed: %v", err)
	}

	if os.Getenv("ENV") != "production" || os.Getenv("SEED_TEST_USERS") == "true" {
    if err := repository.SeedTestUsers(pool, ctx); err != nil {
        log.Printf("test user seeding failed: %v", err)
    }
}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	slog.Info("Starting server", "port", port, "env", os.Getenv("ENV"))

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      middleware.StructuredLogger(router),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("Server listening on :%s", port)
	if err := server.ListenAndServe(); err != nil {
		slog.Error("Server failed to start", "error", err)
		os.Exit(1)
	}
}