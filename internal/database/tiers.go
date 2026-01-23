package database

// GetUserTier gets the tier for a user
func (db *DB) GetUserTier(userID int64) (*Tier, error) {
	tier := &Tier{}
	err := db.QueryRow(`
		SELECT t.id, t.name, t.display_name, t.price_monthly, t.max_storage_bytes, 
		       t.max_secrets, t.max_scripts, t.max_ai_generations, t.rate_limit,
		       t.allow_public, t.allow_unlisted, t.allow_private, t.allow_ai_generation,
		       t.created_at, t.updated_at
		FROM tiers t
		JOIN users u ON u.tier_id = t.id
		WHERE u.id = ?
	`, userID).Scan(
		&tier.ID, &tier.Name, &tier.DisplayName, &tier.PriceMonthly,
		&tier.MaxStorageBytes, &tier.MaxSecrets, &tier.MaxScripts, &tier.MaxAIGenerations,
		&tier.RateLimit, &tier.AllowPublic, &tier.AllowUnlisted, &tier.AllowPrivate,
		&tier.AllowAIGeneration, &tier.CreatedAt, &tier.UpdatedAt,
	)
	return tier, err
}

// GetAllTiers lists all available tiers
func (db *DB) GetAllTiers() ([]*Tier, error) {
	rows, err := db.Query(`
		SELECT id, name, display_name, price_monthly, max_storage_bytes, 
		       max_secrets, max_scripts, max_ai_generations, rate_limit,
		       allow_public, allow_unlisted, allow_private, allow_ai_generation,
		       created_at, updated_at
		FROM tiers
		ORDER BY price_monthly ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var tiers []*Tier
	for rows.Next() {
		tier := &Tier{}
		if err := rows.Scan(
			&tier.ID, &tier.Name, &tier.DisplayName, &tier.PriceMonthly,
			&tier.MaxStorageBytes, &tier.MaxSecrets, &tier.MaxScripts, &tier.MaxAIGenerations,
			&tier.RateLimit, &tier.AllowPublic, &tier.AllowUnlisted, &tier.AllowPrivate,
			&tier.AllowAIGeneration, &tier.CreatedAt, &tier.UpdatedAt,
		); err != nil {
			return nil, err
		}
		tiers = append(tiers, tier)
	}
	return tiers, rows.Err()
}

// UpdateUserTier changes a user's tier
func (db *DB) UpdateUserTier(userID, tierID int64) error {
	_, err := db.Exec("UPDATE users SET tier_id = ? WHERE id = ?", tierID, userID)
	return err
}

// GetMonthlyUsage gets or creates usage stats for current month
func (db *DB) GetMonthlyUsage(userID int64) (*UsageStats, error) {
	stats := &UsageStats{}
	
	// Get current month (first day)
	err := db.QueryRow(`
		SELECT id, user_id, month, storage_used, secrets_count, scripts_count, ai_generations_count
		FROM usage_stats
		WHERE user_id = ? AND month = DATE_FORMAT(NOW(), '%Y-%m-01')
	`, userID).Scan(
		&stats.ID, &stats.UserID, &stats.Month, &stats.StorageUsed,
		&stats.SecretsCount, &stats.ScriptsCount, &stats.AIGenerationsCount,
	)
	
	if err != nil {
		// Create if doesn't exist
		result, err := db.Exec(`
			INSERT INTO usage_stats (user_id, month)
			VALUES (?, DATE_FORMAT(NOW(), '%Y-%m-01'))
		`, userID)
		if err != nil {
			return nil, err
		}
		id, _ := result.LastInsertId()
		stats.ID = id
		stats.UserID = userID
	}
	
	return stats, nil
}

// IncrementAIGenerations increments the AI generation count
func (db *DB) IncrementAIGenerations(userID int64) error {
	_, err := db.Exec(`
		INSERT INTO usage_stats (user_id, month, ai_generations_count)
		VALUES (?, DATE_FORMAT(NOW(), '%Y-%m-01'), 1)
		ON DUPLICATE KEY UPDATE ai_generations_count = ai_generations_count + 1
	`, userID)
	return err
}

// RecordAIGeneration logs an AI generation
func (db *DB) RecordAIGeneration(userID int64, prompt, provider, model, script string, tokens int) error {
	_, err := db.Exec(`
		INSERT INTO ai_generations (user_id, prompt, provider, model, tokens_used, script_generated)
		VALUES (?, ?, ?, ?, ?, ?)
	`, userID, prompt, provider, model, tokens, script)
	return err
}

// CanGenerateAI checks if user can generate AI scripts
func (db *DB) CanGenerateAI(userID int64, isAdmin bool) (bool, int, error) {
	// Admins are unlimited
	if isAdmin {
		return true, -1, nil
	}
	
	tier, err := db.GetUserTier(userID)
	if err != nil {
		return false, 0, err
	}
	
	if !tier.AllowAIGeneration {
		return false, 0, nil
	}
	
	usage, err := db.GetMonthlyUsage(userID)
	if err != nil {
		return false, 0, err
	}
	
	remaining := tier.MaxAIGenerations - usage.AIGenerationsCount
	return remaining > 0, remaining, nil
}
