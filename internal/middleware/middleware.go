package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/ikennarichard/genderize-classifier/internal/domain"
	"github.com/ikennarichard/genderize-classifier/internal/repository"
	"github.com/ikennarichard/genderize-classifier/internal/service"
	"github.com/ikennarichard/genderize-classifier/internal/utils"
)

type Middleware struct {
	tokenService *service.TokenService
	userRepo     domain.UserRepository
	sessionRepo repository.SessionRepository
}

func NewMiddleware(ts *service.TokenService, ur domain.UserRepository, sr repository.SessionRepository) *Middleware {
	return &Middleware{
		tokenService: ts,
		userRepo:     ur,
		sessionRepo: sr,
	}
}

func (m *Middleware) AuthenticateJWT(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			  var tokenString string

        authHeader := r.Header.Get("Authorization")
        if strings.HasPrefix(authHeader, "Bearer ") {
            tokenString = strings.TrimPrefix(authHeader, "Bearer ")
        } else {
            atCookie, err := r.Cookie("at")
            if err != nil {
                utils.RespondError(w, http.StatusUnauthorized, "Missing or invalid authorization")
                return
            }
            tokenString = atCookie.Value
        }

        claims, err := m.tokenService.ValidateAccessToken(tokenString)
        if err != nil {
            utils.RespondError(w, http.StatusUnauthorized, "Invalid or expired token")
            return
        }

        user, err := m.userRepo.FindByID(claims.UserID)
        if err != nil {
            utils.RespondError(w, http.StatusUnauthorized, "User session no longer valid")
            return
        }

        if !user.IsActive {
            _ = m.sessionRepo.RevokeAllUserSessions(r.Context(), user.ID)
            utils.RespondError(w, http.StatusForbidden, "Your account has been deactivated")
            return
        }

        ctx := context.WithValue(r.Context(), "user", user)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}


func (m *Middleware) VersionCheck(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-Version") != "1" {
			utils.Respond(w, http.StatusBadRequest, map[string]string{
				"status":  "error",
				"message": "API version header required",
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (m *Middleware) RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := r.Context().Value("user").(*domain.User)
			
			if role == "admin" && user.Role != "admin" {
				utils.RespondError(w, http.StatusForbidden, "Admin access required")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (m *Middleware) ValidateCSRF(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// CSRF only applies to state-changing methods
		if r.Method == http.MethodGet || r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}

		// Read token from cookie
		_, err := r.Cookie("csrf_token")
		if err != nil {
			utils.RespondError(w, http.StatusForbidden, "Missing CSRF cookie")
			return
		}

		next.ServeHTTP(w, r)
	})
}