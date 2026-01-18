package database

import "time"

type User struct {
	ID           int64
	Username     string
	Email        string
	PasswordHash string
	OAuthProvider string
	OAuthID      string
	IsAdmin      bool
	RateLimit    int
	CreatedAt    time.Time
}

type KeyPair struct {
	ID        int64
	UserID    int64
	Name      string
	PublicKey string
	CreatedAt time.Time
}

type Script struct {
	ID          int64
	UserID      int64
	Name        string
	Description string
	Visibility  string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type ScriptVersion struct {
	ID          int64
	ScriptID    int64
	Version     int
	ContentHash string
	Signature   string
	Checksum    string
	Size        int64
	CreatedAt   time.Time
}

type ScriptContent struct {
	VersionID       int64
	Content         []byte
	StoragePath     string
	EncryptionKeyID *int64
	WrappedKey      []byte // Encrypted symmetric key (encrypted with RSA public key)
}

type ShareToken struct {
	ID        int64
	ScriptID  int64
	Token     string
	Revoked   bool
	CreatedAt time.Time
}

type Tag struct {
	ID        int64
	ScriptID  int64
	TagName   string
	VersionID int64
}

type UserLimits struct {
	UserID         int64
	MaxScripts     *int
	MaxScriptSize  *int64
}
