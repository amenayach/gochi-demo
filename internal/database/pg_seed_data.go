package database

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

func InitPgDB(db *sqlx.DB) {
	// Create table
	// schema := `
	// CREATE TABLE IF NOT EXISTS users
	// (
	// 	id integer NOT NULL DEFAULT nextval('users_id_seq'::regclass),
	// 	name text COLLATE pg_catalog."default" NOT NULL,
	// 	age integer NOT NULL,
	// 	CONSTRAINT users_pkey PRIMARY KEY (id)
	// )

	// TABLESPACE pg_default;`
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		name VARCHAR(100) NOT NULL,
		age INT
	);`

	db.MustExec(schema)

	// user count
	var userCount int
	err := db.Get(&userCount, `select count(*) from users`)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("userCount: %#v\n", userCount)

	// ---- CREATE ----
	if userCount < 100 {
		createUserQuery := `INSERT INTO users (name, age) VALUES (:name, :age)`
		var result sql.Result

		for i := 1; i <= 100; i++ {
			user := User{Name: fmt.Sprintf("Alice%v", i), Age: 25}

			result, err = db.NamedExec(createUserQuery, user)
			if err != nil {
				log.Fatal(err)
			}
		}
		id, _ := result.LastInsertId()
		fmt.Println("Inserted user ID:", id)
	}

	// // ---- READ (single) ----
	// var fetched User
	// err = db.Get(&fetched, `SELECT * FROM users WHERE id = ?`, id)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// fmt.Printf("Fetched user: %#v\n", fetched)

	// // ---- READ (multiple) ----
	// var users []User
	// err = db.Select(&users, `SELECT * FROM users ORDER BY id`)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// fmt.Println("All users:", users)

	// // ---- UPDATE ----
	// updateQuery := `UPDATE users SET age = :age WHERE id = :id`
	// fetched.Age = 30
	// _, err = db.NamedExec(updateQuery, fetched)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// fmt.Println("Updated user age to 30")

	// ---- DELETE ----
	// _, err = db.Exec(`DELETE FROM users WHERE id = ?`, id)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// fmt.Println("Deleted user ID:", id)
}
