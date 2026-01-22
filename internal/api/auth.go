package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"shebang.run/internal/auth"
	"shebang.run/internal/config"
	"shebang.run/internal/database"
	"shebang.run/internal/middleware"
)

type AuthHandler struct {
	db     *database.DB
	cfg    *config.Config
	github *auth.OAuthProvider
	google *auth.OAuthProvider
}

func NewAuthHandler(db *database.DB, cfg *config.Config) *AuthHandler {
	var github, google *auth.OAuthProvider
	
	if cfg.GitHubClientID != "" {
		github = auth.NewGitHubProvider(cfg.GitHubClientID, cfg.GitHubClientSecret, "http://localhost/api/auth/oauth/github/callback")
	}
	if cfg.GoogleClientID != "" {
		google = auth.NewGoogleProvider(cfg.GoogleClientID, cfg.GoogleClientSecret, "http://localhost/api/auth/oauth/google/callback")
	}
	
	return &AuthHandler{db: db, cfg: cfg, github: github, google: google}
}

type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token string      `json:"token"`
	User  interface{} `json:"user"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Username == "" || req.Email == "" || req.Password == "" {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	isFirst, err := h.db.IsFirstUser()
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	user, err := h.db.CreateUser(req.Username, req.Email, hash, "", "", isFirst)
	if err != nil {
		if strings.Contains(err.Error(), "Duplicate") {
			http.Error(w, "Username or email already exists", http.StatusConflict)
			return
		}
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	token, err := auth.GenerateToken(user.ID, user.Username, user.IsAdmin, h.cfg.JWTSecret)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AuthResponse{
		Token: token,
		User: map[string]interface{}{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
			"is_admin": user.IsAdmin,
		},
	})
}

func (h *AuthHandler) CheckUsername(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	if username == "" {
		http.Error(w, "Username required", http.StatusBadRequest)
		return
	}
	
	// Check if username exists
	existingUser, err := h.db.GetUserByUsername(username)
	available := err != nil // Available if user not found
	
	// If authenticated, allow current user's username
	if authHeader := r.Header.Get("Authorization"); authHeader != "" {
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := auth.ValidateToken(tokenString, h.cfg.JWTSecret)
		if err == nil && existingUser != nil && existingUser.ID == claims.UserID {
			available = true
		}
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"available": available})
}

func (h *AuthHandler) SetUsername(w http.ResponseWriter, r *http.Request) {
	// This endpoint is for OAuth users to set their username after first login
	// Extract user ID from temp token
	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	
	var req struct {
		Username string `json:"username"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	
	if req.Username == "" || len(req.Username) < 3 {
		http.Error(w, "Username must be at least 3 characters", http.StatusBadRequest)
		return
	}
	
	// Check if username is available (or is current user's username)
	existingUser, err := h.db.GetUserByUsername(req.Username)
	if err == nil && existingUser.ID != claims.UserID {
		http.Error(w, "Username already taken", http.StatusConflict)
		return
	}
	
	// Update username
	err = h.db.UpdateUsername(claims.UserID, req.Username)
	if err != nil {
		http.Error(w, "Failed to update username", http.StatusInternalServerError)
		return
	}
	
	// Get updated user
	user, err := h.db.GetUserByID(claims.UserID)
	if err != nil {
		http.Error(w, "Failed to get user", http.StatusInternalServerError)
		return
	}
	
	// Generate new token with updated username
	token, err := auth.GenerateToken(user.ID, user.Username, user.IsAdmin, h.cfg.JWTSecret)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AuthResponse{
		Token: token,
		User: map[string]interface{}{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
			"is_admin": user.IsAdmin,
		},
	})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	user, err := h.db.GetUserByUsername(req.Username)
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	if !auth.CheckPassword(req.Password, user.PasswordHash) {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	token, err := auth.GenerateToken(user.ID, user.Username, user.IsAdmin, h.cfg.JWTSecret)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AuthResponse{
		Token: token,
		User: map[string]interface{}{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
			"is_admin": user.IsAdmin,
		},
	})
}

func (h *AuthHandler) OAuthLogin(w http.ResponseWriter, r *http.Request, provider string) {
	var p *auth.OAuthProvider
	var callbackPath string
	
	switch provider {
	case "github":
		callbackPath = "/api/auth/oauth/github/callback"
		if h.github != nil {
			// Update redirect URL based on request
			redirectURL := buildRedirectURL(r, callbackPath)
			fmt.Printf("GitHub OAuth redirect URL: %s\n", redirectURL)
			p = auth.NewGitHubProvider(h.cfg.GitHubClientID, h.cfg.GitHubClientSecret, redirectURL)
		}
	case "google":
		callbackPath = "/api/auth/oauth/google/callback"
		if h.google != nil {
			redirectURL := buildRedirectURL(r, callbackPath)
			fmt.Printf("Google OAuth redirect URL: %s\n", redirectURL)
			p = auth.NewGoogleProvider(h.cfg.GoogleClientID, h.cfg.GoogleClientSecret, redirectURL)
		}
	default:
		http.Error(w, "Unknown provider", http.StatusBadRequest)
		return
	}

	if p == nil {
		http.Error(w, "Provider not configured", http.StatusServiceUnavailable)
		return
	}

	state, _ := auth.GenerateRandomToken(32)
	url := p.GetAuthURL(state)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (h *AuthHandler) OAuthCallback(w http.ResponseWriter, r *http.Request, provider string) {
	var p *auth.OAuthProvider
	var callbackPath string
	
	switch provider {
	case "github":
		callbackPath = "/api/auth/oauth/github/callback"
		if h.github != nil {
			redirectURL := buildRedirectURL(r, callbackPath)
			p = auth.NewGitHubProvider(h.cfg.GitHubClientID, h.cfg.GitHubClientSecret, redirectURL)
		}
	case "google":
		callbackPath = "/api/auth/oauth/google/callback"
		if h.google != nil {
			redirectURL := buildRedirectURL(r, callbackPath)
			p = auth.NewGoogleProvider(h.cfg.GoogleClientID, h.cfg.GoogleClientSecret, redirectURL)
		}
	default:
		http.Error(w, "Unknown provider", http.StatusBadRequest)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "No code provided", http.StatusBadRequest)
		return
	}

	token, err := p.Exchange(r.Context(), code)
	if err != nil {
		http.Error(w, "Failed to exchange token", http.StatusInternalServerError)
		return
	}

	oauthUser, err := p.GetUserInfo(r.Context(), token)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get user info: %v", err), http.StatusInternalServerError)
		return
	}

	isNewUser := false
	user, err := h.db.GetUserByOAuth(provider, oauthUser.ID)
	if err != nil {
		// Check if user exists with this email
		existingUser, emailErr := h.db.GetUserByEmail(oauthUser.Email)
		if emailErr == nil {
			// User exists with this email, log them in
			// This allows linking OAuth to existing password accounts
			user = existingUser
		} else {
			// Create new user
			isNewUser = true
			isFirst, _ := h.db.IsFirstUser()
			username := oauthUser.Username
			if username == "" {
				username = strings.Split(oauthUser.Email, "@")[0]
			}
			
			// Try to create user, handle duplicate username
			user, err = h.db.CreateUser(username, oauthUser.Email, "", provider, oauthUser.ID, isFirst)
			if err != nil {
				// If username exists, try with provider suffix
				if strings.Contains(err.Error(), "Duplicate") && strings.Contains(err.Error(), "username") {
					username = username + "_" + provider
					user, err = h.db.CreateUser(username, oauthUser.Email, "", provider, oauthUser.ID, isFirst)
				}
				if err != nil {
					http.Error(w, fmt.Sprintf("Failed to create user: %v", err), http.StatusInternalServerError)
					return
				}
			}
		}
	}

	jwtToken, err := auth.GenerateToken(user.ID, user.Username, user.IsAdmin, h.cfg.JWTSecret)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	// Check if username needs to be set (new OAuth user)
	redirectPath := "/dashboard"
	if isNewUser {
		redirectPath = "/select-username?temp_token=" + jwtToken
	}

	// Create HTML page that stores token and redirects
	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head><title>Login Success</title></head>
<body>
<script>
%s
window.location.href = '%s';
</script>
<p>Logging in...</p>
</body>
</html>
`, 
	func() string {
		if !isNewUser {
			return fmt.Sprintf(`localStorage.setItem('token', '%s');
localStorage.setItem('user', JSON.stringify({
	id: %d,
	username: '%s',
	email: '%s',
	is_admin: %t
}));`, jwtToken, user.ID, user.Username, user.Email, user.IsAdmin)
		}
		return ""
	}(),
	redirectPath)

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}


func buildRedirectURL(r *http.Request, path string) string {
	scheme := "https"
	if r.TLS == nil {
		scheme = "http"
	}
	// Check X-Forwarded-Proto header from reverse proxy
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}
	host := r.Host
	return scheme + "://" + host + path
}
