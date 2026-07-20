package auth

import (
	"database/sql"
	"time"
)

// User represents a row in the users table.
type User struct {
	ID           int
	Username     string
	PasswordHash string
	Role         string
	CreatedAt    time.Time
}

// UserStore provides user persistence operations against PostgreSQL.
type UserStore struct {
	db *sql.DB
}

// NewUserStore wraps an existing database connection.
func NewUserStore(db *sql.DB) *UserStore {
	return &UserStore{db: db}
}

// CreateUser inserts a new user with a bcrypt-hashed password.
func (s *UserStore) CreateUser(username, password, role string) error {
	hash, err := HashPassword(password)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(
		"INSERT INTO users (username, password_hash, role) VALUES ($1, $2, $3)",
		username, hash, role,
	)
	return err
}

// GetUserByUsername looks up a user by their unique username.
func (s *UserStore) GetUserByUsername(username string) (*User, error) {
	u := &User{}
	err := s.db.QueryRow(
		"SELECT id, username, password_hash, role, created_at FROM users WHERE username = $1",
		username,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}
