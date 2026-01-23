-- Migration 005: User tiers and AI generation limits

-- Tiers table with flexible features
CREATE TABLE tiers (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(50) UNIQUE NOT NULL,
    display_name VARCHAR(100) NOT NULL,
    price_monthly DECIMAL(10,2) NOT NULL,
    max_storage_bytes BIGINT NOT NULL,
    max_secrets INT NOT NULL,
    max_scripts INT NOT NULL,
    max_ai_generations INT NOT NULL,
    rate_limit INT NOT NULL,
    features JSON NOT NULL,  -- Flexible feature flags
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- Seed default tiers
INSERT INTO tiers (name, display_name, price_monthly, max_storage_bytes, max_secrets, max_scripts, max_ai_generations, rate_limit, features) VALUES
('free', 'Free', 0.00, 10485760, 5, 25, 0, 50, 
 '{"public": true, "unlisted": false, "private": false, "ai_generation": false, "api_access": true}'),
('pro', 'Pro', 10.00, 524288000, 50, 100, 5, 200, 
 '{"public": true, "unlisted": true, "private": false, "ai_generation": false, "api_access": true, "priority_support": true}'),
('ultimate', 'Ultimate', 20.00, 1073741824, 500, 500, 500, 1000, 
 '{"public": true, "unlisted": true, "private": true, "ai_generation": true, "api_access": true, "priority_support": true, "advanced_analytics": true}');

-- Add tier to users (default to free tier)
ALTER TABLE users ADD COLUMN tier_id BIGINT DEFAULT 1;
ALTER TABLE users ADD FOREIGN KEY (tier_id) REFERENCES tiers(id);

-- AI generation tracking
CREATE TABLE ai_generations (
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
);

-- Track monthly usage
CREATE TABLE usage_stats (
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
);

-- Subscriptions (for future billing integration)
CREATE TABLE subscriptions (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id BIGINT NOT NULL,
    tier_id BIGINT NOT NULL,
    status ENUM('active', 'cancelled', 'expired', 'trial') DEFAULT 'active',
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NULL,
    stripe_subscription_id VARCHAR(255) NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (tier_id) REFERENCES tiers(id),
    INDEX idx_user_id (user_id),
    INDEX idx_status (status)
);

-- Set existing users to free tier
UPDATE users SET tier_id = 1 WHERE tier_id IS NULL;

-- Admins are unlimited (set to ultimate tier but will be checked separately)
UPDATE users SET tier_id = 3 WHERE is_admin = TRUE;
