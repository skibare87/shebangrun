package api

import (
	"encoding/json"
	"net/http"

	"shebang.run/internal/database"
)

type SetupHandler struct {
	db *database.DB
}

func NewSetupHandler(db *database.DB) *SetupHandler {
	return &SetupHandler{db: db}
}

func (h *SetupHandler) Status(w http.ResponseWriter, r *http.Request) {
	isFirst, err := h.db.IsFirstUser()
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"setup_complete": !isFirst,
		"needs_setup":    isFirst,
	})
}
