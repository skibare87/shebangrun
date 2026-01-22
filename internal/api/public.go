package api

import (
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"shebang.run/internal/auth"
	"shebang.run/internal/config"
	"shebang.run/internal/crypto"
	"shebang.run/internal/database"
	"shebang.run/internal/storage"

	"github.com/go-chi/chi/v5"
)

type PublicHandler struct {
	db      *database.DB
	storage storage.Storage
	cfg     *config.Config
}

func NewPublicHandler(db *database.DB, storage storage.Storage, cfg *config.Config) *PublicHandler {
	return &PublicHandler{db: db, storage: storage, cfg: cfg}
}

func (h *PublicHandler) GetScript(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")
	scriptSpec := chi.URLParam(r, "script")

	scriptName := scriptSpec
	tag := "latest"
	
	if strings.Contains(scriptSpec, "@") {
		parts := strings.SplitN(scriptSpec, "@", 2)
		scriptName = parts[0]
		tag = parts[1]
	}

	user, err := h.db.GetUserByUsername(username)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	script, err := h.db.GetScriptByUserAndName(user.ID, scriptName)
	if err != nil {
		http.Error(w, "Script not found", http.StatusNotFound)
		return
	}

	// Get current user ID if authenticated
	var currentUserID *int64
	if authHeader := r.Header.Get("Authorization"); authHeader != "" {
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if claims, err := auth.ValidateToken(tokenString, h.cfg.JWTSecret); err == nil {
			currentUserID = &claims.UserID
		}
	}

	// Check ACL for unlisted scripts
	if script.Visibility == "unlisted" {
		canAccess, err := h.db.CanAccessScript(script.ID, currentUserID)
		if err != nil || !canAccess {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}

	if script.Visibility == "private" {
		token := r.URL.Query().Get("token")
		if token == "" {
			// Check if script is encrypted - if so, return encrypted content
			version, err := h.db.GetLatestScriptVersion(script.ID)
			if err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			
			content, err := h.db.GetScriptContent(version.ID)
			if err != nil || content.EncryptionKeyID == nil {
				// Not encrypted, require token
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			// Script is encrypted, will return encrypted content below
		} else {
			// Validate share token
			shareToken, err := h.db.GetShareToken(token)
			if err != nil || shareToken.ScriptID != script.ID || shareToken.Revoked {
				http.Error(w, "Invalid or revoked token", http.StatusUnauthorized)
				return
			}
		}
	}

	var version *database.ScriptVersion
	if strings.HasPrefix(tag, "v") {
		versionNum, err := strconv.Atoi(tag[1:])
		if err != nil {
			http.Error(w, "Invalid version", http.StatusBadRequest)
			return
		}
		version, err = h.db.GetScriptVersionByNumber(script.ID, versionNum)
	} else {
		version, err = h.db.GetVersionByTag(script.ID, tag)
	}

	if err != nil {
		http.Error(w, "Version not found", http.StatusNotFound)
		return
	}

	content, err := h.db.GetScriptContent(version.ID)
	if err != nil {
		http.Error(w, "Content not found", http.StatusNotFound)
		return
	}

	var scriptData []byte
	if content.StoragePath != "" {
		reader, err := h.storage.Get(r.Context(), content.StoragePath)
		if err != nil {
			http.Error(w, "Failed to retrieve content", http.StatusInternalServerError)
			return
		}
		defer reader.Close()
		scriptData, err = io.ReadAll(reader)
		if err != nil {
			http.Error(w, "Failed to read content", http.StatusInternalServerError)
			return
		}
	} else {
		scriptData = content.Content
	}

	if content.EncryptionKeyID != nil {
		// Return encrypted content with metadata
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("X-Script-Version", strconv.Itoa(version.Version))
		w.Header().Set("X-Script-Checksum", version.Checksum)
		w.Header().Set("X-Encrypted", "true")
		w.Header().Set("X-Encryption-KeyID", strconv.FormatInt(*content.EncryptionKeyID, 10))
		
		// Include wrapped key in header (base64 encoded)
		if len(content.WrappedKey) > 0 {
			wrappedKeyB64 := hex.EncodeToString(content.WrappedKey)
			w.Header().Set("X-Wrapped-Key", wrappedKeyB64)
		}
		
		w.Write(scriptData)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("X-Script-Version", strconv.Itoa(version.Version))
	w.Header().Set("X-Script-Checksum", version.Checksum)
	w.Write(scriptData)
}

func (h *PublicHandler) GetMetadata(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")
	scriptName := chi.URLParam(r, "script")

	user, err := h.db.GetUserByUsername(username)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	script, err := h.db.GetScriptByUserAndName(user.ID, scriptName)
	if err != nil {
		http.Error(w, "Script not found", http.StatusNotFound)
		return
	}

	if script.Visibility != "public" {
		http.Error(w, "Not available", http.StatusForbidden)
		return
	}

	version, err := h.db.GetLatestScriptVersion(script.ID)
	if err != nil {
		http.Error(w, "No versions found", http.StatusNotFound)
		return
	}

	metadata := map[string]interface{}{
		"name":        script.Name,
		"description": script.Description,
		"visibility":  script.Visibility,
		"version":     version.Version,
		"checksum":    version.Checksum,
		"size":        version.Size,
		"created_at":  script.CreatedAt.Format("2006-01-02T15:04:05Z"),
		"updated_at":  script.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metadata)
}

func (h *PublicHandler) VerifySignature(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")
	scriptName := chi.URLParam(r, "script")

	user, err := h.db.GetUserByUsername(username)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	script, err := h.db.GetScriptByUserAndName(user.ID, scriptName)
	if err != nil {
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

	var scriptData []byte
	if content.StoragePath != "" {
		reader, err := h.storage.Get(r.Context(), content.StoragePath)
		if err != nil {
			http.Error(w, "Failed to retrieve content", http.StatusInternalServerError)
			return
		}
		defer reader.Close()
		scriptData, err = io.ReadAll(reader)
		if err != nil {
			http.Error(w, "Failed to read content", http.StatusInternalServerError)
			return
		}
	} else {
		scriptData = content.Content
	}

	result := map[string]interface{}{
		"checksum": version.Checksum,
		"signed":   version.Signature != "",
	}

	if version.Signature != "" && content.EncryptionKeyID != nil {
		kp, err := h.db.GetKeyPairByID(*content.EncryptionKeyID)
		if err == nil {
			pubKey, err := crypto.DecodePublicKey(kp.PublicKey)
			if err == nil {
				sig, _ := hex.DecodeString(version.Signature)
				err = crypto.VerifySignature(scriptData, sig, pubKey)
				result["verified"] = err == nil
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
