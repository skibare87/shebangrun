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
	TierID       int64
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
	EncryptionType  string // 'none', 'server_managed', 'user_managed'
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

// Server-side encryption models

type UserEncryptionKey struct {
	ID            int64
	UserID        int64
	EncryptedUDEK []byte
	KeyVersion    int
	CreatedAt     time.Time
	RotatedAt     *time.Time
}

type Secret struct {
	ID             int64
	UserID         int64
	KeyName        string
	EncryptedValue []byte
	Version        int
	CreatedAt      time.Time
	UpdatedAt      time.Time
	LastAccessed   *time.Time
	ExpiresAt      *time.Time
}

type SecretAudit struct {
	ID          int64
	SecretID    int64
	UserID      int64
	Action      string // 'read', 'write', 'delete'
	IPAddress   string
	UserAgent   string
	AccessedAt  time.Time
}

type ScriptAccess struct {
	ID         int64
	ScriptID   int64
	AccessType string // 'link', 'user', 'group'
	UserID     *int64
	GroupID    *int64
	GrantedBy  int64
	GrantedAt  time.Time
	ExpiresAt  *time.Time
}

type UserGroup struct {
	ID          int64
	OwnerID     int64
	Name        string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type GroupMember struct {
	GroupID  int64
	UserID   int64
	AddedBy  int64
	AddedAt  time.Time
}

// Tier system models

type Tier struct {
	ID                 int64
	Name               string
	DisplayName        string
	PriceMonthly       float64
	MaxStorageBytes    int64
	MaxSecrets         int
	MaxScripts         int
	MaxAIGenerations   int
	RateLimit          int
	AllowPublic        bool
	AllowUnlisted      bool
	AllowPrivate       bool
	AllowAIGeneration  bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type AIGeneration struct {
	ID              int64
	UserID          int64
	Prompt          string
	Provider        string
	Model           string
	TokensUsed      int
	ScriptGenerated string
	CreatedAt       time.Time
}

type UsageStats struct {
	ID                  int64
	UserID              int64
	Month               time.Time
	StorageUsed         int64
	SecretsCount        int
	ScriptsCount        int
	AIGenerationsCount  int
}

type Subscription struct {
	ID                    int64
	UserID                int64
	TierID                int64
	Status                string
	StartedAt             time.Time
	ExpiresAt             *time.Time
	StripeSubscriptionID  string
}
