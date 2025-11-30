package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/gochi-demo/internal/models"
)

func Hello(w http.ResponseWriter, _ *http.Request) {
	w.Write([]byte("Hello from Chi!"))
}

func GetUserID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	w.Write([]byte("User ID: " + id))
}

func GetJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"message": "hello"}`))
}

func GetUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.User{ID: 1, Name: "Amen"})
}

func GetUsers(w http.ResponseWriter, r *http.Request) {
	var req models.CreateUserRequest

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"message": "` + req.Name + `"}`))
}

func InitTestHandler(r *chi.Mux) {

	r.Get("/", Hello)

	r.Get("/test/users/{id}", GetUserID)

	r.Get("/json", GetJSON)

	r.Get("/test/user", GetUser)

	r.Post("/test/users", GetUsers)
}
