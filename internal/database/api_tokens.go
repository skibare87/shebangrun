package database

import (
	"database/sql"
	"errors"
)

type APIToken struct {
	ID           int64
	UserID       int64
	Name         string
	ClientID     string
	ClientSecret string
	CreatedAt    string
	LastUsed     sql.NullString
}

func (db *DB) CreateAPIToken(userID int64, name, clientID, clientSecret string) (*APIToken, error) {
	result, err := db.Exec(
		"INSERT INTO api_tokens (user_id, name, client_id, client_secret) VALUES (?, ?, ?, ?)",
		userID, name, clientID, clientSecret,
	)
	if err != nil {
		return nil, err
	}
	
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	
	return db.GetAPITokenByID(id)
}

func (db *DB) GetAPITokenByID(id int64) (*APIToken, error) {
	token := &APIToken{}
	err := db.QueryRow(
		"SELECT id, user_id, name, client_id, client_secret, created_at, last_used FROM api_tokens WHERE id = ?",
		id,
	).Scan(&token.ID, &token.UserID, &token.Name, &token.ClientID, &token.ClientSecret, &token.CreatedAt, &token.LastUsed)
	
	if err == sql.ErrNoRows {
		return nil, errors.New("token not found")
	}
	return token, err
}

func (db *DB) GetAPITokenByClientID(clientID string) (*APIToken, error) {
	token := &APIToken{}
	err := db.QueryRow(
		"SELECT id, user_id, name, client_id, client_secret, created_at, last_used FROM api_tokens WHERE client_id = ?",
		clientID,
	).Scan(&token.ID, &token.UserID, &token.Name, &token.ClientID, &token.ClientSecret, &token.CreatedAt, &token.LastUsed)
	
	if err == sql.ErrNoRows {
		return nil, errors.New("token not found")
	}
	return token, err
}

func (db *DB) GetAPITokensByUserID(userID int64) ([]*APIToken, error) {
	rows, err := db.Query(
		"SELECT id, user_id, name, client_id, client_secret, created_at, last_used FROM api_tokens WHERE user_id = ? ORDER BY created_at DESC",
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var tokens []*APIToken
	for rows.Next() {
		t := &APIToken{}
		if err := rows.Scan(&t.ID, &t.UserID, &t.Name, &t.ClientID, &t.ClientSecret, &t.CreatedAt, &t.LastUsed); err != nil {
			return nil, err
		}
		tokens = append(tokens, t)
	}
	return tokens, rows.Err()
}

func (db *DB) DeleteAPIToken(id, userID int64) error {
	result, err := db.Exec("DELETE FROM api_tokens WHERE id = ? AND user_id = ?", id, userID)
	if err != nil {
		return err
	}
	
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("token not found")
	}
	return nil
}

func (db *DB) UpdateAPITokenLastUsed(clientID string) error {
	_, err := db.Exec("UPDATE api_tokens SET last_used = NOW() WHERE client_id = ?", clientID)
	return err
}
