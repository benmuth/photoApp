package main

import (
	"database/sql"
	"log"
	"math/rand"
	"net/http"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func()

func main() {
	db, err := sql.Open("sqlite3", "/Users/moose1/Documents/photoApp/testDB")
	defer db.Close()
	if err != nil {
		log.Printf("failed to open database: %s", err)
		return
	}

	_, err = db.Exec("create table foo (value int)")
	if err != nil {
		log.Printf("failed to create table: %s", err)
	}

	_, err = db.Exec("insert into foo (value) values (2)")

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(time.Duration(rand.Float64() * float64(time.Second)))
		_, err := db.Exec("insert into foo (value) values (1)")
		if err != nil {
			log.Printf("failed to insert into db: %s", err)
		}
		_, err = db.Query("select * from foo")
		if err != nil {
			log.Printf("failed to query db: %s", err)
		}
		func() {
			r, err := http.NewRequest("GET", "localhost:8082/a")
			if err != nil {
				log.Printf("failed to reach /a: %s", err)
			}

		}()

	})

	http.HandleFunc("/a", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(time.Duration(rand.Float64() * float64(time.Second)))
		_, err := db.Exec("insert into foo (value) values (1)")
		if err != nil {
			log.Printf("failed to insert into db: %s", err)
		}
		_, err = db.Query("select * from foo")
		if err != nil {
			log.Printf("failed to query db: %s", err)
		}
		
	})

	http.ListenAndServe(":8082", nil)
	_, err = db.Exec("drop table foo")
	if err != nil {
		log.Printf("failed to drop table: %s", err)
	}
}
