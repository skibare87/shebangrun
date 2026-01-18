package database

import (
	"database/sql"
	"errors"
)

func (db *DB) CreateKeyPair(userID int64, name, publicKey string) (*KeyPair, error) {
	result, err := db.Exec(
		"INSERT INTO keypairs (user_id, name, public_key) VALUES (?, ?, ?)",
		userID, name, publicKey,
	)
	if err != nil {
		return nil, err
	}
	
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	
	return db.GetKeyPairByID(id)
}

func (db *DB) GetKeyPairByID(id int64) (*KeyPair, error) {
	kp := &KeyPair{}
	err := db.QueryRow(
		"SELECT id, user_id, name, public_key, created_at FROM keypairs WHERE id = ?",
		id,
	).Scan(&kp.ID, &kp.UserID, &kp.Name, &kp.PublicKey, &kp.CreatedAt)
	
	if err == sql.ErrNoRows {
		return nil, errors.New("keypair not found")
	}
	return kp, err
}

func (db *DB) GetKeyPairsByUserID(userID int64) ([]*KeyPair, error) {
	rows, err := db.Query(
		"SELECT id, user_id, name, public_key, created_at FROM keypairs WHERE user_id = ? ORDER BY created_at DESC",
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var keypairs []*KeyPair
	for rows.Next() {
		kp := &KeyPair{}
		if err := rows.Scan(&kp.ID, &kp.UserID, &kp.Name, &kp.PublicKey, &kp.CreatedAt); err != nil {
			return nil, err
		}
		keypairs = append(keypairs, kp)
	}
	return keypairs, rows.Err()
}

func (db *DB) DeleteKeyPair(id, userID int64) error {
	result, err := db.Exec("DELETE FROM keypairs WHERE id = ? AND user_id = ?", id, userID)
	if err != nil {
		return err
	}
	
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("keypair not found")
	}
	return nil
}
