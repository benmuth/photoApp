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
	"context"
	"database/sql"
	"fmt"
	"html/template"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

/*
const dbInit = "CREATE TABLE users (id integer primary key, email text unique);\n" +
	"CREATE TABLE albums (id integer primary key, user_id integer references users(id), name text not null);\n" +
	"CREATE TABLE photos (id integer primary key, album_id integer references albums(id), user_id integer references users(id), path TEXT UNIQUE);\n" +
	"CREATE TABLE album_permissions (album_id INTEGER REFERENCES albums(id), user_id INTEGER REFERENCES users(id));\n" +
	"INSERT INTO users (email) VALUES ('user1@example.com');\n" +
	"INSERT INTO users (email) VALUES ('user2@example.com');\n" +
	"INSERT INTO albums (user_id, name) VALUES (1, '1 main');\n" +
	"INSERT INTO albums (user_id, name) VALUES (2, '2 main');\n" +
	"INSERT INTO albums (user_id, name) VALUES (1, '1s Birthday!');\n" +
	"INSERT INTO album_permissions (album_id, user_id) VALUES (1,1);\n" +
	"INSERT INTO album_permissions (album_id, user_id) VALUES (1,2);\n" +
	"INSERT INTO album_permissions (album_id, user_id) VALUES (2,2);\n" +
	"INSERT INTO photos (album_id, user_id, path) VALUES (1, 1, '/Users/moose1/Documents/photoApp/Photos/1.jpg');\n" +
	"INSERT INTO photos (album_id, user_id, path) VALUES (1, 1, '/Users/moose1/Documents/photoApp/Photos/2.png');\n"

const dbClear = "DROP TABLE album_permissions;\n" +
	"DROP TABLE photos;\n" +
	"DROP TABLE albums;\n" +
	"DROP TABLE users;\n"
*/

// these functions are to be used with a database that includes following tables (! = primary key):
// users: id!|email		albums: id!|userid|name	     photos: id!|album_id|user_id|path		album_permissions: album_id|user_id	tags: photo_id|tagged_id

func check(e error) {
	if e != nil {
		panic(e)
	}
}

type syncDB struct {
	Db *sql.DB
	Mu sync.Mutex
}

// create a new user along with an initial album
func newUser(email string, tx *sql.Tx) error {
	r, err := tx.Exec("insert into users (email) values (?)", email)
	if err != nil {
		return fmt.Errorf("failed to add users: %w", err)
	}
	userID, err := r.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get new user id: %w", err)
	}
	mainAlbum := fmt.Sprintf("%s's Photos", email)
	r, err = tx.Exec("insert into albums (user_id, name) values (?, ?)", userID, mainAlbum)
	if err != nil {
		return fmt.Errorf("failed to create user album: %w", err)
	}
	albumID, err := r.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get album id: %w", err)
	}
	err = givePerm(albumID, userID, tx)
	if err != nil {
		return fmt.Errorf("failed to give user permission to main album: %w", err)
	}
	return nil
}

func newAlbum(name string, userID int64, tx *sql.Tx) error {
	r, err := tx.Exec("insert into albums (name, user_id) values (?, ?)", name, userID)
	if err != nil {
		return fmt.Errorf("failed to create album: %w", err)
	}
	albumID, err := r.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get album id: %w", err)
	}
	err = givePerm(albumID, userID, tx)
	if err != nil {
		return fmt.Errorf("failed to give user permission to album: %w", err)
	}
	return nil
}

// checks if the given user has permission to access the given album
func checkPerm(albumID int64, userID int64, tx *sql.Tx) bool {
	//retrieve all albums that a user has access to
	permittedAlbumRows, err := tx.Query("select album_id from album_permissions where user_id = ? and album_id = ?", userID, albumID)
	defer permittedAlbumRows.Close()
	if err != nil {
		log.Printf("failed to access album_permissions: %s", err)
	}
	// copy all album ids that the specified user has access to into a slice
	return permittedAlbumRows.Next()
}

// add a photo to a specified album if the calling user has permission according to the album_permissions table
func addPhoto(albumID int64, userID int64, db *syncDB) (int64, string, error) {
	var photoID int64
	var path string
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tx, err := db.Db.BeginTx(ctx, nil)
	if err != nil {
		return 0, "", fmt.Errorf("failed to begin transaction: %w", err)
	}
	if checkPerm(albumID, userID, tx) == true {
		db.Mu.Lock()
		defer db.Mu.Unlock()
		res, err := tx.Exec("INSERT INTO photos (user_id, album_id) VALUES (?, ?)", userID, albumID)
		if err != nil {
			return 0, "", fmt.Errorf("failed to insert photo: %w", err)
		}
		photoID, err = res.LastInsertId()
		check(err)
		path = "/Users/moose1/Documents/photoApp/Photos/" + strconv.FormatInt(photoID, 10) //TODO: get image format
		_, err = tx.Exec("UPDATE photos SET path = ? WHERE id = ?", path, photoID)
		if err != nil {
			return 0, "", fmt.Errorf("failed to add path to photo table: %w", err)
		}
	} else {
		return 0, "", fmt.Errorf("user doesn't have permission to access album")
	}
	err = tx.Commit()
	if err != nil {
		return 0, "", fmt.Errorf("failed to commit transaction: %w", err)
	}
	return photoID, path, nil
	//add a tag feature to this function?
}

// give a user permission to view and add photos to an album
func givePerm(albumID int64, userID int64, tx *sql.Tx) error {
	if checkPerm(albumID, userID, tx) == false {
		_, err := tx.Exec("insert into album_permissions (album_id, user_id) values (?, ?)", albumID, userID)
		if err != nil {
			return fmt.Errorf("failed to give permission: %w", err)
		}
	} else {
		return fmt.Errorf("That user already has permission to access the album!\n")
	}
	return nil
}

func showTags(userID int64, db *syncDB) ([]int64, []int64) {
	db.Mu.Lock()
	taggedPhotoRows, err := db.Db.Query("SELECT id FROM photos JOIN tags ON photos.id = tags.photo_id WHERE tagged_user_id = ?", userID)
	defer taggedPhotoRows.Close()
	check(err)
	db.Mu.Unlock()
	taggedPhotos := make([]int64, 0)
	for i := 0; taggedPhotoRows.Next(); i++ {
		var newElem int64
		taggedPhotos = append(taggedPhotos, newElem)
		err := taggedPhotoRows.Scan(&taggedPhotos[i])
		check(err)
	}
	db.Mu.Lock()
	taggedAlbumRows, err := db.Db.Query("SELECT album_id FROM photos JOIN tags ON photos.id = tags.photo_id WHERE tagged_user_id = ?", userID)
	defer taggedAlbumRows.Close()
	check(err)
	db.Mu.Unlock()
	taggedAlbums := make([]int64, 0)
	for i := 0; taggedAlbumRows.Next(); i++ {
		var newElem int64
		taggedAlbums = append(taggedAlbums, newElem)
		err := taggedAlbumRows.Scan(&taggedAlbums[i])
		check(err)
	}
	return taggedPhotos, taggedAlbums
}

var templates = template.Must(template.ParseFiles("templates/home.html", "templates/album.html", "templates/photo.html"))

type page interface {
	query() string
	render(w http.ResponseWriter, r *http.Request, rows *sql.Rows)
}

type homepage struct {
	UserID int64
	Albums []int64
}

type albumpage struct {
	AlbumID int64
	Photos  []int64
}

type photopage struct {
	AlbumID int64
	PhotoID int64
	Path    string
}

func (h homepage) query() string {
	//TODO: get user_id from http request header
	return "SELECT id FROM albums WHERE user_id = " + strconv.FormatInt(h.UserID, 10)
}

func (a albumpage) query() string {
	return "SELECT id FROM photos WHERE album_id = " + strconv.FormatInt(a.AlbumID, 10)
}

func (p photopage) render(w http.ResponseWriter) error {
	return templates.ExecuteTemplate(w, "photo.html", p)
}

// TODO: handle errors instead of panicking
func (h homepage) render(w http.ResponseWriter, r *http.Request, rows *sql.Rows) error {
	albums := make([]int64, 0)
	for i := 0; rows.Next(); i++ {
		var newElem int64
		albums = append(albums, newElem)
		err := rows.Scan(&albums[i])
		if err != nil {
			return err
		}
	}
	h.Albums = albums

	return templates.ExecuteTemplate(w, "home.html", h)
}

func (a albumpage) render(w http.ResponseWriter, r *http.Request, rows *sql.Rows) error {
	photos := make([]int64, 0)
	for i := 0; rows.Next(); i++ {
		var newElem int64
		photos = append(photos, newElem)
		err := rows.Scan(&photos[i])
		if err != nil {
			return err
		}
	}
	a.Photos = photos
	fmt.Printf("album photos: %v\n", a.Photos)
	return templates.ExecuteTemplate(w, "album.html", a)
}

func homeHandler(w http.ResponseWriter, r *http.Request, db *syncDB) {
	h := homepage{}
	m := validPath.FindStringSubmatch(r.URL.Path)
	id, err := strconv.ParseInt(m[2], 10, 64)
	check(err)
	h.UserID = id

	rows, err := db.Db.Query(h.query())
	defer rows.Close()
	if err != nil {
		log.Panic("ERROR: invalid user id\n")
	}
	err = h.render(w, r, rows)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func albumHandler(w http.ResponseWriter, r *http.Request, db *syncDB) {
	a := albumpage{}
	m := validPath.FindStringSubmatch(r.URL.Path)
	id, err := strconv.ParseInt(m[2], 10, 64)
	check(err)
	a.AlbumID = id
	fmt.Printf("album query: %s \n", a.query())
	db.Mu.Lock()
	rows, err := db.Db.Query(a.query())
	check(err)
	db.Mu.Unlock()
	defer rows.Close()

	/*
		if err != nil {
			log.Fatalf("ERROR: album query\n%e\n", err)
		}
	*/
	err = a.render(w, r, rows)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

var validPath = regexp.MustCompile("^/(home|album|photo|photos|upload)/([a-zA-Z0-9]+)$")

// serves HTML
func photoHandler(w http.ResponseWriter, r *http.Request, db *syncDB) {
	p := photopage{}
	m := validPath.FindStringSubmatch(r.URL.Path)

	var err error
	id, err := strconv.ParseInt(m[2], 10, 64)
	check(err)
	p.PhotoID = id

	err = p.render(w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// serves images /photos/1 -> /Users/moose1/Documents/photoApp/Photos/1.jpg
func photosHandler(w http.ResponseWriter, r *http.Request, db *syncDB) {
	db.Mu.Lock()
	rows, err := db.Db.Query("SELECT path FROM photos WHERE album_id = 1")
	check(err)
	db.Mu.Unlock()
	var result string
	if rows.Next() {
		err = rows.Scan(&result)
		check(err)
	}
	fmt.Printf(">>>>>>>>>>>>>>>>> query result: %v\n", result)
	fmt.Printf("Photos path request: % +v\n", r.URL.Path)
	m := validPath.FindStringSubmatch(r.URL.Path)
	id, err := strconv.ParseInt(m[2], 10, 64)
	check(err)
	var path string
	db.Mu.Lock()
	err = db.Db.QueryRow("SELECT path FROM photos WHERE id = ?", id).Scan(&path)
	check(err)
	db.Mu.Unlock()
	/*
		if err != nil {
			log.Fatalf("ERROR: path query\n %e\n", err)
		}
	*/
	f, err := os.Open(path)
	check(err)
	_, err = io.Copy(w, f)
	check(err)
}

func uploadHandler(w http.ResponseWriter, r *http.Request, db *syncDB) {
	err := r.ParseMultipartForm(1000000)
	check(err)
	imageInput := r.MultipartForm.File
	fh := imageInput["photo"]
	if len(fh) < 1 {
		log.Fatalf("ERR: no file uploaded")
	} else if len(fh) > 1 {
		log.Fatalf("ERR: too many files uploaded\n")
	}
	fmt.Printf("uploaded file size: %v\n", fh[0].Size)
	mpf, err := fh[0].Open()
	defer mpf.Close()
	check(err)

	m := validPath.FindStringSubmatch(r.URL.Path)
	albumID, err := strconv.ParseInt(m[2], 10, 64)
	check(err)
	//TODO: Get userID from site token or cookie

	photoID, path, err := addPhoto(albumID, 1, db)
	if err != nil {
		log.Printf("failed to add photo to database: %s", err)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	f, err := os.Create(path)
	if err != nil {
		log.Printf("failed to create file with path %s: %s", path, err)
		http.Redirect(w, r, "/album/"+strconv.FormatInt(albumID, 10), http.StatusInternalServerError)
		return
	}
	_, err = io.Copy(f, mpf)
	if err != nil {
		log.Printf("failed to copy data from multipart file to photo file: %s", err)
		http.Redirect(w, r, "/album/"+strconv.FormatInt(albumID, 10), http.StatusInternalServerError)
		return
	}
	info, err := f.Stat()
	if err != nil {
		log.Printf("failed to get information about the created file: %s", err)
		http.Redirect(w, r, "/album/"+strconv.FormatInt(albumID, 10), http.StatusInternalServerError)
		return
	}
	fmt.Printf("copied file size: %v\n", info.Size())
	http.Redirect(w, r, "/photo/"+strconv.FormatInt(photoID, 10), http.StatusFound)
}

func makeHandler(fn func(http.ResponseWriter, *http.Request, *syncDB), db *syncDB) http.HandlerFunc {
	log.Printf("DB: %p", db.Db)
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("DB: %p", db.Db)
		fn(w, r, db)
	}
}

func main() {
	fmt.Println("working")
	log.SetFlags(log.Lshortfile)
	dbPath := "/Users/moose1/Documents/photoApp/photoAppDB"
	diskDB, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Printf("failed to open database: %s\n", dbPath)
	}
	defer diskDB.Close()
	db := &syncDB{Db: diskDB}

	/*
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		go func() {
			for sig := range c {
				fmt.Printf("---Signal: %s\n", sig)
				db.Mu.Lock()
				_, err := db.Db.Exec(dbClear)
				db.Mu.Unlock()
				if err != nil {
					log.Printf("failed to clear database: %s\n", err)
				}
				os.Exit(1)
			}
		}()
		func() {
			db.Mu.Lock()
			defer db.Mu.Unlock()
			_, err = db.Db.Exec(dbInit)
			if err != nil {
				log.Print("failed to initialize database: %s")
				panic(err)
			}
		}()
		defer func() {
			fmt.Println("Clearing should be working!")
			db.Mu.Lock()
			defer db.Mu.Unlock()
			_, err = db.Db.Exec(dbClear)
			log.Printf("failed to clear database: %s", err)
		}()
	*/

	http.HandleFunc("/home/", makeHandler(homeHandler, db))
	http.HandleFunc("/album/", makeHandler(albumHandler, db))
	http.HandleFunc("/photo/", makeHandler(photoHandler, db))
	http.HandleFunc("/photos/", makeHandler(photosHandler, db))
	http.HandleFunc("/upload/", makeHandler(uploadHandler, db)) //TODO: change upload path and regexp parser

	log.Println(http.ListenAndServe(":8080", nil))
}
