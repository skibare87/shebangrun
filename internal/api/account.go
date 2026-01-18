package api

import (
	"encoding/json"
	"net/http"
	"time"

	"shebang.run/internal/auth"
	"shebang.run/internal/config"
	"shebang.run/internal/database"
	"shebang.run/internal/middleware"
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
