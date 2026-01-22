package kms

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"os"
)

// KeyManager interface for different key management backends
type KeyManager interface {
	GetMasterKey() ([]byte, error)
	Encrypt(plaintext []byte) ([]byte, error)
	Decrypt(ciphertext []byte) ([]byte, error)
}

// EnvKeyManager uses environment variable for master key (dev/testing)
type EnvKeyManager struct {
	key []byte
}

func NewEnvKeyManager(envVar string) (*EnvKeyManager, error) {
	keyStr := os.Getenv(envVar)
	if keyStr == "" {
		return nil, errors.New("master key not found in environment")
	}
	
	key, err := base64.StdEncoding.DecodeString(keyStr)
	if err != nil {
		return nil, err
	}
	
	if len(key) != 32 {
		return nil, errors.New("master key must be 32 bytes")
	}
	
	return &EnvKeyManager{key: key}, nil
}

func (m *EnvKeyManager) GetMasterKey() ([]byte, error) {
	return m.key, nil
}

func (m *EnvKeyManager) Encrypt(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(m.key)
	if err != nil {
		return nil, err
	}
	
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

func (m *EnvKeyManager) Decrypt(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(m.key)
	if err != nil {
		return nil, err
	}
	
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}
	
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}
	
	return plaintext, nil
}

// GenerateMasterKey generates a new 32-byte master key
func GenerateMasterKey() (string, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(key), nil
}
