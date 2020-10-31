package main

import (
	"database/sql"
	"log"
	"net/http"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	db, err := sql.Open("sqlite3", "/Users/moose1/Documents/photoApp/testDB")
	defer db.Close()
	if err != nil {
		log.Printf("failed to open database: %s", err)
		return
	}

	_, err = db.Exec("insert into ex (value) values (2)")

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, err := db.Exec("insert into ex (value) values (1)")
		if err != nil {
			log.Printf("failed to insert into db: %s", err)
		}
		_, err = db.Query("select * from ex")
		if err != nil {
			log.Printf("failed to query db: %s", err)
		}

	})

	http.ListenAndServe(":8082", nil)
}
