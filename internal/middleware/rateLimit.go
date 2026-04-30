package middleware

import (
	"net/http"
	"sync"

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


func RateLimit(limit int, burst int) func(http.Handler) http.Handler {
	limiter := NewIPRateLimiter(rate.Limit(float64(limit)/60), burst)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr
			
			if !limiter.GetLimiter(ip).Allow() {
				utils.RespondError(w, http.StatusTooManyRequests, "Rate limit exceeded. Try again later.")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}