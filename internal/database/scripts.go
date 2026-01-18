package database

import (
	"database/sql"
	"errors"
)

func (db *DB) CreateScript(userID int64, name, description, visibility string) (*Script, error) {
	result, err := db.Exec(
		"INSERT INTO scripts (user_id, name, description, visibility) VALUES (?, ?, ?, ?)",
		userID, name, description, visibility,
	)
	if err != nil {
		return nil, err
	}
	
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	
	return db.GetScriptByID(id)
}

func (db *DB) GetScriptByID(id int64) (*Script, error) {
	script := &Script{}
	err := db.QueryRow(
		"SELECT id, user_id, name, description, visibility, created_at, updated_at FROM scripts WHERE id = ?",
		id,
	).Scan(&script.ID, &script.UserID, &script.Name, &script.Description, &script.Visibility, &script.CreatedAt, &script.UpdatedAt)
	
	if err == sql.ErrNoRows {
		return nil, errors.New("script not found")
	}
	return script, err
}

func (db *DB) GetScriptByUserAndName(userID int64, name string) (*Script, error) {
	script := &Script{}
	err := db.QueryRow(
		"SELECT id, user_id, name, description, visibility, created_at, updated_at FROM scripts WHERE user_id = ? AND name = ?",
		userID, name,
	).Scan(&script.ID, &script.UserID, &script.Name, &script.Description, &script.Visibility, &script.CreatedAt, &script.UpdatedAt)
	
	if err == sql.ErrNoRows {
		return nil, errors.New("script not found")
	}
	return script, err
}

func (db *DB) GetScriptsByUserID(userID int64) ([]*Script, error) {
	rows, err := db.Query(
		"SELECT id, user_id, name, description, visibility, created_at, updated_at FROM scripts WHERE user_id = ? ORDER BY updated_at DESC",
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var scripts []*Script
	for rows.Next() {
		s := &Script{}
		if err := rows.Scan(&s.ID, &s.UserID, &s.Name, &s.Description, &s.Visibility, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		scripts = append(scripts, s)
	}
	return scripts, rows.Err()
}

func (db *DB) UpdateScript(id int64, description, visibility string) error {
	_, err := db.Exec(
		"UPDATE scripts SET description = ?, visibility = ? WHERE id = ?",
		description, visibility, id,
	)
	return err
}

func (db *DB) DeleteScript(id, userID int64) error {
	result, err := db.Exec("DELETE FROM scripts WHERE id = ? AND user_id = ?", id, userID)
	if err != nil {
		return err
	}
	
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("script not found")
	}
	return nil
}

func (db *DB) GetScriptCount(userID int64) (int, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM scripts WHERE user_id = ?", userID).Scan(&count)
	return count, err
}

func (db *DB) CreateScriptVersion(scriptID int64, version int, contentHash, signature, checksum string, size int64) (*ScriptVersion, error) {
	result, err := db.Exec(
		"INSERT INTO script_versions (script_id, version, content_hash, signature, checksum, size) VALUES (?, ?, ?, ?, ?, ?)",
		scriptID, version, contentHash, signature, checksum, size,
	)
	if err != nil {
		return nil, err
	}
	
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	
	return db.GetScriptVersionByID(id)
}

func (db *DB) GetScriptVersionByID(id int64) (*ScriptVersion, error) {
	sv := &ScriptVersion{}
	err := db.QueryRow(
		"SELECT id, script_id, version, content_hash, signature, checksum, size, created_at FROM script_versions WHERE id = ?",
		id,
	).Scan(&sv.ID, &sv.ScriptID, &sv.Version, &sv.ContentHash, &sv.Signature, &sv.Checksum, &sv.Size, &sv.CreatedAt)
	
	if err == sql.ErrNoRows {
		return nil, errors.New("version not found")
	}
	return sv, err
}

func (db *DB) GetLatestScriptVersion(scriptID int64) (*ScriptVersion, error) {
	sv := &ScriptVersion{}
	err := db.QueryRow(
		"SELECT id, script_id, version, content_hash, signature, checksum, size, created_at FROM script_versions WHERE script_id = ? ORDER BY version DESC LIMIT 1",
		scriptID,
	).Scan(&sv.ID, &sv.ScriptID, &sv.Version, &sv.ContentHash, &sv.Signature, &sv.Checksum, &sv.Size, &sv.CreatedAt)
	
	if err == sql.ErrNoRows {
		return nil, errors.New("no versions found")
	}
	return sv, err
}

func (db *DB) GetScriptVersionByNumber(scriptID int64, version int) (*ScriptVersion, error) {
	sv := &ScriptVersion{}
	err := db.QueryRow(
		"SELECT id, script_id, version, content_hash, signature, checksum, size, created_at FROM script_versions WHERE script_id = ? AND version = ?",
		scriptID, version,
	).Scan(&sv.ID, &sv.ScriptID, &sv.Version, &sv.ContentHash, &sv.Signature, &sv.Checksum, &sv.Size, &sv.CreatedAt)
	
	if err == sql.ErrNoRows {
		return nil, errors.New("version not found")
	}
	return sv, err
}

func (db *DB) SaveScriptContent(versionID int64, content []byte, storagePath string, encryptionKeyID *int64, wrappedKey []byte) error {
	_, err := db.Exec(
		"INSERT INTO script_content (version_id, content, storage_path, encryption_key_id, wrapped_key) VALUES (?, ?, ?, ?, ?)",
		versionID, content, storagePath, encryptionKeyID, wrappedKey,
	)
	return err
}

func (db *DB) GetScriptContent(versionID int64) (*ScriptContent, error) {
	sc := &ScriptContent{VersionID: versionID}
	var encKeyID sql.NullInt64
	var wrappedKey sql.NullString
	err := db.QueryRow(
		"SELECT content, storage_path, encryption_key_id, wrapped_key FROM script_content WHERE version_id = ?",
		versionID,
	).Scan(&sc.Content, &sc.StoragePath, &encKeyID, &wrappedKey)
	
	if err == sql.ErrNoRows {
		return nil, errors.New("content not found")
	}
	if encKeyID.Valid {
		val := encKeyID.Int64
		sc.EncryptionKeyID = &val
	}
	if wrappedKey.Valid {
		sc.WrappedKey = []byte(wrappedKey.String)
	}
	return sc, err
}

func (db *DB) CreateTag(scriptID int64, tagName string, versionID int64) error {
	_, err := db.Exec(
		"INSERT INTO tags (script_id, tag_name, version_id) VALUES (?, ?, ?) ON DUPLICATE KEY UPDATE version_id = ?",
		scriptID, tagName, versionID, versionID,
	)
	return err
}

func (db *DB) GetVersionByTag(scriptID int64, tagName string) (*ScriptVersion, error) {
	sv := &ScriptVersion{}
	err := db.QueryRow(
		"SELECT sv.id, sv.script_id, sv.version, sv.content_hash, sv.signature, sv.checksum, sv.size, sv.created_at FROM script_versions sv JOIN tags t ON sv.id = t.version_id WHERE t.script_id = ? AND t.tag_name = ?",
		scriptID, tagName,
	).Scan(&sv.ID, &sv.ScriptID, &sv.Version, &sv.ContentHash, &sv.Signature, &sv.Checksum, &sv.Size, &sv.CreatedAt)
	
	if err == sql.ErrNoRows {
		return nil, errors.New("tag not found")
	}
	return sv, err
}
