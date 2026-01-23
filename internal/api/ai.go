package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	
	"shebang.run/internal/ai"
	"shebang.run/internal/middleware"
)

type AIHandler struct {
	db        *sql.DB
	providers map[string]ai.AIProvider
}

func NewAIHandler(db *sql.DB, providers map[string]ai.AIProvider) *AIHandler {
	return &AIHandler{
		db:        db,
		providers: providers,
	}
}

func (h *AIHandler) Generate(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	
	// Check if user can use AI (admins bypass)
	if !claims.IsAdmin {
		tier, ok := middleware.GetTierFromContext(r.Context())
		if !ok || !tier.Features["ai_generation"] {
			http.Error(w, "AI generation not available in your tier. Upgrade to Ultimate.", http.StatusForbidden)
			return
		}
		
		// Check monthly limit
		var count int
		h.db.QueryRow(`
			SELECT COALESCE(ai_generations_count, 0) 
			FROM usage_stats 
			WHERE user_id = ? AND month = DATE_FORMAT(NOW(), '%Y-%m-01')
		`, claims.UserID).Scan(&count)
		
		if count >= tier.MaxAIGenerations {
			http.Error(w, fmt.Sprintf("AI generation limit reached (%d/%d this month)", count, tier.MaxAIGenerations), http.StatusForbidden)
			return
		}
	}
	
	var req ai.GenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	
	if req.Prompt == "" {
		http.Error(w, "Prompt required", http.StatusBadRequest)
		return
	}
	
	// Select provider
	providerName := req.Provider
	if providerName == "" {
		providerName = "claude" // Default
	}
	
	provider, ok := h.providers[providerName]
	if !ok {
		http.Error(w, "Provider not available", http.StatusBadRequest)
		return
	}
	
	// Generate script
	script, tokens, err := provider.GenerateScript(r.Context(), req.Prompt, req.Args)
	if err != nil {
		log.Printf("AI generation error: %v", err)
		http.Error(w, "Failed to generate script: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	// Record generation
	h.db.Exec(`
		INSERT INTO ai_generations (user_id, prompt, provider, model, tokens_used, script_generated)
		VALUES (?, ?, ?, ?, ?, ?)
	`, claims.UserID, req.Prompt, providerName, provider.Name(), tokens, script)
	
	// Increment usage counter
	h.db.Exec(`
		INSERT INTO usage_stats (user_id, month, ai_generations_count)
		VALUES (?, DATE_FORMAT(NOW(), '%Y-%m-01'), 1)
		ON DUPLICATE KEY UPDATE ai_generations_count = ai_generations_count + 1
	`, claims.UserID)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ai.GenerateResponse{
		Script:   script,
		Provider: providerName,
		Model:    provider.Name(),
		Tokens:   tokens,
	})
}

func (h *AIHandler) GetUsage(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	
	var count int
	h.db.QueryRow(`
		SELECT COALESCE(ai_generations_count, 0) 
		FROM usage_stats 
		WHERE user_id = ? AND month = DATE_FORMAT(NOW(), '%Y-%m-01')
	`, claims.UserID).Scan(&count)
	
	tier, _ := middleware.GetTierFromContext(r.Context())
	limit := 0
	if tier != nil {
		limit = tier.MaxAIGenerations
	}
	
	if claims.IsAdmin {
		limit = -1 // Unlimited
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"used":      count,
		"limit":     limit,
		"unlimited": claims.IsAdmin,
	})
}
