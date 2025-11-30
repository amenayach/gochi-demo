package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gochi-demo/internal/app"
	"github.com/gochi-demo/internal/database"
	"github.com/gochi-demo/internal/handlers"
	"github.com/jmoiron/sqlx"
)

func main() {
	// Open database using sqlx
	// db, err := sqlx.Open("sqlite", "app.db")
	db, err := sqlx.Open("sqlite", "file:app.db?cache=shared&mode=rwc")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	a := &app.App{
		Users: database.NewSQLiteUserStore(db),
	}

	r := chi.NewRouter()

	// Middlewares
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Routes
	h := handlers.NewUserHandler(a)
	r.Get("/users/{id}", h.GetUser)
	handlers.InitTestHandler(r)

	database.InitDB(db)

	http.ListenAndServe(":3000", r)
}
