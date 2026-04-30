package main

import (
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
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
		AllowedOrigins:   []string{"http://localhost:5500", "http://127.0.0.1:5500"},
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

	// seed data
	if err := repository.SeedFromJSON(pool, ctx, "seed_profiles.json"); err != nil {
		log.Printf("seeding failed: %v", err)
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

func RegisterRoutes(r *chi.Mux, h *handler.ProfileHandler, authH *handler.AuthHandler, m *middleware.Middleware) http.Handler {

	r.Use(chimiddleware.Logger)    
	r.Use(chimiddleware.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			utils.Respond(w, http.StatusOK, map[string]string{
					"status": "ok",
					"env":    os.Getenv("ENV"),
			})
	})

	r.Route("/auth", func(r chi.Router) {
		r.Use(middleware.RateLimit(60, 60))
		r.Get("/github", authH.GitHubLogin)
		r.Get("/github/callback", authH.GitHubCallback)
		r.Post("/refresh", authH.RefreshToken)
		r.Post("/logout", authH.Logout)
	})

	r.Route("/api/v1", func(r chi.Router) {
		r.Use(middleware.RateLimit(60, 60))
		r.Use(m.AuthenticateJWT) 
		r.Use(m.ValidateCSRF) 
		r.Use(m.RequireRole("analyst"))
		
		r.Get("/me", authH.GetCurrentUser)
		
		r.Route("/profiles", func(r chi.Router) {
			r.Use(m.VersionCheck)
			// analyst access: read only routes
			r.Group(func(r chi.Router) {
				r.Get("/", h.ListProfiles)
				r.Get("/search", h.SearchProfiles)
				r.Get("/{id}", h.GetProfile)
			})

			// admin access: Write & Export
			r.Group(func(r chi.Router) {
				r.Use(m.RequireRole("admin"))
				r.Post("/", h.CreateProfile)
				r.Get("/export", h.ExportProfiles)
			})
		})
	})

	return r
}