package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"shebang.run/internal/crypto"
	"shebang.run/internal/database"
	"shebang.run/internal/middleware"

	"github.com/go-chi/chi/v5"
)

type KeyHandler struct {
	db *database.DB
}

func NewKeyHandler(db *database.DB) *KeyHandler {
	return &KeyHandler{db: db}
}

type GenerateKeyRequest struct {
	Name string `json:"name"`
}

type GenerateKeyResponse struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	PublicKey  string `json:"public_key"`
	PrivateKey string `json:"private_key"`
}

type ImportKeyRequest struct {
	Name      string `json:"name"`
	PublicKey string `json:"public_key"`
}

type KeyResponse struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	PublicKey string `json:"public_key"`
	CreatedAt string `json:"created_at"`
}

func (h *KeyHandler) List(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	keypairs, err := h.db.GetKeyPairsByUserID(claims.UserID)
	if err != nil {
		http.Error(w, "Failed to fetch keypairs", http.StatusInternalServerError)
		return
	}

	var response []KeyResponse
	for _, kp := range keypairs {
		response = append(response, KeyResponse{
			ID:        kp.ID,
			Name:      kp.Name,
			PublicKey: kp.PublicKey,
			CreatedAt: kp.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *KeyHandler) Generate(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req GenerateKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}

	privateKey, err := crypto.GenerateKeyPair()
	if err != nil {
		http.Error(w, "Failed to generate keypair", http.StatusInternalServerError)
		return
	}

	publicKeyPEM, err := crypto.EncodePublicKey(&privateKey.PublicKey)
	if err != nil {
		http.Error(w, "Failed to encode public key", http.StatusInternalServerError)
		return
	}

	kp, err := h.db.CreateKeyPair(claims.UserID, req.Name, publicKeyPEM)
	if err != nil {
		http.Error(w, "Failed to save keypair", http.StatusInternalServerError)
		return
	}

	privateKeyPEM := crypto.EncodePrivateKey(privateKey)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(GenerateKeyResponse{
		ID:         kp.ID,
		Name:       kp.Name,
		PublicKey:  publicKeyPEM,
		PrivateKey: privateKeyPEM,
	})
}

func (h *KeyHandler) Import(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req ImportKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Name == "" || req.PublicKey == "" {
		http.Error(w, "Name and public key are required", http.StatusBadRequest)
		return
	}

	if _, err := crypto.DecodePublicKey(req.PublicKey); err != nil {
		http.Error(w, "Invalid public key format", http.StatusBadRequest)
		return
	}

	kp, err := h.db.CreateKeyPair(claims.UserID, req.Name, req.PublicKey)
	if err != nil {
		http.Error(w, "Failed to save keypair", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(KeyResponse{
		ID:        kp.ID,
		Name:      kp.Name,
		PublicKey: kp.PublicKey,
		CreatedAt: kp.CreatedAt.Format("2006-01-02T15:04:05Z"),
	})
}

func (h *KeyHandler) Delete(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid keypair ID", http.StatusBadRequest)
		return
	}

	if err := h.db.DeleteKeyPair(id, claims.UserID); err != nil {
		http.Error(w, "Failed to delete keypair", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
