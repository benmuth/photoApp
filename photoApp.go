package main

import (
	"database/sql"
	"fmt"
	_ "image/jpeg"
	_ "image/png"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

// these functions are to be used with a database that includes following tables (! = primary key):
// userid!|email			albumid!|userid|name			photoid!|albumid|userid		albumid|userid

type photoInfo struct {
	photoID int
	albumID int
	userID  int
}

func check(e error) {
	if e != nil {
		log.Fatal(e)
	}
}

func openDB() *sql.DB {
	//dbName := fmt.Sprintf("/Users/moose1/Downloads/%s", name)
	db, err := sql.Open("sqlite3", "/Users/moose1/Downloads/photoApp")
	check(err)
	return db
}

// create a new user along with an initial album
func newUser(email string, database string) {
	db, err := sql.Open("sqlite3", database)
	check(err)
	defer db.Close()

	r, err := db.Exec("insert into users (email) values (?)", email)
	check(err)

	userID, err := r.LastInsertId()
	check(err)
	mainAlbum := fmt.Sprintf("%s's Photos", email)
	r, err = db.Exec("insert into albums (user_id, name) values (?, ?)", userID, mainAlbum)
	check(err)
	albumID, err := r.LastInsertId()
	check(err)
	givePerm(albumID, userID, database)
}

func newAlbum(name string, userID int64, database string) {
	db, err := sql.Open("sqlite3", database)
	check(err)
	defer db.Close()

	r, err := db.Exec("insert into albums (name, user_id) values (?, ?)", name, userID)
	check(err)
	albumID, err := r.LastInsertId()
	check(err)
	givePerm(albumID, userID, database)
}

// checks if the given user has permission to access the given album
func checkPerm(albumID int64, userID int64, database string) bool {
	db, err := sql.Open("sqlite3", database)
	check(err)
	defer db.Close()

	//retrieve all albums that a user has access to
	permittedAlbumRows, err := db.Query("select album_id from album_permissions where user_id = ? and album_id = ?", userID, albumID)
	defer permittedAlbumRows.Close()
	check(err)
	// copy all album ids that the specified user has access to into a slice
	permittedAlbum := make([]int64, 0)
	for i := 0; permittedAlbumRows.Next(); i++ {
		var newElem int64
		permittedAlbum = append(permittedAlbum, newElem)
		err = permittedAlbumRows.Scan(&permittedAlbum[i])
		check(err)
		//i++
	}
	var hasPerm bool
	if len(permittedAlbum) > 0 {
		hasPerm = true
	}
	return hasPerm
	/*
		//columns, err := permittedAlbumRows.Columns()
		//fmt.Printf("columns of query from album permissions: %s\n", columns)
		//fmt.Printf("permitted albums for user %v: %v\n", userID, permittedAlbums)

		// iterate through the slice of album ids until an id matches the specified album id parameter and set hasPerm accordingly
		var hasPerm bool
		for _, permittedAlbum := range permittedAlbums {
			if permittedAlbum == albumID {
				hasPerm = true
				break
			}
		}
		return hasPerm
	*/
}

// add a photo to a specified album if the calling user has permission according to the album_permissions table
func addPhoto(albumID int64, userID int64, database string) {
	db, err := sql.Open("sqlite3", database)
	check(err)
	defer db.Close()

	if checkPerm(albumID, userID, database) == true {
		_, err := db.Exec("insert into photos (user_id, album_id) values (?, ?)", userID, albumID)
		check(err)

	} else {
		fmt.Printf("That user doesn't have permission to access the album!\n")
	}
}

// give a user permission to view and add photos to an album
func givePerm(albumID int64, userID int64, database string) {
	db, err := sql.Open("sqlite3", database)
	check(err)
	defer db.Close()

	if checkPerm(albumID, userID, database) == false {
		_, err := db.Exec("insert into album_permissions (album_id, user_id) values (?, ?)", albumID, userID)
		check(err)
	} else {
		fmt.Printf("That user already has permission to access the album!\n")
	}
}

func main() {
	/*fmt.Printf("-add two new users\n")
	newUser("one@ex.com")
	newUser("two@ex.com")
	*/
	maindb := "/Users/moose1/Downloads/photoApp"
	fmt.Printf("User 1 can access album 3: %v\n", checkPerm(3, 1, maindb))
	fmt.Printf("User 2 can access album 1: %v\n", checkPerm(2, 1, maindb))
	//testdb := ":memory:"
	//fmt.Printf("-both users add a photo to their main album \n")
	//addPhoto(1, 1, maindb)
	//addPhoto(2, 2, maindb)
	/*
		fmt.Printf("-user one adds a new album\n")
		newAlbum("one's birthday trip", 1)
	*/
	//fmt.Printf("-user one adds a photo to their new album\n")
	//addPhoto(3, 1, maindb)
	//fmt.Printf("-user two tries to add a photo to one's album\n")
	//addPhoto(3, 2, maindb)
	//fmt.Printf("-user one shares the album with user two\n")
	//givePerm(3, 2)
	//fmt.Printf("-user two adds the photo to one's album\n")
	//addPhoto(3, 2)
}
