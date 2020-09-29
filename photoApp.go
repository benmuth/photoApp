package main

/*
type photoInfo struct {
	photoID   int
	albumID   int
	userID    int
	tagged_id int
}
*/

import (
	"database/sql"
	"fmt"
	"html/template"
	_ "image/jpeg"
	_ "image/png"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"

	_ "github.com/mattn/go-sqlite3"
)

// these functions are to be used with a database that includes following tables (! = primary key):
// users: id!|email		albums: id!|userid|name	     photos: id!|albumid|userid		album_permissions: albumid|userid	tags: photo_id|tagged_id

func check(e error) {
	if e != nil {
		log.Fatal(e)
	}
}

var templates = template.Must(template.ParseFiles("templates/home.html"))

func renderTemplate(w http.ResponseWriter, tmpl string, p *page) {
	err := templates.ExecuteTemplate(w, tmpl+".html", p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

type page struct {
	UserId int64
	Albums []int64
}

// create a new user along with an initial album
func newUser(email string, db *sql.DB) {
	r, err := db.Exec("insert into users (email) values (?)", email)
	check(err)

	userID, err := r.LastInsertId()
	check(err)
	mainAlbum := fmt.Sprintf("%s's Photos", email)
	r, err = db.Exec("insert into albums (user_id, name) values (?, ?)", userID, mainAlbum)
	check(err)
	albumID, err := r.LastInsertId()
	check(err)
	givePerm(albumID, userID, db)
}

func newAlbum(name string, userID int64, db *sql.DB) {
	r, err := db.Exec("insert into albums (name, user_id) values (?, ?)", name, userID)
	check(err)
	albumID, err := r.LastInsertId()
	check(err)
	givePerm(albumID, userID, db)
}

// checks if the given user has permission to access the given album
func checkPerm(albumID int64, userID int64, db *sql.DB) bool {
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
func addPhoto(albumID int64, userID int64, photoPath string, db *sql.DB) {
	if checkPerm(albumID, userID, db) == true {
		res, err := db.Exec("insert into photos (user_id, album_id) values (?, ?)", userID, albumID)
		check(err)
		photoId, err := res.LastInsertId()
		check(err)
		photoData, err := ioutil.ReadFile(photoPath)
		check(err)
		err = ioutil.WriteFile("Photos/"+strconv.Itoa(photoId), photoData, 00007)
		check(err)
	} else {
		fmt.Printf("That user doesn't have permission to access the album!\n")
	}
	//add a tag feature to this function?
}

// give a user permission to view and add photos to an album
func givePerm(albumID int64, userID int64, db *sql.DB) {
	if checkPerm(albumID, userID, db) == false {
		_, err := db.Exec("insert into album_permissions (album_id, user_id) values (?, ?)", albumID, userID)
		check(err)
	} else {
		fmt.Printf("That user already has permission to access the album!\n")
	}
}

func showTags(userID int64, db *sql.DB) ([]int64, []int64) {
	taggedPhotoRows, err := db.Query("SELECT id FROM photos JOIN tags ON photos.id = tags.photo_id WHERE tagged_user_id = ?", userID)
	defer taggedPhotoRows.Close()
	check(err)

	taggedPhotos := make([]int64, 0)
	for i := 0; taggedPhotoRows.Next(); i++ {
		var newElem int64
		taggedPhotos = append(taggedPhotos, newElem)
		err := taggedPhotoRows.Scan(&taggedPhotos[i])
		check(err)
	}

	taggedAlbumRows, err := db.Query("SELECT album_id FROM photos JOIN tags ON photos.id = tags.photo_id WHERE tagged_user_id = ?", userID)
	defer taggedAlbumRows.Close()
	check(err)

	taggedAlbums := make([]int64, 0)
	for i := 0; taggedAlbumRows.Next(); i++ {
		var newElem int64
		taggedAlbums = append(taggedAlbums, newElem)
		err := taggedAlbumRows.Scan(&taggedAlbums[i])
		check(err)
	}

	return taggedPhotos, taggedAlbums
}

func main() {
	/*fmt.Printf("-add two new users\n")
	newUser("one@ex.com")
	newUser("two@ex.com")
	*/
	db, err := sql.Open("sqlite3", "/Users/moose1/Downloads/photoApp")
	check(err)
	defer db.Close()
	fmt.Printf("User 1 can access album 3: %v\n", checkPerm(3, 1, db))
	fmt.Printf("User 2 can access album 1: %v\n", checkPerm(2, 1, db))
}
