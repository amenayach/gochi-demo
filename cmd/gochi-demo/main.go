package main

import (
	"embed"
	"html/template"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gochi-demo/internal/app"
	"github.com/gochi-demo/internal/database"
	"github.com/gochi-demo/internal/handlers"
	"github.com/gochi-demo/internal/web"
	"github.com/jmoiron/sqlx"
)

// //go:embed ../../templates/*.html
// var tplFS embed.FS

// //go:embed ../../static/*
// var staticFS embed.FS

//
// ─── TEMPLATE PARSER ─────────────────────────────────────────────────────────
//

var templates = template.Must(template.ParseFS(web.FS, "templates/*.html"))

//
// ─── STATIC FILE SERVER FOR EMBED FS ─────────────────────────────────────────
//

func EmbeddedFileServer(r chi.Router, route string, fsys embed.FS) {
	// Convert embed.FS → http.FileSystem
	fs := http.FS(fsys)

	// Ensure /static/* matches
	if route != "/" && route[len(route)-1] != '/' {
		route += "/"
	}

	// Example: "/static/*"
	r.Handle(route+"*", http.StripPrefix(route, http.FileServer(fs)))
}

//
// ─── MAIN ───────────────────────────────────────────────────────────────────
//

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

	// Serve embedded static files
	EmbeddedFileServer(r, "/", web.FS)

	//
	// HTML ROUTE USING EMBEDDED TEMPLATES
	//
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		data := map[string]any{
			"Title": "Hello from Go Embedded Templates!",
			"Msg":   "This is rendered from template files bundled inside the binary.",
		}

		err := templates.ExecuteTemplate(w, "index.html", data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	// Routes
	h := handlers.NewUserHandler(a)
	r.Get("/users/{id}", h.GetUser)
	handlers.InitTestHandler(r)

	database.InitDB(db)

	http.ListenAndServe(":10000", r)
}
