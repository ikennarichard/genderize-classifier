package middleware

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ikennarichard/genderize-classifier/internal/utils"
	"golang.org/x/time/rate"
)

type IPRateLimiter struct {
	ips map[string]*rate.Limiter
	mu  sync.Mutex
	r   rate.Limit
	b   int
}

func NewIPRateLimiter(r rate.Limit, b int) *IPRateLimiter {
	return &IPRateLimiter{
		ips: make(map[string]*rate.Limiter),
		r:   r,
		b:   b,
	}
}

func (i *IPRateLimiter) GetLimiter(ip string) *rate.Limiter {
	i.mu.Lock()
	defer i.mu.Unlock()

	limiter, exists := i.ips[ip]
	if !exists {
		limiter = rate.NewLimiter(i.r, i.b)
		i.ips[ip] = limiter
	}

	return limiter
}

func RateLimit(maxRequests int, windowSeconds int) func(http.Handler) http.Handler {
    type client struct {
        count    int
        resetAt  time.Time
    }

    var (
        mu      sync.Mutex
        clients = make(map[string]*client)
    )

    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            ip := r.RemoteAddr
            if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
                ip = strings.Split(forwarded, ",")[0]
            }

            mu.Lock()
            c, exists := clients[ip]
            if !exists || time.Now().After(c.resetAt) {
                clients[ip] = &client{
                    count:   1,
                    resetAt: time.Now().Add(time.Duration(windowSeconds) * time.Second),
                }
                mu.Unlock()
                next.ServeHTTP(w, r)
                return
            }
            c.count++
            if c.count > maxRequests {
                mu.Unlock()
                utils.RespondError(w, http.StatusTooManyRequests, "Rate limit exceeded — try again later")
                return
            }
            mu.Unlock()
            next.ServeHTTP(w, r)
        })
    }
}