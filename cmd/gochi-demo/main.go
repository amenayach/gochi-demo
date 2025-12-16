package main

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gochi-demo/internal/app"
	"github.com/gochi-demo/internal/auth"
	"github.com/gochi-demo/internal/config"
	"github.com/gochi-demo/internal/database"
	"github.com/gochi-demo/internal/handlers"
	"github.com/gochi-demo/internal/web"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// ─── TEMPLATE PARSER ─────────────────────────────────────────────────────────
var templates = template.Must(template.ParseFS(web.FS, "templates/*.html"))

// ─── STATIC FILE SERVER FOR EMBED FS ─────────────────────────────────────────
func EmbeddedFileServer(r chi.Router, route string, fsys embed.FS) {
	// Convert embed.FS → http.FileSystem
	fs := http.FS(fsys)

	// Ensure /static/* matches
	if route != "/" && route[len(route)-1] != '/' {
		route += "/"
	}

	// r.Handle("/templates/*", func (w *http.ResponseWriter, r *http.Request)  {
	// 	http.Redirect(w, r, "/", http.StatusSeeOther)
	// })

	// Example: "/static/*"
	r.Handle(route+"*", http.StripPrefix(route, http.FileServer(fs)))
}

// ─── SERVE TEMPLATES AT ROOT PATH ────────────────────────────────────────────
// Serves files from templates/ subdirectory at "/" (root path)
func ServeTemplatesAtRoot(r chi.Router, fsys embed.FS, subdir string) {
	// Get the subdirectory from the embedded FS
	subFS, err := fs.Sub(fsys, subdir)
	if err != nil {
		log.Fatalf("Failed to get subdirectory %s: %v", subdir, err)
	}

	// Convert to http.FileSystem
	httpFS := http.FS(subFS)

	// Serve at root path "/"
	// Files from templates/ will be accessible at /filename.html
	r.Handle("/*", http.FileServer(httpFS))
}

// HTML ROUTE USING EMBEDDED TEMPLATES
func HandleTemplates(r *chi.Mux) {
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		user, err := auth.GetUserFromSession(r)
		// if err != nil {
		// http.Error(w, err.Error(), http.StatusForbidden)
		// http.Redirect(w, r, "/login", http.StatusSeeOther)
		// }

		var firstname string
		if err == nil {
			firstname = user.FirstName
		}

		data := map[string]any{
			"Title":    "Hello from Go Embedded Templates!",
			"Msg":      "This is rendered from template files bundled inside the binary.",
			"Username": firstname,
		}

		err = templates.ExecuteTemplate(w, "index.html", data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	r.Get("/posts", func(w http.ResponseWriter, r *http.Request) {
		user, err := auth.GetUserFromSession(r)

		var firstname string
		if err == nil {
			firstname = user.FirstName
		}

		data := map[string]any{
			"Title":    "Hello from Go Embedded Templates!",
			"Msg":      "This is rendered from template files bundled inside the binary.",
			"Username": firstname,
			"Date":     "2025-12-10",
			"Tags":     []string{"cat1", "cat2"},
			"Content":  "This is rendered from template files bundled inside the binary.",
			"PrevPost": map[string]any{
				"URL": "https://google.com/prev",
			},
			"NextPost": map[string]any{
				"URL": "https://google.com/next",
			},
		}

		err = templates.ExecuteTemplate(w, "post.html", data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
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

	clientID := config.GetConfig("CLIENT_ID")
	fmt.Println("ClientID: ", clientID)
	fmt.Println("ClientSecret: ", config.GetConfig("CLIENT_SECRET"))
	fmt.Println("ClientCallbackURL: ", config.GetConfig("CLIENT_CALLBACK_URL"))

	enablePg := strings.ToUpper(config.GetConfig("ENABLE_PG")) == "TRUE"

	var a *app.App

	if enablePg {
		pgdb, err := database.NewPostgres("postgres://admin:admin@localhost:5432/mydatabase?sslmode=disable")
		if err != nil {
			log.Fatal(err)
			// return
		}
		defer pgdb.Close()

		a = &app.App{
			Users:   database.NewSQLiteUserStore(db),
			PgUsers: *database.NewPgUserStore(pgdb),
		}

		database.InitPgDB(pgdb)
	} else {
		a = &app.App{
			Users: database.NewSQLiteUserStore(db),
		}
	}

	r := chi.NewRouter()

	// Middlewares
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	auth.NewAuth(r)

	// Serve embedded static files
	EmbeddedFileServer(r, "/", web.FS)

	// Register specific routes first (before catch-all)
	// Serve templates (for dynamic template rendering)
	HandleTemplates(r)

	// Routes
	h := handlers.NewUserHandler(a)
	r.Get("/users/{id}", h.GetUser)
	r.Get("/users/sqlite/{id}", h.GetSqliteUser)
	handlers.InitTestHandler(r)

	// Serve template files from templates/ folder at root path "/"
	// e.g., templates/test.html will be accessible at /test.html
	// This catch-all should be registered last so specific routes take precedence
	// ServeTemplatesAtRoot(r, web.FS, "templates")

	database.InitSqliteDB(db)

	http.ListenAndServe(":10000", r)
}
