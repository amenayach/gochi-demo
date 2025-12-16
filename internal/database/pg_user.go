package database

import (
	"context"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
)

type PgUserStore struct {
	db *sqlx.DB
}

func NewPostgres(dsn string) (*sqlx.DB, error) {
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(30 * time.Minute)

	return db, err
}

func NewPgUserStore(db *sqlx.DB) *PgUserStore {
	return &PgUserStore{db: db}
}

func (s *PgUserStore) GetByID(ctx context.Context, id int64) (*User, error) {
	var u User
	err := s.db.GetContext(ctx, &u, "SELECT id, name, age FROM users WHERE id = $1", id)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *PgUserStore) Create(ctx context.Context, u *User) error {
	_, err := s.db.ExecContext(ctx, "INSERT INTO users(name, age) VALUES(?, ?)", u.Name, u.Age)
	return err
}
