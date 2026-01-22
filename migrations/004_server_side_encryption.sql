-- Migration 004: Server-side encryption and secrets management
-- Phase 1: Foundation

-- User encryption keys (UDEK)
CREATE TABLE user_encryption_keys (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id BIGINT NOT NULL,
    encrypted_udek TEXT NOT NULL,  -- Encrypted with master key
    key_version INT DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    rotated_at TIMESTAMP NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    INDEX idx_user_id (user_id)
);

-- Update scripts table for server-side encryption
ALTER TABLE scripts 
    ADD COLUMN encryption_type ENUM('none', 'server_managed', 'user_managed') DEFAULT 'none' AFTER visibility;

-- Update script_content table
ALTER TABLE script_content
    ADD COLUMN encrypted_content LONGBLOB NULL AFTER content;

-- Secrets store
CREATE TABLE secrets (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id BIGINT NOT NULL,
    key_name VARCHAR(255) NOT NULL,
    encrypted_value TEXT NOT NULL,
    version INT DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    last_accessed TIMESTAMP NULL,
    expires_at TIMESTAMP NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE KEY unique_user_key (user_id, key_name),
    INDEX idx_user_id (user_id),
    INDEX idx_expires_at (expires_at)
);

-- Secrets audit log
CREATE TABLE secrets_audit (
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
);

-- Script access control (ACL)
CREATE TABLE script_access (
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
);

-- User groups (for future group-based sharing)
CREATE TABLE user_groups (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    owner_id BIGINT NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE CASCADE,
    INDEX idx_owner_id (owner_id)
);

CREATE TABLE group_members (
    group_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    added_by BIGINT NOT NULL,
    added_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (group_id, user_id),
    FOREIGN KEY (group_id) REFERENCES user_groups(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (added_by) REFERENCES users(id),
    INDEX idx_user_id (user_id)
);

-- Enforce unique usernames (case-insensitive)
ALTER TABLE users ADD COLUMN username_lower VARCHAR(255) GENERATED ALWAYS AS (LOWER(username)) STORED;
CREATE UNIQUE INDEX idx_username_lower ON users(username_lower);

-- Migrate existing data
-- Set encryption_type for existing scripts
UPDATE scripts SET encryption_type = 'user_managed' WHERE keypair_id IS NOT NULL;
UPDATE scripts SET encryption_type = 'none' WHERE keypair_id IS NULL;
