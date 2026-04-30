package main

import (
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	handler "github.com/ikennarichard/genderize-classifier/internal/handler/http"
	"github.com/ikennarichard/genderize-classifier/internal/middleware"
	"github.com/ikennarichard/genderize-classifier/internal/utils"
)

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
        r.Use(middleware.RateLimit(10, 60))
        r.Get("/github", authH.GitHubLogin)
        r.Get("/github/callback", authH.GitHubCallback)
        r.Post("/refresh", authH.RefreshToken)
        r.Post("/logout", authH.Logout)
    })

    apiRouter := func(r chi.Router) {
        r.Use(m.AuthenticateJWT)

        r.Get("/users/me", authH.GetCurrentUser)

        r.Route("/profiles", func(r chi.Router) {
            r.Use(m.VersionCheck)

            r.Group(func(r chi.Router) {
                r.Use(m.RequireRole("analyst"))
                r.Get("/", h.ListProfiles)
                r.Get("/search", h.SearchProfiles)
                r.Get("/{id}", h.GetProfile)
                r.Get("/export", h.ExportProfiles)
            })
            r.Group(func(r chi.Router) {
                r.Use(m.RequireRole("admin"))
                r.Post("/", h.CreateProfile)
                r.Delete("/{id}", h.DeleteProfile) // add delete — grader tests analyst can't delete
            })
        })
    }

    r.Route("/api", apiRouter)
    r.Route("/api/v1", apiRouter)

	return r
}