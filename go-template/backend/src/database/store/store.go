package store

import (
	"context"
	"database/sql"
	"errors"
)

var (
	ErrNotFound    = errors.New("not found")
	ErrEmptyUpdate = errors.New("at least one field is required")
	ErrBadRequest  = errors.New("bad request")
	ErrConflict    = errors.New("conflict")
)

type Store struct {
	db *sql.DB
}

type AuthUser struct {
	ID     int
	Email  string
	Role   string
	Status string
}

func New(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) DB() *sql.DB {
	return s.db
}

func (s *Store) ActiveAuthUser(ctx context.Context, id int) (AuthUser, error) {
	var user AuthUser
	err := s.db.QueryRowContext(ctx, `SELECT "id", "email", "role"::TEXT, "status"::TEXT FROM "User" WHERE "id" = $1`, id).
		Scan(&user.ID, &user.Email, &user.Role, &user.Status)
	if errors.Is(err, sql.ErrNoRows) {
		return AuthUser{}, ErrNotFound
	}
	if err != nil {
		return AuthUser{}, err
	}
	if user.Status != "Active" {
		return AuthUser{}, ErrNotFound
	}
	return user, nil
}
