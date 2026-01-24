package middleware

import (
	"context"
	"net/http"
	"time"
	
	"shebang.run/internal/database"
)

type tierKey string

const TierContextKey tierKey = "tier"

// TierMiddleware loads user's tier into context and checks expiration
func TierMiddleware(db *database.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := GetUserFromContext(r.Context())
			if !ok {
				next.ServeHTTP(w, r)
				return
			}
			
			// Admins bypass expiration checks
			if !claims.IsAdmin {
				// Check if subscription expired (from JWT)
				if claims.SubscriptionExpiry != nil && time.Now().After(*claims.SubscriptionExpiry) {
					// Downgrade to Free tier
					db.UpdateUserTier(claims.UserID, 1)
					db.Exec("UPDATE users SET subscription_expires_at = NULL, subscription_managed_by = 'manual' WHERE id = ?", claims.UserID)
					
					// Force re-login by returning unauthorized
					http.Error(w, "Subscription expired. Please log in again.", http.StatusUnauthorized)
					return
				}
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
