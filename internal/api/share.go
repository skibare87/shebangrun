package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"
	
	"github.com/go-chi/chi/v5"
	"shebang.run/internal/database"
	"shebang.run/internal/middleware"
)

type ShareHandler struct {
	db *sql.DB
}

func NewShareHandler(db *sql.DB) *ShareHandler {
	return &ShareHandler{db: db}
}

type ShareRequest struct {
	AccessType string   `json:"access_type"` // 'link' or 'user'
	Usernames  []string `json:"usernames"`   // For 'user' type
	ExpiresAt  *time.Time `json:"expires_at"`
}

type ShareResponse struct {
	ID         int64      `json:"id"`
	AccessType string     `json:"access_type"`
	Username   string     `json:"username,omitempty"`
	GrantedAt  time.Time  `json:"granted_at"`
	ExpiresAt  *time.Time `json:"expires_at"`
}

func (h *ShareHandler) GetAccess(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	
	scriptID := chi.URLParam(r, "id")
	
	// Verify ownership
	var ownerID int64
	err := h.db.QueryRow("SELECT user_id FROM scripts WHERE id = ?", scriptID).Scan(&ownerID)
	if err != nil || ownerID != claims.UserID {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	
	// Get access list
	rows, err := h.db.Query(`
		SELECT sa.id, sa.access_type, COALESCE(u.username, ''), sa.granted_at, sa.expires_at
		FROM script_access sa
		LEFT JOIN users u ON sa.user_id = u.id
		WHERE sa.script_id = ?
		ORDER BY sa.granted_at DESC
	`, scriptID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	
	var shares []ShareResponse
	for rows.Next() {
		var s ShareResponse
		if err := rows.Scan(&s.ID, &s.AccessType, &s.Username, &s.GrantedAt, &s.ExpiresAt); err != nil {
			continue
		}
		shares = append(shares, s)
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(shares)
}

func (h *ShareHandler) AddAccess(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	
	scriptID := chi.URLParam(r, "id")
	
	// Verify ownership
	var ownerID int64
	err := h.db.QueryRow("SELECT user_id FROM scripts WHERE id = ?", scriptID).Scan(&ownerID)
	if err != nil || ownerID != claims.UserID {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	
	var req ShareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	
	if req.AccessType == "link" {
		// Add "anyone with link" access
		_, err := h.db.Exec(`
			INSERT INTO script_access (script_id, access_type, granted_by, expires_at)
			VALUES (?, 'link', ?, ?)
			ON DUPLICATE KEY UPDATE expires_at = VALUES(expires_at)
		`, scriptID, claims.UserID, req.ExpiresAt)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else if req.AccessType == "user" {
		// Add specific users
		for _, username := range req.Usernames {
			var userID int64
			err := h.db.QueryRow("SELECT id FROM users WHERE username = ?", username).Scan(&userID)
			if err != nil {
				continue // Skip invalid usernames
			}
			
			h.db.Exec(`
				INSERT IGNORE INTO script_access (script_id, access_type, user_id, granted_by, expires_at)
				VALUES (?, 'user', ?, ?, ?)
			`, scriptID, userID, claims.UserID, req.ExpiresAt)
		}
	}
	
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func (h *ShareHandler) RemoveAccess(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	
	scriptID := chi.URLParam(r, "id")
	accessID := chi.URLParam(r, "access_id")
	
	// Verify ownership
	var ownerID int64
	err := h.db.QueryRow("SELECT user_id FROM scripts WHERE id = ?", scriptID).Scan(&ownerID)
	if err != nil || ownerID != claims.UserID {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	
	_, err = h.db.Exec("DELETE FROM script_access WHERE id = ? AND script_id = ?", accessID, scriptID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.WriteHeader(http.StatusNoContent)
}

func (h *ShareHandler) SearchUsers(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" || len(query) < 2 {
		json.NewEncoder(w).Encode([]string{})
		return
	}
	
	rows, err := h.db.Query(`
		SELECT username FROM users 
		WHERE username LIKE ? 
		LIMIT 10
	`, "%"+query+"%")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	
	var usernames []string
	for rows.Next() {
		var username string
		if err := rows.Scan(&username); err != nil {
			continue
		}
		usernames = append(usernames, username)
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(usernames)
}
