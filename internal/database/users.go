package database

import (
	"database/sql"
	"errors"
)

func (db *DB) CreateUser(username, email, passwordHash, oauthProvider, oauthID string, isAdmin bool) (*User, error) {
	result, err := db.Exec(
		"INSERT INTO users (username, email, password_hash, oauth_provider, oauth_id, is_admin) VALUES (?, ?, ?, ?, ?, ?)",
		username, email, passwordHash, nullString(oauthProvider), nullString(oauthID), isAdmin,
	)
	if err != nil {
		return nil, err
	}
	
	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	
	return db.GetUserByID(id)
}

func (db *DB) GetUserByID(id int64) (*User, error) {
	user := &User{}
	err := db.QueryRow(
		"SELECT id, username, email, password_hash, COALESCE(oauth_provider, ''), COALESCE(oauth_id, ''), is_admin, rate_limit, created_at FROM users WHERE id = ?",
		id,
	).Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.OAuthProvider, &user.OAuthID, &user.IsAdmin, &user.RateLimit, &user.CreatedAt)
	
	if err == sql.ErrNoRows {
		return nil, errors.New("user not found")
	}
	return user, err
}

func (db *DB) GetUserByUsername(username string) (*User, error) {
	user := &User{}
	err := db.QueryRow(
		"SELECT id, username, email, password_hash, COALESCE(oauth_provider, ''), COALESCE(oauth_id, ''), is_admin, rate_limit, created_at FROM users WHERE username = ?",
		username,
	).Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.OAuthProvider, &user.OAuthID, &user.IsAdmin, &user.RateLimit, &user.CreatedAt)
	
	if err == sql.ErrNoRows {
		return nil, errors.New("user not found")
	}
	return user, err
}

func (db *DB) GetUserByEmail(email string) (*User, error) {
	user := &User{}
	err := db.QueryRow(
		"SELECT id, username, email, password_hash, COALESCE(oauth_provider, ''), COALESCE(oauth_id, ''), is_admin, rate_limit, created_at FROM users WHERE email = ?",
		email,
	).Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.OAuthProvider, &user.OAuthID, &user.IsAdmin, &user.RateLimit, &user.CreatedAt)
	
	if err == sql.ErrNoRows {
		return nil, errors.New("user not found")
	}
	return user, err
}

func (db *DB) GetUserByOAuth(provider, oauthID string) (*User, error) {
	user := &User{}
	err := db.QueryRow(
		"SELECT id, username, email, password_hash, COALESCE(oauth_provider, ''), COALESCE(oauth_id, ''), is_admin, rate_limit, created_at FROM users WHERE oauth_provider = ? AND oauth_id = ?",
		provider, oauthID,
	).Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.OAuthProvider, &user.OAuthID, &user.IsAdmin, &user.RateLimit, &user.CreatedAt)
	
	if err == sql.ErrNoRows {
		return nil, errors.New("user not found")
	}
	return user, err
}

func (db *DB) IsFirstUser() (bool, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	return count == 0, err
}

func (db *DB) ListUsers(limit, offset int) ([]*User, error) {
	rows, err := db.Query(
		"SELECT id, username, email, password_hash, COALESCE(oauth_provider, ''), COALESCE(oauth_id, ''), is_admin, rate_limit, created_at FROM users ORDER BY created_at DESC LIMIT ? OFFSET ?",
		limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var users []*User
	for rows.Next() {
		user := &User{}
		if err := rows.Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.OAuthProvider, &user.OAuthID, &user.IsAdmin, &user.RateLimit, &user.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, rows.Err()
}

func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
