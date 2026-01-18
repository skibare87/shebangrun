package api

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"

	"shebang.run/internal/config"
	"shebang.run/internal/crypto"
	"shebang.run/internal/database"
	"shebang.run/internal/middleware"
	"shebang.run/internal/storage"
	"shebang.run/internal/auth"

	"github.com/go-chi/chi/v5"
)

type ScriptHandler struct {
	db      *database.DB
	storage storage.Storage
	cfg     *config.Config
}

func NewScriptHandler(db *database.DB, storage storage.Storage, cfg *config.Config) *ScriptHandler {
	return &ScriptHandler{db: db, storage: storage, cfg: cfg}
}

type CreateScriptRequest struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Visibility  string      `json:"visibility"`
	Content     string      `json:"content"`
	KeyPairID   interface{} `json:"keypair_id"` // Can be null, int, or string
}

type UpdateScriptRequest struct {
	Description string      `json:"description"`
	Visibility  string      `json:"visibility"`
	Content     string      `json:"content"`
	KeyPairID   interface{} `json:"keypair_id"` // Can be null, int, or string
	Tag         string      `json:"tag"`
}

type ScriptResponse struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Visibility  string `json:"visibility"`
	Version     int    `json:"version"`
	Encrypted   bool   `json:"encrypted"`
	KeyPairID   *int64 `json:"keypair_id"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

func (h *ScriptHandler) List(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	scripts, err := h.db.GetScriptsByUserID(claims.UserID)
	if err != nil {
		http.Error(w, "Failed to fetch scripts", http.StatusInternalServerError)
		return
	}

	var response []ScriptResponse
	for _, s := range scripts {
		version, _ := h.db.GetLatestScriptVersion(s.ID)
		versionNum := 0
		var encrypted bool
		var keyPairID *int64
		
		if version != nil {
			versionNum = version.Version
			content, _ := h.db.GetScriptContent(version.ID)
			if content != nil && content.EncryptionKeyID != nil {
				encrypted = true
				keyPairID = content.EncryptionKeyID
			}
		}
		
		response = append(response, ScriptResponse{
			ID:          s.ID,
			Name:        s.Name,
			Description: s.Description,
			Visibility:  s.Visibility,
			Version:     versionNum,
			Encrypted:   encrypted,
			KeyPairID:   keyPairID,
			CreatedAt:   s.CreatedAt.Format("2006-01-02T15:04:05Z"),
			UpdatedAt:   s.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *ScriptHandler) Get(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid script ID", http.StatusBadRequest)
		return
	}

	script, err := h.db.GetScriptByID(id)
	if err != nil || script.UserID != claims.UserID {
		http.Error(w, "Script not found", http.StatusNotFound)
		return
	}

	version, _ := h.db.GetLatestScriptVersion(script.ID)
	versionNum := 0
	var encrypted bool
	var keyPairID *int64
	
	if version != nil {
		versionNum = version.Version
		content, _ := h.db.GetScriptContent(version.ID)
		if content != nil && content.EncryptionKeyID != nil {
			encrypted = true
			keyPairID = content.EncryptionKeyID
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ScriptResponse{
		ID:          script.ID,
		Name:        script.Name,
		Description: script.Description,
		Visibility:  script.Visibility,
		Version:     versionNum,
		Encrypted:   encrypted,
		KeyPairID:   keyPairID,
		CreatedAt:   script.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:   script.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	})
}

func (h *ScriptHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req CreateScriptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if req.Name == "" || req.Content == "" {
		http.Error(w, "Name and content are required", http.StatusBadRequest)
		return
	}

	if req.Visibility == "" {
		req.Visibility = "private"
	}

	count, err := h.db.GetScriptCount(claims.UserID)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	maxScripts, maxSize, err := h.db.GetUserLimits(claims.UserID, h.cfg.DefaultMaxScripts, h.cfg.DefaultMaxScriptSize)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	if count >= maxScripts {
		http.Error(w, fmt.Sprintf("Script limit reached (%d)", maxScripts), http.StatusForbidden)
		return
	}

	if int64(len(req.Content)) > maxSize {
		http.Error(w, fmt.Sprintf("Script too large (max %d bytes)", maxSize), http.StatusRequestEntityTooLarge)
		return
	}

	script, err := h.db.CreateScript(claims.UserID, req.Name, req.Description, req.Visibility)
	if err != nil {
		http.Error(w, "Failed to create script", http.StatusInternalServerError)
		return
	}

	// Convert keypair_id from interface{} to *int64
	var keyPairID *int64
	if req.KeyPairID != nil {
		switch v := req.KeyPairID.(type) {
		case float64:
			id := int64(v)
			keyPairID = &id
		case string:
			if v != "" && v != "null" {
				if id, err := strconv.ParseInt(v, 10, 64); err == nil {
					keyPairID = &id
				}
			}
		case int:
			id := int64(v)
			keyPairID = &id
		case int64:
			keyPairID = &v
		}
	}

	if err := h.createVersion(script, []byte(req.Content), keyPairID, claims.UserID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(ScriptResponse{
		ID:          script.ID,
		Name:        script.Name,
		Description: script.Description,
		Visibility:  script.Visibility,
		Version:     1,
		CreatedAt:   script.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:   script.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	})
}

func (h *ScriptHandler) createVersion(script *database.Script, content []byte, keyPairID *int64, userID int64) error {
	latestVersion, _ := h.db.GetLatestScriptVersion(script.ID)
	newVersion := 1
	if latestVersion != nil {
		newVersion = latestVersion.Version + 1
	}

	hash := sha256.Sum256(content)
	contentHash := hex.EncodeToString(hash[:])
	checksum := hex.EncodeToString(hash[:])

	var signature string
	var storedContent []byte
	var encKeyID *int64
	var wrappedKey []byte

	if script.Visibility == "private" && keyPairID != nil {
		kp, err := h.db.GetKeyPairByID(*keyPairID)
		if err != nil || kp.UserID != userID {
			return fmt.Errorf("invalid keypair")
		}

		pubKey, err := crypto.DecodePublicKey(kp.PublicKey)
		if err != nil {
			return fmt.Errorf("invalid public key")
		}

		// Generate symmetric encryption key
		encKey, err := crypto.GenerateEncryptionKey()
		if err != nil {
			return err
		}

		// Encrypt content with symmetric key
		encrypted, err := crypto.EncryptData(content, encKey)
		if err != nil {
			return err
		}

		// Wrap (encrypt) the symmetric key with RSA public key
		wrappedKey, err = crypto.WrapKey(encKey, pubKey)
		if err != nil {
			return fmt.Errorf("failed to wrap encryption key: %v", err)
		}

		storedContent = encrypted
		encKeyID = keyPairID

		// Note: Signing requires private key which we don't store
		// Signature should be generated client-side if needed
	} else {
		storedContent = content
	}

	version, err := h.db.CreateScriptVersion(script.ID, newVersion, contentHash, signature, checksum, int64(len(content)))
	if err != nil {
		return err
	}

	storagePath := fmt.Sprintf("%d/%d/%d", userID, script.ID, version.ID)
	ctx := context.Background()
	if err := h.storage.Put(ctx, storagePath, bytes.NewReader(storedContent), int64(len(storedContent))); err != nil {
		return err
	}

	if err := h.db.SaveScriptContent(version.ID, nil, storagePath, encKeyID, wrappedKey); err != nil {
		return err
	}

	if err := h.db.CreateTag(script.ID, "latest", version.ID); err != nil {
		return err
	}

	return nil
}

func (h *ScriptHandler) Update(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid script ID", http.StatusBadRequest)
		return
	}

	script, err := h.db.GetScriptByID(id)
	if err != nil || script.UserID != claims.UserID {
		http.Error(w, "Script not found", http.StatusNotFound)
		return
	}

	var req UpdateScriptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Failed to decode update request: %v", err)
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	// Update metadata if provided
	needsMetadataUpdate := false
	desc := script.Description
	vis := script.Visibility
	
	if req.Description != "" && req.Description != script.Description {
		desc = req.Description
		needsMetadataUpdate = true
	}
	if req.Visibility != "" && req.Visibility != script.Visibility {
		vis = req.Visibility
		needsMetadataUpdate = true
	}
	
	if needsMetadataUpdate {
		if err := h.db.UpdateScript(id, desc, vis); err != nil {
			http.Error(w, "Failed to update script", http.StatusInternalServerError)
			return
		}
		script.Description = desc
		script.Visibility = vis
	}

	if req.Content != "" {
		// Convert keypair_id from interface{} to *int64
		var keyPairID *int64
		if req.KeyPairID != nil {
			switch v := req.KeyPairID.(type) {
			case float64:
				id := int64(v)
				keyPairID = &id
			case string:
				if v != "" && v != "null" {
					if id, err := strconv.ParseInt(v, 10, 64); err == nil {
						keyPairID = &id
					}
				}
			case int:
				id := int64(v)
				keyPairID = &id
			case int64:
				keyPairID = &v
			}
		}
		
		if err := h.createVersion(script, []byte(req.Content), keyPairID, claims.UserID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if req.Tag != "" && (req.Tag == "dev" || req.Tag == "beta") {
			latestVersion, _ := h.db.GetLatestScriptVersion(script.ID)
			if latestVersion != nil {
				h.db.CreateTag(script.ID, req.Tag, latestVersion.ID)
			}
		}
	}

	w.WriteHeader(http.StatusOK)
}

func (h *ScriptHandler) Delete(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid script ID", http.StatusBadRequest)
		return
	}

	if err := h.db.DeleteScript(id, claims.UserID); err != nil {
		http.Error(w, "Failed to delete script", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *ScriptHandler) GetEncryptedContent(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid script ID", http.StatusBadRequest)
		return
	}

	script, err := h.db.GetScriptByID(id)
	if err != nil || script.UserID != claims.UserID {
		http.Error(w, "Script not found", http.StatusNotFound)
		return
	}

	version, err := h.db.GetLatestScriptVersion(script.ID)
	if err != nil {
		http.Error(w, "No versions found", http.StatusNotFound)
		return
	}

	content, err := h.db.GetScriptContent(version.ID)
	if err != nil {
		http.Error(w, "Content not found", http.StatusNotFound)
		return
	}

	if content.EncryptionKeyID == nil {
		http.Error(w, "Script is not encrypted", http.StatusBadRequest)
		return
	}

	// Get encrypted content from storage
	var encryptedData []byte
	if content.StoragePath != "" {
		reader, err := h.storage.Get(r.Context(), content.StoragePath)
		if err != nil {
			http.Error(w, "Failed to retrieve content", http.StatusInternalServerError)
			return
		}
		defer reader.Close()
		encryptedData, err = io.ReadAll(reader)
		if err != nil {
			http.Error(w, "Failed to read content", http.StatusInternalServerError)
			return
		}
	} else {
		encryptedData = content.Content
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"encrypted_content": encryptedData,
		"wrapped_key":       content.WrappedKey,
		"keypair_id":        *content.EncryptionKeyID,
	})
}

func (h *ScriptHandler) GenerateShareToken(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid script ID", http.StatusBadRequest)
		return
	}

	script, err := h.db.GetScriptByID(id)
	if err != nil || script.UserID != claims.UserID {
		http.Error(w, "Script not found", http.StatusNotFound)
		return
	}

	token, err := auth.GenerateRandomToken(32)
	if err != nil {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	if err := h.db.CreateShareToken(id, token); err != nil {
		http.Error(w, "Failed to create share token", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": token})
}

func (h *ScriptHandler) RevokeShareToken(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.GetUserFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	token := chi.URLParam(r, "token")
	if err := h.db.RevokeShareToken(token, claims.UserID); err != nil {
		http.Error(w, "Failed to revoke token", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
