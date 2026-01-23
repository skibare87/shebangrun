package database

import (
	"database/sql"
	"errors"
)

func (db *DB) CreateShareToken(scriptID int64, token string) error {
	_, err := db.Exec(
		"INSERT INTO share_tokens (script_id, token) VALUES (?, ?)",
		scriptID, token,
	)
	return err
}

func (db *DB) GetShareToken(token string) (*ShareToken, error) {
	st := &ShareToken{}
	err := db.QueryRow(
		"SELECT id, script_id, token, revoked, created_at FROM share_tokens WHERE token = ?",
		token,
	).Scan(&st.ID, &st.ScriptID, &st.Token, &st.Revoked, &st.CreatedAt)
	
	if err == sql.ErrNoRows {
		return nil, errors.New("token not found")
	}
	return st, err
}

func (db *DB) RevokeShareToken(token string, userID int64) error {
	result, err := db.Exec(
		"UPDATE share_tokens st JOIN scripts s ON st.script_id = s.id SET st.revoked = TRUE WHERE st.token = ? AND s.user_id = ?",
		token, userID,
	)
	if err != nil {
		return err
	}
	
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("token not found or unauthorized")
	}
	return nil
}

func (db *DB) GetUserLimits(userID int64, defaultMaxScripts int, defaultMaxScriptSize int64) (int, int64, error) {
	// Get tier limits first
	tier, err := db.GetUserTier(userID)
	maxScripts := defaultMaxScripts
	maxScriptSize := defaultMaxScriptSize
	
	if err == nil {
		maxScripts = tier.MaxScripts
		maxScriptSize = tier.MaxStorageBytes
	}
	
	// Check for user-specific overrides
	var overrideScripts sql.NullInt32
	var overrideSize sql.NullInt64
	
	err = db.QueryRow(
		"SELECT max_scripts, max_script_size FROM user_limits WHERE user_id = ?",
		userID,
	).Scan(&overrideScripts, &overrideSize)
	
	// Apply overrides if set
	if err == nil {
		if overrideScripts.Valid {
			maxScripts = int(overrideScripts.Int32)
		}
		if overrideSize.Valid {
			maxScriptSize = overrideSize.Int64
		}
	}
	
	return maxScripts, maxScriptSize, nil
}

func (db *DB) SetUserLimits(userID int64, maxScripts *int, maxScriptSize *int64) error {
	_, err := db.Exec(
		"INSERT INTO user_limits (user_id, max_scripts, max_script_size) VALUES (?, ?, ?) ON DUPLICATE KEY UPDATE max_scripts = ?, max_script_size = ?",
		userID, maxScripts, maxScriptSize, maxScripts, maxScriptSize,
	)
	return err
}
