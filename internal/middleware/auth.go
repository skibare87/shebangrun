package middleware

import (
	"context"
	"encoding/base64"
	"net/http"
	"strings"

	"shebang.run/internal/auth"
	"shebang.run/internal/database"
)

type contextKey string

const UserContextKey contextKey = "user"

func AuthMiddleware(jwtSecret string, db *database.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Check if it's Basic Auth (API tokens)
			if strings.HasPrefix(authHeader, "Basic ") {
				encoded := strings.TrimPrefix(authHeader, "Basic ")
				decoded, err := base64.StdEncoding.DecodeString(encoded)
				if err != nil {
					http.Error(w, "Invalid authorization", http.StatusUnauthorized)
					return
				}
				
				parts := strings.SplitN(string(decoded), ":", 2)
				if len(parts) != 2 {
					http.Error(w, "Invalid authorization", http.StatusUnauthorized)
					return
				}
				
				clientID := parts[0]
				clientSecret := parts[1]
				
				// Validate API token
				token, err := db.GetAPITokenByClientID(clientID)
				if err != nil || token.ClientSecret != clientSecret {
					http.Error(w, "Invalid credentials", http.StatusUnauthorized)
					return
				}
				
				// Update last used
				db.UpdateAPITokenLastUsed(clientID)
				
				// Get user
				user, err := db.GetUserByID(token.UserID)
				if err != nil {
					http.Error(w, "User not found", http.StatusUnauthorized)
					return
				}
				
				// Create claims
				claims := &auth.Claims{
					UserID:   user.ID,
					Username: user.Username,
					IsAdmin:  user.IsAdmin,
				}
				
				ctx := context.WithValue(r.Context(), UserContextKey, claims)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// Bearer token (JWT)
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, "Invalid authorization header", http.StatusUnauthorized)
				return
			}

			claims, err := auth.ValidateToken(parts[1], jwtSecret)
			if err != nil {
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), UserContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// APITokenMiddleware is deprecated - use AuthMiddleware which handles both
func APITokenMiddleware(db *database.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	}
}

func AdminMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := r.Context().Value(UserContextKey).(*auth.Claims)
		if !ok || !claims.IsAdmin {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func GetUserFromContext(ctx context.Context) (*auth.Claims, bool) {
	claims, ok := ctx.Value(UserContextKey).(*auth.Claims)
	return claims, ok
}
