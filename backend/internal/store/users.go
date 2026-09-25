package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

// User es una fila de users.
type User struct {
	Email        string
	Name         string
	PasswordHash string
}

// SeedUser crea el usuario demo sin sobrescribirlo (RF-D-06).
func (s *Store) SeedUser(ctx context.Context, email, name, hash string) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO users (email, name, password_hash) VALUES ($1, $2, $3) ON CONFLICT (email) DO NOTHING`, email, name, hash)
	return err
}

// GetUser busca por email exacto; nil si no existe.
func (s *Store) GetUser(ctx context.Context, email string) (*User, error) {
	var u User
	err := s.pool.QueryRow(ctx, `SELECT email, name, password_hash FROM users WHERE email = $1`, email).Scan(&u.Email, &u.Name, &u.PasswordHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}
