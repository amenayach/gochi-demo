package database

import (
	"context"

	"github.com/jmoiron/sqlx"
)

type SQLiteUserStore struct {
	db *sqlx.DB
}

func NewSQLiteUserStore(db *sqlx.DB) *SQLiteUserStore {
	return &SQLiteUserStore{db: db}
}

func (s *SQLiteUserStore) GetByID(ctx context.Context, id int64) (*User, error) {
	var u User
	err := s.db.GetContext(ctx, &u, "SELECT id, name, age FROM users WHERE id = ?", id)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *SQLiteUserStore) Create(ctx context.Context, u *User) error {
	_, err := s.db.ExecContext(ctx, "INSERT INTO users(name, age) VALUES(?, ?)", u.Name, u.Age)
	return err
}
