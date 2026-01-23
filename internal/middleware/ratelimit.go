package middleware

import (
	"net/http"
	"sync"
	"time"
)

type rateLimiter struct {
	requests map[string][]time.Time
	mu       sync.RWMutex
	limit    int
	window   time.Duration
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	rl := &rateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
	
	go rl.cleanup()
	
	return rl
}

func (rl *rateLimiter) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for key, times := range rl.requests {
			var valid []time.Time
			for _, t := range times {
				if now.Sub(t) < rl.window {
					valid = append(valid, t)
				}
			}
			if len(valid) == 0 {
				delete(rl.requests, key)
			} else {
				rl.requests[key] = valid
			}
		}
		rl.mu.Unlock()
	}
}

func (rl *rateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	now := time.Now()
	times := rl.requests[key]
	
	var valid []time.Time
	for _, t := range times {
		if now.Sub(t) < rl.window {
			valid = append(valid, t)
		}
	}
	
	if len(valid) >= rl.limit {
		return false
	}
	
	valid = append(valid, now)
	rl.requests[key] = valid
	
	return true
}

var globalRateLimiter *rateLimiter

func RateLimitMiddleware(limit int) func(http.Handler) http.Handler {
	if globalRateLimiter == nil {
		globalRateLimiter = newRateLimiter(limit, time.Minute)
	}
	
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if user is admin (bypass rate limiting)
			if claims, ok := GetUserFromContext(r.Context()); ok && claims.IsAdmin {
				next.ServeHTTP(w, r)
				return
			}
			
			ip := r.RemoteAddr
			
			if !globalRateLimiter.allow(ip) {
				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}
			
			next.ServeHTTP(w, r)
		})
	}
}
