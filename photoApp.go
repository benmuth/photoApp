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
	"flag"
	"fmt"
	"html/template"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

// these functions are to be used with a database that includes following tables (! = primary key):
// users: id!|email|password	albums: id!|userid|name	     photos: id!|album_id|user_id|path		album_permissions: album_id|user_id	tags: photo_id|tagged_id
// sessions: user_id|session_id
// create a new user along with an initial album
func newUser(email string, password string, tx *sql.Tx) (int64, error) {
	passwordBytes := []byte(password)
	hashedPassword, err := bcrypt.GenerateFromPassword(passwordBytes, 1)
	if err != nil {
		return 0, fmt.Errorf("failed to hash password: %w", err)
	}
	r, err := tx.Exec("INSERT INTO users (email, password) VALUES (?, ?)", email, hashedPassword)
	if err != nil {
		return 0, fmt.Errorf("failed to add user: %w", err)
	}
	userID, err := r.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get new user id: %w", err)
	}
	mainAlbum := fmt.Sprintf("%s's Photos", email)
	r, err = tx.Exec("insert into albums (user_id, name) values (?, ?)", userID, mainAlbum)
	if err != nil {
		return 0, fmt.Errorf("failed to create user album: %w", err)
	}
	albumID, err := r.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get album id: %w", err)
	}
	err = givePerm(albumID, userID, tx)
	if err != nil {
		return 0, fmt.Errorf("failed to give user permission to main album: %w", err)
	}
	return userID, nil
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
func addPhoto(albumID int64, userID int64, db *sql.DB) (int64, string, error) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, "", fmt.Errorf("failed to begin transaction: %w", err)
	}
	//if checkPerm(albumID, userID, tx) == true {
	res, err := tx.Exec("INSERT INTO photos (user_id, album_id) VALUES (?, ?)", userID, albumID)
	if err != nil {
		return 0, "", fmt.Errorf("failed to insert photo: %w", err)
	}

	photoID, err := res.LastInsertId()
	if err != nil {
		return 0, "", fmt.Errorf("failed to get photoID: %w", err)
	}

	dir := os.Getenv("SILSILA_PHOTO_PATH")

	photoPath := path.Join(dir, strconv.FormatInt(photoID, 10)) //TODO: get image format
	_, err = tx.Exec("UPDATE photos SET path = ? WHERE id = ?", photoPath, photoID)
	if err != nil {
		return 0, "", fmt.Errorf("failed to add path to photo table: %w", err)
	}
	//} else {
	//	return 0, "", fmt.Errorf("user doesn't have permission to access album")
	//}
	err = tx.Commit()
	if err != nil {
		return 0, "", fmt.Errorf("failed to commit transaction: %w", err)
	}
	return photoID, photoPath, nil
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

func showTags(userID int64, db *sql.DB) ([]int64, []int64, error) {
	taggedPhotoRows, err := db.Query("SELECT id FROM photos JOIN tags ON photos.id = tags.photo_id WHERE tagged_user_id = ?", userID)
	defer taggedPhotoRows.Close()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to access photo tags: %w", err)
	}
	taggedPhotos := make([]int64, 0)
	for i := 0; taggedPhotoRows.Next(); i++ {
		var newElem int64
		taggedPhotos = append(taggedPhotos, newElem)
		err := taggedPhotoRows.Scan(&taggedPhotos[i])
		if err != nil {
			return nil, nil, fmt.Errorf("failed to scan tagged photo ids: %w", err)

		}
	}
	taggedAlbumRows, err := db.Query("SELECT album_id FROM photos JOIN tags ON photos.id = tags.photo_id WHERE tagged_user_id = ?", userID)
	defer taggedAlbumRows.Close()
	if err != nil {
		return taggedPhotos, nil, fmt.Errorf("failed to access tagged albums: %w", err)
	}
	taggedAlbums := make([]int64, 0)
	for i := 0; taggedAlbumRows.Next(); i++ {
		var newElem int64
		taggedAlbums = append(taggedAlbums, newElem)
		err := taggedAlbumRows.Scan(&taggedAlbums[i])
		if err != nil {
			return taggedPhotos, nil, fmt.Errorf("failed to scan tagged album")
		}
	}
	return taggedPhotos, taggedAlbums, nil
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func randString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Int63()%int64(len(letterBytes))]
	}
	return string(b)
}

// TODO: move begin transaction code to separate function
func checkSesh(w http.ResponseWriter, r *http.Request, db *sql.DB) (int64, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}

	cookie, err := r.Cookie("session_cookie")
	if err != nil {
		http.Redirect(w, r, "/login/", http.StatusFound)
		return 0, fmt.Errorf("failed to get cookie from request: %w", err)
	}
	row := tx.QueryRow("SELECT user_id FROM sessions WHERE session_id = ?", cookie.Value)
	var userID int64
	err = row.Scan(&userID)
	if err != nil {
		http.Redirect(w, r, "/login/", http.StatusFound)
		return 0, fmt.Errorf("failed to scan query result: %w", err)
	}
	if err := tx.Commit(); err != nil {
		log.Printf("%s", err)
	}
	return userID, nil
}

var templates = template.Must(template.ParseFiles("templates/home.html", "templates/album.html", "templates/photo.html", "templates/login.html", "templates/register.html"))

type page interface {
	query() string
	render(w http.ResponseWriter, r *http.Request, rows *sql.Rows)
}

type homepage struct {
	UserID int64
	Albums []int64
}

type albumpage struct {
	UserID  int64
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

func registerHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("failed to begin transaction: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(r.URL.Query()) > 0 {
		email := r.FormValue("email")
		log.Printf("entered email: %s", email)

		password := r.FormValue("password")
		log.Printf("entered password: %s", password)
		id, err := newUser(email, password, tx)
		if err != nil {
			log.Printf("failed to add user: %s", err)
			http.Redirect(w, r, "/register/", http.StatusInternalServerError)
			return
		} else {
			sessionID := randString(10)
			_, err = tx.Exec("INSERT INTO sessions (user_id, session_id) VALUES (?, ?)", id, sessionID)
			if err != nil {
				log.Printf("failed to insert session id into database: %s", err)
				http.Redirect(w, r, "/login/", http.StatusInternalServerError)
			}
			cookie := http.Cookie{
				Name:  "session_cookie",
				Value: sessionID,
				Path:  "/",
			}
			http.SetCookie(w, &cookie)
			http.Redirect(w, r, "/home/"+strconv.FormatInt(id, 10), http.StatusFound)
		}
	} else {
		if err := templates.ExecuteTemplate(w, "register.html", homepage{}); err != nil {
			log.Printf("failed to execute register template: %s", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	if err := tx.Commit(); err != nil {
		log.Printf("%s", err)
	}
}

func loginHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("failed to begin transaction: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(r.URL.Query()) > 0 { // is there a better way to check if user credentials were input?
		if r.URL.Query().Get("logout") == "yes" {
			cookie, err := r.Cookie("session_cookie")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			_, err = tx.Exec("DELETE FROM sessions WHERE session_id = ?", cookie.Value)
			if err != nil {
				log.Printf("failed to delete session id on logout")
			}
			http.Redirect(w, r, "/login/", http.StatusFound)
			return
		}
		email := r.FormValue("email")
		log.Printf("entered email: %s", email)
		row := tx.QueryRow("SELECT id FROM users WHERE email = ?", email)
		var id int64
		err = row.Scan(&id)
		if err != nil {
			log.Printf("failed to scan row: %s", err)
			http.Redirect(w, r, "/login/", http.StatusFound)
			return
		}

		inputPassword := r.FormValue("password")
		log.Printf("entered password: %s", inputPassword)
		inputPasswordBytes := []byte(inputPassword)
		/*hashedPassword, err := bcrypt.GenerateFromPassword(inputPassword, 1) // what does minimum cost argument mean?
		if err != nil {
			log.Printf("failed to hash password: %s", err)
			http.Redirect(w, r, "/login/", http.StatusUnauthorized)
			return
		}
		*/
		row = tx.QueryRow("SELECT password FROM users WHERE email = ?", email)
		var storedPassword string
		if err = row.Scan(&storedPassword); err != nil {
			log.Printf("failed to retrieve user password: %s", err)
			http.Redirect(w, r, "/login/", http.StatusUnauthorized)
			return
		}
		storedPasswordBytes := []byte(storedPassword)
		if err = bcrypt.CompareHashAndPassword(storedPasswordBytes, inputPasswordBytes); err != nil {
			log.Printf("user input incorrect password: %s", err)
			log.Printf("input password: %s | stored password: %s", inputPasswordBytes, storedPasswordBytes)
			http.Redirect(w, r, "/login/", http.StatusUnauthorized)
			return
		} else {
			sessionID := randString(10)
			_, err = tx.Exec("INSERT INTO sessions (user_id, session_id) VALUES (?, ?)", id, sessionID)
			if err != nil {
				log.Printf("failed to insert session id into database: %s", err)
				http.Redirect(w, r, "/login/", http.StatusInternalServerError)
			}
			cookie := http.Cookie{
				Name:  "session_cookie",
				Value: sessionID,
				Path:  "/",
			}
			http.SetCookie(w, &cookie)
			http.Redirect(w, r, "/home/"+strconv.FormatInt(id, 10), http.StatusFound)
		}
	} else { // if there is no query, send to login page
		if err := templates.ExecuteTemplate(w, "login.html", homepage{}); err != nil {
			log.Printf("failed to execute login template: %s", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	if err := tx.Commit(); err != nil {
		log.Printf("%s", err)
	}
}

func homeHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	_, err := checkSesh(w, r, db)
	if err != nil {
		log.Printf("failed to validate user session: %s", err)
	}

	h := homepage{}

	id := path.Base(r.URL.Path)
	h.UserID, err = strconv.ParseInt(path.Base(r.URL.Path), 10, 64)
	if err != nil {
		log.Printf("failed to convert user id string to int: %s", err)
		http.Redirect(w, r, "/login/", http.StatusFound)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("failed to begin transaction: %s", err)
		return
	}

	if len(r.URL.Query()) > 0 {

		//if the form has been filled in, create a new album with the entered name
		albumName := r.FormValue("album name")
		if albumName == "" {
			log.Printf("user didn't input album name, no album created")
			http.Redirect(w, r, path.Join("/home/", id), http.StatusFound)
		} else {
			res, err := tx.Exec("INSERT INTO albums (user_id, name) VALUES (?, ?)", h.UserID, albumName)
			if err != nil {
				log.Printf("failed to create new album: %s", err)
				http.Redirect(w, r, path.Join("/home/", id), http.StatusFound)
			}
			albumID, err := res.LastInsertId()
			if err != nil {
				log.Printf("failed to get id of new album: %s", err)
			}
			log.Printf("album %s with id %v created", albumName, albumID)
			http.Redirect(w, r, path.Join("/album/", strconv.FormatInt(albumID, 10)), http.StatusFound)
		}
	}

	rows, err := tx.Query(h.query())
	defer rows.Close()
	if err != nil {
		log.Printf("failed to query database for user albums: %s", err)
	}
	err = h.render(w, r, rows)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := tx.Commit(); err != nil {
		log.Printf("%s", err)
	}
}

func albumHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if _, err := checkSesh(w, r, db); err != nil {
		log.Printf("failed to validate user session: %s", err)
	}

	a := albumpage{}

	id, err := strconv.ParseInt(path.Base(r.URL.Path), 10, 64)
	if err != nil {
		log.Printf("failed to convert album id string to int: %s", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("failed to begin transaction: %s", err)
		return
	}

	a.AlbumID = id

	fmt.Printf("album query: %s \n", a.query())
	photoRows, err := tx.Query(a.query())
	defer photoRows.Close()

	userRow := tx.QueryRow("SELECT user_id FROM albums WHERE id = ?", id)
	var userID int64
	userRow.Scan(&userID)
	a.UserID = userID

	if err != nil {
		log.Printf("failed to query user photos: %s", err)
		return
	}
	/*
		if err != nil {
			log.Fatalf("ERROR: album query\n%e\n", err)
		}
	*/
	err = a.render(w, r, photoRows)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	if err := tx.Commit(); err != nil {
		log.Printf("%s", err)
	}
}

func deleteAlbumHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if _, err := checkSesh(w, r, db); err != nil {
		log.Printf("failed to validate user session: %s", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("failed to begin transaction: %s", err)
		return
	}

	albumID := path.Base(r.URL.Path)
	var userID string
	if err = tx.QueryRow("SELECT user_id FROM albums WHERE id = ?", albumID).Scan(&userID); err != nil {
		log.Printf("failed to select user id from database: %s", err)
		http.Redirect(w, r, path.Join("/home/", userID), http.StatusFound)
	}

	//check if last element of path is a number
	if _, err = strconv.Atoi(albumID); err != nil {
		log.Printf("failed to get album id to be deleted: %s", err)
		http.Redirect(w, r, path.Join("/home/", userID), http.StatusFound)
		return
	}

	if _, err = tx.Exec("DELETE FROM albums WHERE id = ?", albumID); err != nil {
		log.Printf("failed to delete album %s from database: %s", albumID, err)
		http.Redirect(w, r, path.Join("/home/", userID), http.StatusFound)
		return
	}

	if _, err = tx.Exec("DELETE FROM photos WHERE album_id = ?", albumID); err != nil {
		log.Printf("failed to delete photos from album %s from database: %s", albumID, err)
		http.Redirect(w, r, path.Join("/home/", userID), http.StatusFound)
		return
	}

	log.Printf("Deleted album %s and its photos", albumID)

	if err := tx.Commit(); err != nil {
		log.Printf("%s", err)
	}
	http.Redirect(w, r, path.Join("/home/", userID), http.StatusFound)
}

// serves HTML
func photoHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	_, err := checkSesh(w, r, db)
	if err != nil {
		log.Printf("failed to validate user session: %s", err)
	}

	p := photopage{}

	id, err := strconv.ParseInt(path.Base(r.URL.Path), 10, 64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	p.PhotoID = id

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("failed to begin transaction: %s", err)
		return
	}
	row := tx.QueryRow("SELECT album_id FROM photos WHERE id = ?", id)
	var albumID int64
	if err := row.Scan(&albumID); err != nil {
		log.Printf("failed to scan albumID: %s", err)
	}
	p.AlbumID = albumID
	err = p.render(w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// serves images /photos/1 -> /Users/moose1/Documents/photoApp/Photos/1.jpg
func photosHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	_, err := checkSesh(w, r, db)
	if err != nil {
		log.Printf("failed to validate user session: %s", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("failed to begin transaction: %s", err)
		return
	}
	/*
		rows, err := tx.Query("SELECT path FROM photos WHERE album_id = 1") //TODO: replace album id with variable
		defer rows.Close()                                                  //forgot this close!!
		if err != nil {
			log.Printf("failed to query database for photo path: %s", err)
			return
		}
		var result string
		if rows.Next() {
			err = rows.Scan(&result)
			if err != nil {
				log.Printf("failed to scan path query result: %s", err)
				return
			}
		}
		fmt.Printf(">>>>>>>>>>>>>>>>> query result: %v\n", result)
	*/
	fmt.Printf("Photos path request: % +v\n", r.URL.Path)

	id, err := strconv.ParseInt(path.Base(r.URL.Path), 10, 64)
	if err != nil {
		log.Printf("failed to convert id string to int: %s", err)
		return
	}
	var path string
	err = tx.QueryRow("SELECT path FROM photos WHERE id = ?", id).Scan(&path)
	if err != nil {
		log.Printf("failed to get photo path: %s", err)
		return
	}
	/*
		if err != nil {
			log.Fatalf("ERROR: path query\n %e\n", err)
		}
	*/
	f, err := os.Open(path)
	if err != nil {
		log.Printf("failed to open photo: %s", err)
		return
	}
	_, err = io.Copy(w, f)
	if err != nil {
		log.Printf("failed to copy photo to response writer: %s", err)
		return
	}
	if err := tx.Commit(); err != nil {
		log.Printf("%s", err)
	}
}

//TODO: move checkPerm call from addPhoto to uploadHandler
func uploadHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	_, err := checkSesh(w, r, db)
	if err != nil {
		log.Printf("failed to validate user session: %s", err)
	}

	err = r.ParseMultipartForm(1000000)
	if err != nil {
		log.Printf("failed to parse multipart form: %w", err)
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
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
	if err != nil {
		log.Printf("failed to open multipart file: %w", err)
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}
	albumID, err := strconv.ParseInt(path.Base(r.URL.Path), 10, 64)
	if err != nil {
		log.Printf("failed to convert albumID string to int")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

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

func deletePhotoHandler(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if _, err := checkSesh(w, r, db); err != nil {
		log.Printf("failed to validate user session: %s", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		log.Printf("failed to begin transaction: %s", err)
		return
	}

	photoID := path.Base(r.URL.Path)

	//get albumID of photo to use for redirect destination in case of error
	var albumID string
	if err = tx.QueryRow("SELECT album_id FROM photos WHERE id = ?", photoID).Scan(&albumID); err != nil {
		log.Printf("failed to select album id from database: %s", err)
		http.Redirect(w, r, "/home/", http.StatusFound) //TODO: redirect to specific homepage of user
	}

	//check if last element of path is a number
	if _, err = strconv.Atoi(photoID); err != nil {
		log.Printf("failed to get photo id to be deleted: %s", err)
		http.Redirect(w, r, path.Join("/album/", albumID), http.StatusFound)
		return
	}

	if _, err = tx.Exec("DELETE FROM photos WHERE id = ?", photoID); err != nil {
		log.Printf("failed to delete photo %s from database: %s", photoID, err)
		http.Redirect(w, r, path.Join("/album/", albumID), http.StatusFound)
		return
	}

	if err = os.Remove(filepath.Join(os.Getenv("SILSILA_PHOTO_PATH"), photoID)); err != nil {
		log.Printf("failed to delete photo %s from system: %s", photoID, err)
		http.Redirect(w, r, path.Join("/album/", albumID), http.StatusFound)
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("%s", err)
	}
	http.Redirect(w, r, path.Join("/album/", albumID), http.StatusFound)
}

func makeHandler(fn func(http.ResponseWriter, *http.Request, *sql.DB), db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		header := w.Header()
		header["Cache-control"] = []string{"no-cache", "no-store", "must-revalidate"}
		header["Pragma"] = []string{"no-cache"}
		header["Expires"] = []string{"0"}
		fn(w, r, db)
	}
}

func main() {
	port := flag.Int("port", 8080, "designate port to bind to.")
	dbPath := flag.String("db", "/Users/ben/Documents/photoApp/photoAppDB", "designate database path to use")
	flag.Parse()
	pathenv := "SILSILA_PHOTO_PATH"
	_, ok := os.LookupEnv(pathenv)
	if !ok {
		log.Printf("ERR: Photo upload path not set. %s", pathenv)
		return
	}
	log.SetFlags(log.Lshortfile)
	log.Println("started...")
	f, err := os.Open(*dbPath)
	if err != nil {
		log.Printf("failed to open database: %s", err)
		return
	}
	_, err = f.Stat()
	if err != nil {
		log.Printf("failed to get info about database file: %s", err)
		return
	}
	db, err := sql.Open("sqlite3", *dbPath)
	if err != nil {
		log.Printf("failed to open database: %s\n", dbPath)
	}
	defer db.Close()
	http.HandleFunc("/login/", makeHandler(loginHandler, db))
	http.HandleFunc("/home/", makeHandler(homeHandler, db))
	http.HandleFunc("/album/", makeHandler(albumHandler, db))
	http.HandleFunc("/photo/", makeHandler(photoHandler, db))
	http.HandleFunc("/photos/", makeHandler(photosHandler, db))
	http.HandleFunc("/upload/", makeHandler(uploadHandler, db)) //TODO: change upload path
	http.HandleFunc("/register/", makeHandler(registerHandler, db))
	http.HandleFunc("/photo/delete/", makeHandler(deletePhotoHandler, db))
	http.HandleFunc("/album/delete/", makeHandler(deleteAlbumHandler, db))

	log.Println(http.ListenAndServe(fmt.Sprintf(":%v", *port), nil))
}
