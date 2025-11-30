package database

import "context"

type User struct {
	ID   int    `db:"id"`
	Name string `db:"name"`
	Age  int    `db:"age"`
}

type UserStore interface {
	GetByID(ctx context.Context, id int64) (*User, error)
	Create(ctx context.Context, u *User) error
}
