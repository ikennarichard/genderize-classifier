package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ikennarichard/genderize-classifier/internal/domain"
	"github.com/ikennarichard/genderize-classifier/internal/utils"
)

type entry struct {
	count   int
	resetAt time.Time
}

type RateLimiter struct {
	mu           sync.Mutex
	clients      map[string]*entry
	maxRequests  int
	windowSeconds int
}

func newRateLimiter(maxRequests, windowSeconds int) *RateLimiter {
	rl := &RateLimiter{
		clients:       make(map[string]*entry),
		maxRequests:   maxRequests,
		windowSeconds: windowSeconds,
	}

	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			rl.mu.Lock()
			now := time.Now()
			for key, e := range rl.clients {
				if now.After(e.resetAt) {
					delete(rl.clients, key)
				}
			}
			rl.mu.Unlock()
		}
	}()

	return rl
}

func RateLimit(maxRequests, windowSeconds int) func(http.Handler) http.Handler {
	rl := newRateLimiter(maxRequests, windowSeconds)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := resolveKey(r)

			rl.mu.Lock()
			now := time.Now()
			c, exists := rl.clients[key]

			if !exists || now.After(c.resetAt) {
				rl.clients[key] = &entry{
					count:   1,
					resetAt: now.Add(time.Duration(rl.windowSeconds) * time.Second),
				}
				rl.mu.Unlock()

				setRateLimitHeaders(w, rl.maxRequests, rl.maxRequests-1)
				next.ServeHTTP(w, r)
				return
			}

			c.count++
			remaining := rl.maxRequests - c.count

			if c.count > rl.maxRequests {
				rl.mu.Unlock()

				w.Header().Set("Retry-After", fmt.Sprintf("%d", rl.windowSeconds))
				setRateLimitHeaders(w, rl.maxRequests, 0)
				utils.RespondError(w, http.StatusTooManyRequests, "Rate limit exceeded — try again later")
				return
			}

			rl.mu.Unlock()

			setRateLimitHeaders(w, rl.maxRequests, remaining)
			next.ServeHTTP(w, r)
		})
	}
}


func resolveKey(r *http.Request) string {

	if user, ok := r.Context().Value("user").(*domain.User); ok && user != nil {
		return "user:" + user.ID
	}

	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		return "ip:" + strings.TrimSpace(strings.Split(forwarded, ",")[0])
	}

	return "ip:" + r.RemoteAddr
}

func setRateLimitHeaders(w http.ResponseWriter, limit, remaining int) {
	w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
	w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
}