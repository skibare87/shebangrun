package api

import (
	"encoding/json"
	"net/http"

	"shebang.run/internal/database"
)

type CommunityHandler struct {
	db *database.DB
}

func NewCommunityHandler(db *database.DB) *CommunityHandler {
	return &CommunityHandler{db: db}
}

type CommunityScriptResponse struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Username    string `json:"username"`
	Description string `json:"description"`
	Version     int    `json:"version"`
	UpdatedAt   string `json:"updated_at"`
}

func (h *CommunityHandler) ListPublicScripts(w http.ResponseWriter, r *http.Request) {
	// Get all public scripts from all users
	rows, err := h.db.Query(`
		SELECT s.id, s.name, s.description, s.updated_at, u.username,
		       COALESCE((SELECT MAX(version) FROM script_versions WHERE script_id = s.id), 0) as version
		FROM scripts s
		JOIN users u ON s.user_id = u.id
		WHERE s.visibility = 'public'
		ORDER BY s.updated_at DESC
		LIMIT 100
	`)
	if err != nil {
		http.Error(w, "Failed to fetch scripts", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var scripts []CommunityScriptResponse
	for rows.Next() {
		var s CommunityScriptResponse
		if err := rows.Scan(&s.ID, &s.Name, &s.Description, &s.UpdatedAt, &s.Username, &s.Version); err != nil {
			continue
		}
		scripts = append(scripts, s)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(scripts)
}
