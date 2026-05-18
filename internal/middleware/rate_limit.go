package middleware

import (
	"net/http"
	"sync"
	"time"
)

type RateLimiter struct {
	mu       sync.Mutex
	requests map[string][]time.Time
	limit    int
	window   time.Duration
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr
		rl.mu.Lock()
		now := time.Now()
		cutoff := now.Add(-rl.window)
		filtered := rl.requests[ip][:0]
		for _, t := range rl.requests[ip] {
			if t.After(cutoff) {
				filtered = append(filtered, t)
			}
		}
		if len(filtered) >= rl.limit {
			rl.mu.Unlock()
			http.Error(w, "too many requests", http.StatusTooManyRequests)
			return
		}
		rl.requests[ip] = append(filtered, now)
		rl.mu.Unlock()
		next.ServeHTTP(w, r)
	})
}
