package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	
	"shebang.run/internal/database"
)

type tierKey string

const TierContextKey tierKey = "tier"

// TierMiddleware loads user's tier into context
func TierMiddleware(db *database.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := GetUserFromContext(r.Context())
			if !ok {
				next.ServeHTTP(w, r)
				return
			}
			
			tier, err := db.GetUserTier(claims.UserID)
			if err != nil {
				// Continue without tier (will use defaults)
				next.ServeHTTP(w, r)
				return
			}
			
			ctx := context.WithValue(r.Context(), TierContextKey, tier)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetTierFromContext retrieves tier from context
func GetTierFromContext(ctx context.Context) (*database.Tier, bool) {
	tier, ok := ctx.Value(TierContextKey).(*database.Tier)
	return tier, ok
}

// CheckFeature checks if user's tier allows a feature
func CheckFeature(ctx context.Context, feature string) bool {
	tier, ok := GetTierFromContext(ctx)
	if !ok {
		return false
	}
	
	// Check if user is admin (unlimited)
	claims, ok := GetUserFromContext(ctx)
	if ok && claims.IsAdmin {
		return true
	}
	
	// Check feature in JSON map
	return tier.Features[feature]
}
