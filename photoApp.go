package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

// these functions are to be used with a database that includes following tables (! = primary key):
// userid!|email			albumid!|userid|name			photoid!|albumid|userid		albumid|userid

func check(e error) {
	if e != nil {
		log.Fatal(e)
	}
}

func openDB() *sql.DB {
	db, err := sql.Open("sqlite3", "/Users/moose1/Downloads/photoApp")
	check(err)
	return db
}

// create a new user along with an initial album
func newUser(email string) {
	db := openDB()
	defer db.Close()

	r, err := db.Exec("insert into users (email) values (?)", email)
	check(err)

	userID, err := r.LastInsertId()
	check(err)
	firstAlbum := fmt.Sprintf("%s's Photos", email)
	r, err = db.Exec("insert into albums (user_id, name) values (?, ?)", userID, firstAlbum)
	check(err)
	albumID, err := r.LastInsertId()
	check(err)
	givePerm(albumID, userID)
}

func newAlbum(name string, userID int64) {
	db := openDB()
	defer db.Close()

	r, err := db.Exec("insert into albums (name, user_id) values (?, ?)", name, userID)
	check(err)
	albumID, err := r.LastInsertId()
	check(err)
	givePerm(albumID, userID)
}

// checks if the given user has permission to access the given album
func checkPerm(albumID int64, userID int64) bool {
	db := openDB()
	defer db.Close()

	permittedAlbumRows, err := db.Query("select album_id from album_permissions where user_id = ?", userID)
	defer permittedAlbumRows.Close()
	check(err)

	// copy all album ids that the specified user has access to into a slice
	permittedAlbums := make([]int64, 0)
	var i int
	for permittedAlbumRows.Next() {
		var newElem int64
		permittedAlbums = append(permittedAlbums, newElem)
		err = permittedAlbumRows.Scan(permittedAlbums[i])
		i++
	}

	// iterate through the slice of album ids until an id matches the specified album id parameter and set hasPerm accordingly
	var hasPerm bool
	for _, permittedAlbum := range permittedAlbums {
		if permittedAlbum == albumID {
			hasPerm = true
			break
		}
	}
	return hasPerm
}

// add a photo to a specified album if the calling user has permission according to the album_permissions table
func addPhoto(albumID int64, userID int64) {
	db := openDB()
	defer db.Close()

	if checkPerm(albumID, userID) == true {
		_, err := db.Exec("insert into photos (user_id, album_id) values (?, ?)", userID, albumID)
		check(err)
	} else {
		fmt.Printf("That user doesn't have permission to access the album!\n")
	}
}

// give a user permission to view and add photos to an album
func givePerm(albumID int64, userID int64) {
	db := openDB()
	defer db.Close()

	if checkPerm(albumID, userID) == false {
		_, err := db.Exec("insert into album_permissions (album_id, user_id) values (?, ?)", albumID, userID)
		check(err)
	} else {
		fmt.Printf("That user already has permission to access the album!\n")
	}
}

func main() {
	addPhoto(3, 1)
	addPhoto(3, 2)
	givePerm(3, 2)
	addPhoto(3, 2)
}
