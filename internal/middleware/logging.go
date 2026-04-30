package middleware

import (
	"log/slog"
	"net/http"
	"time"
)


type responseWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if rw.wroteHeader { return }
	rw.status = code
	rw.wroteHeader = true
	rw.ResponseWriter.WriteHeader(code)
}

func StructuredLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := &responseWriter{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		slog.Info("http_request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", wrapped.status,
			"latency", time.Since(start).String(),
			"ip", r.RemoteAddr,
		)
	})
}