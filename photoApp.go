package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

func check(e error) {
	if e != nil {
		log.Fatal(e)
	}
}

func newUser(email string) {
	db, err := sql.Open("sqlite3", "/Users/moose1/Downloads/photoApp")
	defer db.Close()
	check(err)

	r, err := db.Exec("insert into users (email) values (?)", email)
	check(err)

	userID, err := r.LastInsertId()
	check(err)
	firstAlbum := fmt.Sprintf("%s's Photos", email)
	_, err = db.Exec("insert into albums (user_id, name) values (?, ?)", userID, firstAlbum)
	check(err)
}

func main() {
	newUser("hello@example.com")
}
