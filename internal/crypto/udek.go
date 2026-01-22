package crypto

import (
	"crypto/rand"
	"database/sql"
	"errors"
	
	"shebang.run/internal/kms"
)

// UDEKManager handles User Data Encryption Keys
type UDEKManager struct {
	db  *sql.DB
	kms kms.KeyManager
}

func NewUDEKManager(db *sql.DB, km kms.KeyManager) *UDEKManager {
	return &UDEKManager{
		db:  db,
		kms: km,
	}
}

// GetOrCreateUDEK retrieves or creates a UDEK for a user
func (m *UDEKManager) GetOrCreateUDEK(userID int64) ([]byte, error) {
	// Try to get existing UDEK
	var encryptedUDEK []byte
	err := m.db.QueryRow(`
		SELECT encrypted_udek FROM user_encryption_keys 
		WHERE user_id = ? ORDER BY id DESC LIMIT 1
	`, userID).Scan(&encryptedUDEK)
	
	if err == sql.ErrNoRows {
		// Create new UDEK
		return m.createUDEK(userID)
	} else if err != nil {
		return nil, err
	}
	
	// Decrypt UDEK with master key
	udek, err := m.kms.Decrypt(encryptedUDEK)
	if err != nil {
		return nil, err
	}
	
	return udek, nil
}

// createUDEK generates and stores a new UDEK
func (m *UDEKManager) createUDEK(userID int64) ([]byte, error) {
	// Generate 32-byte UDEK
	udek := make([]byte, 32)
	if _, err := rand.Read(udek); err != nil {
		return nil, err
	}
	
	// Encrypt with master key
	encryptedUDEK, err := m.kms.Encrypt(udek)
	if err != nil {
		return nil, err
	}
	
	// Store in database
	_, err = m.db.Exec(`
		INSERT INTO user_encryption_keys (user_id, encrypted_udek, key_version)
		VALUES (?, ?, 1)
	`, userID, encryptedUDEK)
	if err != nil {
		return nil, err
	}
	
	return udek, nil
}

// EncryptWithUDEK encrypts data with a UDEK
func EncryptWithUDEK(plaintext []byte, udek []byte) ([]byte, error) {
	if len(udek) != 32 {
		return nil, errors.New("UDEK must be 32 bytes")
	}
	
	// Use ChaCha20-Poly1305 for content encryption
	return EncryptData(plaintext, udek)
}

// DecryptWithUDEK decrypts data with a UDEK
func DecryptWithUDEK(ciphertext []byte, udek []byte) ([]byte, error) {
	if len(udek) != 32 {
		return nil, errors.New("UDEK must be 32 bytes")
	}
	
	return DecryptData(ciphertext, udek)
}
