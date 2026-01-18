package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"shebang.run/internal/auth"
	"shebang.run/internal/config"
	"shebang.run/internal/database"
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
	switch provider {
	case "github":
		p = h.github
	case "google":
		p = h.google
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
	switch provider {
	case "github":
		p = h.github
	case "google":
		p = h.google
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
		http.Error(w, "Failed to get user info", http.StatusInternalServerError)
		return
	}

	user, err := h.db.GetUserByOAuth(provider, oauthUser.ID)
	if err != nil {
		isFirst, _ := h.db.IsFirstUser()
		username := oauthUser.Username
		if username == "" {
			username = strings.Split(oauthUser.Email, "@")[0]
		}
		user, err = h.db.CreateUser(username, oauthUser.Email, "", provider, oauthUser.ID, isFirst)
		if err != nil {
			http.Error(w, "Failed to create user", http.StatusInternalServerError)
			return
		}
	}

	jwtToken, err := auth.GenerateToken(user.ID, user.Username, user.IsAdmin, h.cfg.JWTSecret)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AuthResponse{
		Token: jwtToken,
		User: map[string]interface{}{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
			"is_admin": user.IsAdmin,
		},
	})
}
