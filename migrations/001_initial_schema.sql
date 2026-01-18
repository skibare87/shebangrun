-- Migration: 001_initial_schema.sql

CREATE TABLE IF NOT EXISTS users (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255),
    oauth_provider VARCHAR(50),
    oauth_id VARCHAR(255),
    is_admin BOOLEAN DEFAULT FALSE,
    rate_limit INT DEFAULT 50,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_username (username),
    INDEX idx_email (email),
    UNIQUE KEY unique_oauth (oauth_provider, oauth_id)
);

CREATE TABLE IF NOT EXISTS keypairs (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    name VARCHAR(100) NOT NULL,
    public_key TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE KEY unique_user_keyname (user_id, name)
);

CREATE TABLE IF NOT EXISTS scripts (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    visibility ENUM('private', 'unlisted', 'public') DEFAULT 'private',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE KEY unique_user_script (user_id, name),
    INDEX idx_visibility (visibility)
);

CREATE TABLE IF NOT EXISTS script_versions (
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
);

CREATE TABLE IF NOT EXISTS script_content (
    version_id BIGINT PRIMARY KEY,
    content MEDIUMBLOB,
    storage_path VARCHAR(500),
    encryption_key_id BIGINT,
    FOREIGN KEY (version_id) REFERENCES script_versions(id) ON DELETE CASCADE,
    FOREIGN KEY (encryption_key_id) REFERENCES keypairs(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS share_tokens (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    script_id BIGINT NOT NULL,
    token VARCHAR(64) UNIQUE NOT NULL,
    revoked BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (script_id) REFERENCES scripts(id) ON DELETE CASCADE,
    INDEX idx_token (token)
);

CREATE TABLE IF NOT EXISTS tags (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    script_id BIGINT NOT NULL,
    tag_name VARCHAR(50) NOT NULL,
    version_id BIGINT NOT NULL,
    FOREIGN KEY (script_id) REFERENCES scripts(id) ON DELETE CASCADE,
    FOREIGN KEY (version_id) REFERENCES script_versions(id) ON DELETE CASCADE,
    UNIQUE KEY unique_script_tag (script_id, tag_name),
    INDEX idx_tag_name (tag_name)
);

CREATE TABLE IF NOT EXISTS user_limits (
    user_id BIGINT PRIMARY KEY,
    max_scripts INT,
    max_script_size BIGINT,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
