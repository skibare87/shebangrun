package database

// CanAccessScript checks if a user can access a script based on ACL
func (db *DB) CanAccessScript(scriptID int64, userID *int64) (bool, error) {
	var ownerID int64
	var visibility string
	err := db.QueryRow("SELECT user_id, visibility FROM scripts WHERE id = ?", scriptID).
		Scan(&ownerID, &visibility)
	if err != nil {
		return false, err
	}
	
	// Owner always has access
	if userID != nil && *userID == ownerID {
		return true, nil
	}
	
	// Public scripts - anyone
	if visibility == "public" {
		return true, nil
	}
	
	// Private scripts - handled separately (share tokens or encryption)
	if visibility == "private" {
		return false, nil
	}
	
	// Unlisted - check ACL
	var count int
	
	// Check for "anyone with link" access
	err = db.QueryRow(`
		SELECT COUNT(*) FROM script_access 
		WHERE script_id = ? AND access_type = 'link' 
		AND (expires_at IS NULL OR expires_at > NOW())
	`, scriptID).Scan(&count)
	if err != nil {
		return false, err
	}
	if count > 0 {
		return true, nil
	}
	
	// Check for specific user access
	if userID != nil {
		err = db.QueryRow(`
			SELECT COUNT(*) FROM script_access 
			WHERE script_id = ? AND access_type = 'user' AND user_id = ?
			AND (expires_at IS NULL OR expires_at > NOW())
		`, scriptID, *userID).Scan(&count)
		if err != nil {
			return false, err
		}
		if count > 0 {
			return true, nil
		}
	}
	
	return false, nil
}
