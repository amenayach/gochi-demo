package app

import "github.com/gochi-demo/internal/database"

type App struct {
	Users   database.UserStore
	PgUsers database.PgUserStore
}
