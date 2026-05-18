package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(ww, r)
		slog.Info("http",
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.status,
			"dur", time.Since(start).String(),
		)
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (s *statusWriter) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

// Flush forwards to the underlying writer so SSE handlers can stream.
// Embedding ResponseWriter is not enough — the http.Flusher type
// assertion checks the concrete wrapper, which doesn't satisfy the
// interface unless we declare the method ourselves.
func (s *statusWriter) Flush() {
	if f, ok := s.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}
