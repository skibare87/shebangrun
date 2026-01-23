package middleware

import (
	"net/http"
	"sync"
	"time"
	
	"shebang.run/internal/database"
)

type rateLimiter struct {
	requests map[string][]time.Time
	mu       sync.RWMutex
	window   time.Duration
}

func newRateLimiter(window time.Duration) *rateLimiter {
	rl := &rateLimiter{
		requests: make(map[string][]time.Time),
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

func (rl *rateLimiter) allow(key string, limit int) bool {
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
	
	if len(valid) >= limit {
		return false
	}
	
	valid = append(valid, now)
	rl.requests[key] = valid
	
	return true
}

var globalRateLimiter *rateLimiter

func RateLimitMiddleware(defaultLimit int, db *database.DB) func(http.Handler) http.Handler {
	if globalRateLimiter == nil {
		globalRateLimiter = newRateLimiter(time.Minute)
	}
	
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if user is admin (bypass rate limiting)
			if claims, ok := GetUserFromContext(r.Context()); ok && claims.IsAdmin {
				next.ServeHTTP(w, r)
				return
			}
			
			// Get user's tier limit (or override)
			limit := defaultLimit
			if claims, ok := GetUserFromContext(r.Context()); ok {
				// Check for user-specific override first
				var userRateLimit *int
				db.DB.QueryRow("SELECT rate_limit FROM users WHERE id = ?", claims.UserID).Scan(&userRateLimit)
				
				if userRateLimit != nil && *userRateLimit > 0 {
					limit = *userRateLimit
				} else if tier, err := db.GetUserTier(claims.UserID); err == nil {
					limit = tier.RateLimit
				}
			}
			
			ip := r.RemoteAddr
			
			if !globalRateLimiter.allow(ip, limit) {
				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}
			
			next.ServeHTTP(w, r)
		})
	}
}
