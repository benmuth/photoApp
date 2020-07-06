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
	_, err := db.Exec("insert into users (email) values (?)", email)
	check(err)
}

func main() {
	fmt.Printf("%v\n", newUser("hello@example.com"))
}
