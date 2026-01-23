package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
	
	"github.com/go-chi/chi/v5"
	"shebang.run/internal/crypto"
	"shebang.run/internal/middleware"
)

type SecretsHandler struct {
	db          *sql.DB
	udekManager *crypto.UDEKManager
}

func NewSecretsHandler(db *sql.DB, udekManager *crypto.UDEKManager) *SecretsHandler {
	return &SecretsHandler{
		db:          db,
		udekManager: udekManager,
	}
}

type CreateSecretRequest struct {
	KeyName   string     `json:"key_name"`
	Value     string     `json:"value"`
	ExpiresAt *time.Time `json:"expires_at"`
}

type SecretResponse struct {
	ID           int64      `json:"id"`
	KeyName      string     `json:"key_name"`
	Version      int        `json:"version"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	LastAccessed *time.Time `json:"last_accessed"`
	ExpiresAt    *time.Time `json:"expires_at"`
}

func (h *SecretsHandler) List(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	
	rows, err := h.db.Query(`
		SELECT id, key_name, version, created_at, updated_at, last_accessed, expires_at
		FROM secrets WHERE user_id = ? ORDER BY key_name
	`, claims.UserID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	
	var secrets []SecretResponse
	for rows.Next() {
		var s SecretResponse
		if err := rows.Scan(&s.ID, &s.KeyName, &s.Version, &s.CreatedAt, &s.UpdatedAt, &s.LastAccessed, &s.ExpiresAt); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		secrets = append(secrets, s)
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(secrets)
}

func (h *SecretsHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	
	var req CreateSecretRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	// Check tier limits (admins bypass)
	if !claims.IsAdmin {
		tier, ok := middleware.GetTierFromContext(r.Context())
		if ok {
			// Check secret count
			var count int
			h.db.QueryRow("SELECT COUNT(*) FROM secrets WHERE user_id = ?", claims.UserID).Scan(&count)
			if count >= tier.MaxSecrets {
				http.Error(w, fmt.Sprintf("Secret limit reached (%d). Upgrade your tier.", tier.MaxSecrets), http.StatusForbidden)
				return
			}
		}
	}
	
	// Get UDEK
	udek, err := h.udekManager.GetOrCreateUDEK(claims.UserID)
	if err != nil {
		log.Printf("Error getting UDEK for user %d: %v", claims.UserID, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	// Encrypt value
	encrypted, err := crypto.EncryptWithUDEK([]byte(req.Value), udek)
	if err != nil {
		log.Printf("Error encrypting secret: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	// Store
	result, err := h.db.Exec(`
		INSERT INTO secrets (user_id, key_name, encrypted_value, expires_at)
		VALUES (?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE 
			encrypted_value = VALUES(encrypted_value),
			version = version + 1,
			updated_at = CURRENT_TIMESTAMP,
			expires_at = VALUES(expires_at)
	`, claims.UserID, req.KeyName, encrypted, req.ExpiresAt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	id, _ := result.LastInsertId()
	
	// Audit log
	h.logAccess(id, claims.UserID, "write", r)
	
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":       id,
		"key_name": req.KeyName,
	})
}

func (h *SecretsHandler) GetValue(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	
	keyName := chi.URLParam(r, "name")
	
	var id int64
	var encrypted []byte
	err := h.db.QueryRow(`
		SELECT id, encrypted_value FROM secrets 
		WHERE user_id = ? AND key_name = ?
		AND (expires_at IS NULL OR expires_at > NOW())
	`, claims.UserID, keyName).Scan(&id, &encrypted)
	if err == sql.ErrNoRows {
		http.Error(w, "Secret not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	// Get UDEK
	udek, err := h.udekManager.GetOrCreateUDEK(claims.UserID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	// Decrypt
	value, err := crypto.DecryptWithUDEK(encrypted, udek)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	// Update last accessed
	h.db.Exec("UPDATE secrets SET last_accessed = NOW() WHERE id = ?", id)
	
	// Audit log
	h.logAccess(id, claims.UserID, "read", r)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"value": string(value)})
}

func (h *SecretsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	
	keyName := chi.URLParam(r, "name")
	
	var id int64
	err := h.db.QueryRow("SELECT id FROM secrets WHERE user_id = ? AND key_name = ?", claims.UserID, keyName).Scan(&id)
	if err == sql.ErrNoRows {
		http.Error(w, "Secret not found", http.StatusNotFound)
		return
	}
	
	// Audit log before delete
	h.logAccess(id, claims.UserID, "delete", r)
	
	_, err = h.db.Exec("DELETE FROM secrets WHERE id = ?", id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.WriteHeader(http.StatusNoContent)
}

func (h *SecretsHandler) GetAuditLog(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	
	keyName := chi.URLParam(r, "name")
	
	var secretID int64
	err := h.db.QueryRow("SELECT id FROM secrets WHERE user_id = ? AND key_name = ?", claims.UserID, keyName).Scan(&secretID)
	if err != nil {
		http.Error(w, "Secret not found", http.StatusNotFound)
		return
	}
	
	rows, err := h.db.Query(`
		SELECT action, ip_address, user_agent, accessed_at
		FROM secrets_audit WHERE secret_id = ?
		ORDER BY accessed_at DESC LIMIT 100
	`, secretID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	
	var logs []map[string]interface{}
	for rows.Next() {
		var action, ip, ua string
		var accessedAt time.Time
		if err := rows.Scan(&action, &ip, &ua, &accessedAt); err != nil {
			continue
		}
		logs = append(logs, map[string]interface{}{
			"action":      action,
			"ip_address":  ip,
			"user_agent":  ua,
			"accessed_at": accessedAt,
		})
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}

func (h *SecretsHandler) logAccess(secretID, userID int64, action string, r *http.Request) {
	ip := r.RemoteAddr
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		ip = forwarded
	}
	
	h.db.Exec(`
		INSERT INTO secrets_audit (secret_id, user_id, action, ip_address, user_agent)
		VALUES (?, ?, ?, ?, ?)
	`, secretID, userID, action, ip, r.UserAgent())
}
