package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"shebang.run/internal/auth"
	"shebang.run/internal/config"
	"shebang.run/internal/database"
	"shebang.run/internal/middleware"

	"github.com/go-chi/chi/v5"
)

type AccountHandler struct {
	db  *database.DB
	cfg *config.Config
}

func NewAccountHandler(db *database.DB, cfg *config.Config) *AccountHandler {
	return &AccountHandler{db: db, cfg: cfg}
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

func (h *AccountHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	user, err := h.db.GetUserByID(claims.UserID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	if !auth.CheckPassword(req.CurrentPassword, user.PasswordHash) {
		http.Error(w, "Current password is incorrect", http.StatusUnauthorized)
		return
	}

	newHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		http.Error(w, "Failed to hash password", http.StatusInternalServerError)
		return
	}

	_, err = h.db.Exec("UPDATE users SET password_hash = ? WHERE id = ?", newHash, claims.UserID)
	if err != nil {
		http.Error(w, "Failed to update password", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

func (h *AccountHandler) GetTier(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	
	tier, err := h.db.GetUserTier(claims.UserID)
	if err != nil {
		http.Error(w, "Failed to get tier", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tier)
}

func (h *AccountHandler) ExportData(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	user, _ := h.db.GetUserByID(claims.UserID)
	scripts, _ := h.db.GetScriptsByUserID(claims.UserID)
	keys, _ := h.db.GetKeyPairsByUserID(claims.UserID)

	export := map[string]interface{}{
		"user":    user,
		"scripts": scripts,
		"keys":    keys,
		"exported_at": time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=shebang-data-export.json")
	json.NewEncoder(w).Encode(export)
}

func (h *AccountHandler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	_, err := h.db.Exec("DELETE FROM users WHERE id = ?", claims.UserID)
	if err != nil {
		http.Error(w, "Failed to delete account", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"deleted"}`))
}

type APITokenResponse struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	ClientID    string `json:"client_id"`
	ClientSecret string `json:"client_secret,omitempty"` // Only shown on creation
	CreatedAt   string `json:"created_at"`
	LastUsed    string `json:"last_used,omitempty"`
}

func (h *AccountHandler) ListAPITokens(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	tokens, err := h.db.GetAPITokensByUserID(claims.UserID)
	if err != nil {
		http.Error(w, "Failed to fetch tokens", http.StatusInternalServerError)
		return
	}

	var response []APITokenResponse
	for _, t := range tokens {
		lastUsed := ""
		if t.LastUsed.Valid {
			lastUsed = t.LastUsed.String
		}
		response = append(response, APITokenResponse{
			ID:        t.ID,
			Name:      t.Name,
			ClientID:  t.ClientID,
			CreatedAt: t.CreatedAt,
			LastUsed:  lastUsed,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *AccountHandler) CreateAPIToken(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}

	// Generate client ID and secret
	clientID, _ := auth.GenerateRandomToken(32)
	clientSecret, _ := auth.GenerateRandomToken(48)

	token, err := h.db.CreateAPIToken(claims.UserID, req.Name, clientID, clientSecret)
	if err != nil {
		http.Error(w, "Failed to create token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(APITokenResponse{
		ID:           token.ID,
		Name:         token.Name,
		ClientID:     token.ClientID,
		ClientSecret: token.ClientSecret, // Only shown once
		CreatedAt:    token.CreatedAt,
	})
}

func (h *AccountHandler) DeleteAPIToken(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid token ID", http.StatusBadRequest)
		return
	}

	if err := h.db.DeleteAPIToken(id, claims.UserID); err != nil {
		http.Error(w, "Failed to delete token", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
