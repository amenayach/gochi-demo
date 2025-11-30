package models

type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type CreateUserRequest struct {
	Name string `json:"name"`
}
