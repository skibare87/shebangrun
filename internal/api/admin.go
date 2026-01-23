package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"shebang.run/internal/auth"
	"shebang.run/internal/config"
	"shebang.run/internal/database"
	"shebang.run/internal/middleware"

	"github.com/go-chi/chi/v5"
)

type AdminHandler struct {
	db  *database.DB
	cfg *config.Config
}

func NewAdminHandler(db *database.DB, cfg *config.Config) *AdminHandler {
	return &AdminHandler{db: db, cfg: cfg}
}

type UserListResponse struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	IsAdmin   bool   `json:"is_admin"`
	TierID    int64  `json:"tier_id"`
	TierName  string `json:"tier_name"`
	RateLimit int    `json:"rate_limit"`
	CreatedAt string `json:"created_at"`
}

type SetLimitsRequest struct {
	IsAdmin            *bool      `json:"is_admin"`
	TierID             *int64     `json:"tier_id"`
	SubscriptionExpiry *string    `json:"subscription_expiry"` // ISO date or null for permanent
	MaxScripts         *int       `json:"max_scripts"`
	MaxScriptSize      *int64     `json:"max_script_size"`
	RateLimit          *int       `json:"rate_limit"`
}

type ConfigResponse struct {
	DefaultRateLimit     int   `json:"default_rate_limit"`
	DefaultMaxScripts    int   `json:"default_max_scripts"`
	DefaultMaxScriptSize int64 `json:"default_max_script_size"`
}

func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")
	
	limit := 50
	offset := 0
	
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil {
			offset = o
		}
	}

	users, err := h.db.ListUsers(limit, offset)
	if err != nil {
		http.Error(w, "Failed to fetch users", http.StatusInternalServerError)
		return
	}

	var response []UserListResponse
	for _, u := range users {
		// Get tier name
		tier, _ := h.db.GetUserTier(u.ID)
		tierName := "Unknown"
		if tier != nil {
			tierName = tier.DisplayName
		}
		
		response = append(response, UserListResponse{
			ID:        u.ID,
			Username:  u.Username,
			Email:     u.Email,
			IsAdmin:   u.IsAdmin,
			TierID:    u.TierID,
			TierName:  tierName,
			RateLimit: u.RateLimit,
			CreatedAt: u.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *AdminHandler) SetUserLimits(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	userID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	var req SetLimitsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Failed to decode limits request: %v", err)
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.MaxScripts != nil || req.MaxScriptSize != nil {
		if err := h.db.SetUserLimits(userID, req.MaxScripts, req.MaxScriptSize); err != nil {
			http.Error(w, "Failed to set limits", http.StatusInternalServerError)
			return
		}
	}

	if req.RateLimit != nil {
		if _, err := h.db.Exec("UPDATE users SET rate_limit = ? WHERE id = ?", *req.RateLimit, userID); err != nil {
			http.Error(w, "Failed to set rate limit", http.StatusInternalServerError)
			return
		}
	}
	
	if req.IsAdmin != nil {
		if _, err := h.db.Exec("UPDATE users SET is_admin = ? WHERE id = ?", *req.IsAdmin, userID); err != nil {
			http.Error(w, "Failed to set admin status", http.StatusInternalServerError)
			return
		}
	}
	
	if req.TierID != nil {
		if err := h.db.UpdateUserTier(userID, *req.TierID); err != nil {
			http.Error(w, "Failed to set tier", http.StatusInternalServerError)
			return
		}
		
		// Set subscription management
		var expiresAt interface{} = nil
		if req.SubscriptionExpiry != nil && *req.SubscriptionExpiry != "" {
			expiresAt = *req.SubscriptionExpiry
		}
		
		h.db.Exec(`
			UPDATE users 
			SET subscription_managed_by = 'admin', subscription_expires_at = ?
			WHERE id = ?
		`, expiresAt, userID)
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

func (h *AdminHandler) ListTiers(w http.ResponseWriter, r *http.Request) {
	tiers, err := h.db.GetAllTiers()
	if err != nil {
		http.Error(w, "Failed to get tiers", http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tiers)
}

func (h *AdminHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	response := ConfigResponse{
		DefaultRateLimit:     h.cfg.DefaultRateLimit,
		DefaultMaxScripts:    h.cfg.DefaultMaxScripts,
		DefaultMaxScriptSize: h.cfg.DefaultMaxScriptSize,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *AdminHandler) ResetUserPassword(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	userID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	var req struct {
		NewPassword string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	hash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		http.Error(w, "Failed to hash password", http.StatusInternalServerError)
		return
	}

	_, err = h.db.Exec("UPDATE users SET password_hash = ? WHERE id = ?", hash, userID)
	if err != nil {
		http.Error(w, "Failed to reset password", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}

func (h *AdminHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	userID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Don't allow deleting yourself
	claims, _ := middleware.GetUserFromContext(r.Context())
	if claims.UserID == userID {
		http.Error(w, "Cannot delete your own account", http.StatusForbidden)
		return
	}

	_, err = h.db.Exec("DELETE FROM users WHERE id = ?", userID)
	if err != nil {
		http.Error(w, "Failed to delete user", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}
