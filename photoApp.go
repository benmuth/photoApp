package main

/*
type photoInfo struct {
	photoID   int
	albumID   int
	userID    int
	tagged_id int
	time	time.Time
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

const dbInit = "CREATE TABLE users (id integer primary key, email text unique);\n" +
	"CREATE TABLE albums (id integer primary key, user_id integer references users(id), name text not null);\n" +
	"CREATE TABLE photos (id integer primary key, album_id integer references albums(id), user_id integer references users(id));\n" +
	"INSERT INTO users (email) VALUES ('user1@example.com');\n" +
	"INSERT INTO users (email) VALUES ('user2@example.com');\n" +
	"INSERT INTO albums (user_id, name) VALUES (1, '1 main');\n" +
	"INSERT INTO albums (user_id, name) VALUES (2, '2 main');\n" +
	"INSERT INTO albums (user_id, name) VALUES (1, '1s Birthday!');\n" +
	"INSERT INTO photos (album_id, user_id) VALUES (1, 1);\n" +
	"INSERT INTO photos (album_id, user_id) VALUES (1, 1);\n" +
	"INSERT INTO photos (album_id, user_id) VALUES (2, 2);\n" +
	"INSERT INTO photos (album_id, user_id) VALUES (3, 1);\n"

// these functions are to be used with a database that includes following tables (! = primary key):
// users: id!|email		albums: id!|userid|name	     photos: id!|albumid|userid		album_permissions: albumid|userid	tags: photo_id|tagged_id

func check(e error) {
	if e != nil {
		log.Fatal(e)
	}
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
}

// add a photo to a specified album if the calling user has permission according to the album_permissions table
func addPhoto(albumID int64, userID int64, photoPath string, db *sql.DB) int64 {
	var photoID int64
	if checkPerm(albumID, userID, db) == true {
		res, err := db.Exec("INSERT INTO photos (user_id, album_id) VALUES (?, ?)", userID, albumID)
		check(err)

		photoID, err = res.LastInsertId()
		check(err)
		photoData, err := ioutil.ReadFile(photoPath)
		check(err)
		err = ioutil.WriteFile("Photos/"+strconv.FormatInt(photoID, 10), photoData, 00007)
		check(err)
	} else {
		fmt.Printf("That user doesn't have permission to access the album!\n")
	}
	return photoID
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

var templates = template.Must(template.ParseFiles("templates/home.html"))

type page interface {
	query() string
	render(w http.ResponseWriter, r sql.Result)
}

type homepage struct {
	UserID int64
	Albums []int64
}

type albumpage struct {
	AlbumID int64
	Photos  []int64
}

func (h homepage) query() string {
	//TODO: get user_id from http request header
	return "SELECT id FROM albums WHERE user_id = 1"
}

func (a albumpage) query() string {
	return "SELECT id FROM photos WHERE album_id = 1"
}

// TODO: handle errors instead of panicking
func (h homepage) render(w http.ResponseWriter, r *sql.Rows) error {
	albums := make([]int64, 0)
	for i := 0; r.Next(); i++ {
		var newElem int64
		albums = append(albums, newElem)
		err := r.Scan(&albums[i])
		if err != nil {
			return err
		}
	}
	fmt.Printf("query result: %+v\n", albums)
	h.Albums = albums
	h.UserID = 1
	return templates.ExecuteTemplate(w, "home.html", h)
}

func (a albumpage) render(w http.ResponseWriter, r *sql.Rows) error {
	photos := make([]int64, 0)
	for i := 0; r.Next(); i++ {
		var newElem int64
		photos = append(photos, newElem)
		err := r.Scan(&photos[i])
		if err != nil {
			return err
		}
	}
	a.Photos = photos
	a.AlbumID = 1
	return templates.ExecuteTemplate(w, "album.html")
}

func homeHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	h := homepage{}
	result, err := db.Query(h.query())
	if err != nil {
		log.Panic("ERROR: invalid user id\n")
	}
	err = h.render(w, result)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func albumHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	a := albumpage{}
	result, err := db.Query(a.query())
	if err != nil {
		log.Panic("ERROR: invalid album\n")
	}
	err = a.render(w, result)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func makeHandler(fn func(http.ResponseWriter, *http.Request, *sql.DB)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		db, err := sql.Open("sqlite3", ":memory:")
		check(err)
		defer db.Close()

		_, err = db.Exec(dbInit)
		check(err)

		fn(w, r, db)
	}
}

func main() {
	http.HandleFunc("/home/", makeHandler(homeHandler))
	http.HandleFunc("/album/", makeHandler(albumHandler))
	log.Fatal(http.ListenAndServe(":8080", nil))
}
