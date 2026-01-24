package database

import (
	"database/sql"
	"log"
)

// InitSchema creates all tables if they don't exist
func InitSchema(db *sql.DB) error {
	log.Println("Initializing database schema...")
	
	statements := []string{
		// Users table
		`CREATE TABLE IF NOT EXISTS users (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			username VARCHAR(50) UNIQUE NOT NULL,
			email VARCHAR(255) UNIQUE NOT NULL,
			password_hash VARCHAR(255),
			oauth_provider VARCHAR(50),
			oauth_id VARCHAR(255),
			is_admin BOOLEAN DEFAULT FALSE,
			tier_id BIGINT DEFAULT 1,
			rate_limit INT NULL DEFAULT NULL,
			subscription_managed_by ENUM('admin', 'stripe', 'manual') DEFAULT 'manual',
			subscription_expires_at TIMESTAMP NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_oauth (oauth_provider, oauth_id),
			INDEX idx_email (email)
		)`,
		
		// Username case-insensitive unique
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS username_lower VARCHAR(255) GENERATED ALWAYS AS (LOWER(username)) STORED`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_username_lower ON users(username_lower)`,
		
		// Tiers
		`CREATE TABLE IF NOT EXISTS tiers (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			name VARCHAR(50) UNIQUE NOT NULL,
			display_name VARCHAR(100) NOT NULL,
			price_monthly DECIMAL(10,2) NOT NULL,
			price_yearly DECIMAL(10,2) NULL,
			max_storage_bytes BIGINT NOT NULL,
			max_secrets INT NOT NULL,
			max_scripts INT NOT NULL,
			max_ai_generations INT NOT NULL,
			rate_limit INT NOT NULL,
			features JSON NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		)`,
		
		// Seed tiers if empty
		`INSERT IGNORE INTO tiers (id, name, display_name, price_monthly, price_yearly, max_storage_bytes, max_secrets, max_scripts, max_ai_generations, rate_limit, features) VALUES
		(1, 'free', 'Free', 0.00, 0.00, 10485760, 5, 25, 0, 120, '{"public": true, "unlisted": false, "private": false, "ai_generation": false, "api_access": true}'),
		(2, 'pro', 'Pro', 10.00, 96.00, 524288000, 50, 100, 5, 300, '{"public": true, "unlisted": true, "private": false, "ai_generation": false, "api_access": true, "priority_support": true}'),
		(3, 'ultimate', 'Ultimate', 20.00, 192.00, 1073741824, 500, 500, 500, 1000, '{"public": true, "unlisted": true, "private": true, "ai_generation": true, "api_access": true, "priority_support": true, "advanced_analytics": true}')`,
		
		// Add tier FK to users
		`ALTER TABLE users ADD FOREIGN KEY IF NOT EXISTS (tier_id) REFERENCES tiers(id)`,
		
		// Keypairs
		`CREATE TABLE IF NOT EXISTS keypairs (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			user_id BIGINT NOT NULL,
			name VARCHAR(100) NOT NULL,
			public_key TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
			INDEX idx_user_id (user_id)
		)`,
		
		// Scripts
		`CREATE TABLE IF NOT EXISTS scripts (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			user_id BIGINT NOT NULL,
			name VARCHAR(100) NOT NULL,
			description TEXT,
			visibility ENUM('private', 'unlisted', 'public') DEFAULT 'private',
			encryption_type ENUM('none', 'server_managed', 'user_managed') DEFAULT 'none',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
			UNIQUE KEY unique_user_script (user_id, name),
			INDEX idx_visibility (visibility)
		)`,
		
		// Script versions
		`CREATE TABLE IF NOT EXISTS script_versions (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			script_id BIGINT NOT NULL,
			version INT NOT NULL,
			content_hash VARCHAR(64) NOT NULL,
			signature TEXT,
			checksum VARCHAR(64) NOT NULL,
			size BIGINT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (script_id) REFERENCES scripts(id) ON DELETE CASCADE,
			UNIQUE KEY unique_script_version (script_id, version),
			INDEX idx_script_id (script_id)
		)`,
		
		// Script content
		`CREATE TABLE IF NOT EXISTS script_content (
			version_id BIGINT PRIMARY KEY,
			content LONGBLOB,
			encrypted_content LONGBLOB NULL,
			storage_path VARCHAR(255),
			encryption_key_id BIGINT,
			wrapped_key BLOB,
			FOREIGN KEY (version_id) REFERENCES script_versions(id) ON DELETE CASCADE,
			FOREIGN KEY (encryption_key_id) REFERENCES keypairs(id) ON DELETE SET NULL
		)`,
		
		// Tags
		`CREATE TABLE IF NOT EXISTS tags (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			script_id BIGINT NOT NULL,
			tag_name VARCHAR(50) NOT NULL,
			version_id BIGINT NOT NULL,
			FOREIGN KEY (script_id) REFERENCES scripts(id) ON DELETE CASCADE,
			FOREIGN KEY (version_id) REFERENCES script_versions(id) ON DELETE CASCADE,
			UNIQUE KEY unique_script_tag (script_id, tag_name)
		)`,
		
		// Share tokens
		`CREATE TABLE IF NOT EXISTS share_tokens (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			script_id BIGINT NOT NULL,
			token VARCHAR(255) UNIQUE NOT NULL,
			revoked BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (script_id) REFERENCES scripts(id) ON DELETE CASCADE,
			INDEX idx_token (token)
		)`,
		
		// User limits
		`CREATE TABLE IF NOT EXISTS user_limits (
			user_id BIGINT PRIMARY KEY,
			max_scripts INT,
			max_script_size BIGINT,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
		
		// API tokens
		`CREATE TABLE IF NOT EXISTS api_tokens (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			user_id BIGINT NOT NULL,
			name VARCHAR(255) NOT NULL,
			client_id VARCHAR(255) UNIQUE NOT NULL,
			client_secret VARCHAR(255) NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_used TIMESTAMP NULL,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
			INDEX idx_client_id (client_id),
			INDEX idx_user_id (user_id)
		)`,
		
		// User encryption keys
		`CREATE TABLE IF NOT EXISTS user_encryption_keys (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			user_id BIGINT NOT NULL,
			encrypted_udek BLOB NOT NULL,
			key_version INT DEFAULT 1,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			rotated_at TIMESTAMP NULL,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
			INDEX idx_user_id (user_id)
		)`,
		
		// Secrets
		`CREATE TABLE IF NOT EXISTS secrets (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			user_id BIGINT NOT NULL,
			key_name VARCHAR(255) NOT NULL,
			encrypted_value BLOB NOT NULL,
			version INT DEFAULT 1,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			last_accessed TIMESTAMP NULL,
			expires_at TIMESTAMP NULL,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
			UNIQUE KEY unique_user_key (user_id, key_name),
			INDEX idx_user_id (user_id),
			INDEX idx_expires_at (expires_at)
		)`,
		
		// Secrets audit
		`CREATE TABLE IF NOT EXISTS secrets_audit (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			secret_id BIGINT NOT NULL,
			user_id BIGINT NOT NULL,
			action ENUM('read', 'write', 'delete') NOT NULL,
			ip_address VARCHAR(45),
			user_agent TEXT,
			accessed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (secret_id) REFERENCES secrets(id) ON DELETE CASCADE,
			INDEX idx_secret_id (secret_id),
			INDEX idx_user_id (user_id),
			INDEX idx_accessed_at (accessed_at)
		)`,
		
		// Script access (ACL)
		`CREATE TABLE IF NOT EXISTS script_access (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			script_id BIGINT NOT NULL,
			access_type ENUM('link', 'user', 'group') NOT NULL,
			user_id BIGINT NULL,
			group_id BIGINT NULL,
			granted_by BIGINT NOT NULL,
			granted_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			expires_at TIMESTAMP NULL,
			FOREIGN KEY (script_id) REFERENCES scripts(id) ON DELETE CASCADE,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
			FOREIGN KEY (granted_by) REFERENCES users(id),
			INDEX idx_script_id (script_id),
			INDEX idx_user_id (user_id),
			INDEX idx_expires_at (expires_at)
		)`,
		
		// User groups
		`CREATE TABLE IF NOT EXISTS user_groups (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			owner_id BIGINT NOT NULL,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE CASCADE,
			INDEX idx_owner_id (owner_id)
		)`,
		
		// Group members
		`CREATE TABLE IF NOT EXISTS group_members (
			group_id BIGINT NOT NULL,
			user_id BIGINT NOT NULL,
			added_by BIGINT NOT NULL,
			added_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (group_id, user_id),
			FOREIGN KEY (group_id) REFERENCES user_groups(id) ON DELETE CASCADE,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
			FOREIGN KEY (added_by) REFERENCES users(id),
			INDEX idx_user_id (user_id)
		)`,
		
		// AI generations
		`CREATE TABLE IF NOT EXISTS ai_generations (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			user_id BIGINT NOT NULL,
			prompt TEXT NOT NULL,
			provider VARCHAR(50) NOT NULL,
			model VARCHAR(100) NOT NULL,
			tokens_used INT,
			script_generated TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
			INDEX idx_user_id (user_id),
			INDEX idx_created_at (created_at)
		)`,
		
		// Usage stats
		`CREATE TABLE IF NOT EXISTS usage_stats (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			user_id BIGINT NOT NULL,
			month DATE NOT NULL,
			storage_used BIGINT DEFAULT 0,
			secrets_count INT DEFAULT 0,
			scripts_count INT DEFAULT 0,
			ai_generations_count INT DEFAULT 0,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
			UNIQUE KEY unique_user_month (user_id, month),
			INDEX idx_user_month (user_id, month)
		)`,
		
		// Subscriptions
		`CREATE TABLE IF NOT EXISTS subscriptions (
			id BIGINT PRIMARY KEY AUTO_INCREMENT,
			user_id BIGINT NOT NULL,
			tier_id BIGINT NOT NULL,
			billing_cycle ENUM('monthly', 'yearly') DEFAULT 'monthly',
			status ENUM('active', 'cancelled', 'expired', 'trial') DEFAULT 'active',
			managed_by ENUM('admin', 'stripe', 'manual') DEFAULT 'manual',
			started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			current_period_end TIMESTAMP NULL,
			cancel_at_period_end BOOLEAN DEFAULT FALSE,
			stripe_subscription_id VARCHAR(255) NULL,
			stripe_customer_id VARCHAR(255) NULL,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
			FOREIGN KEY (tier_id) REFERENCES tiers(id),
			INDEX idx_user_id (user_id),
			INDEX idx_status (status),
			INDEX idx_current_period_end (current_period_end)
		)`,
	}
	
	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			log.Printf("Warning: %v", err)
			// Continue on error - table might already exist
		}
	}
	
	log.Println("Database schema initialized")
	return nil
}
