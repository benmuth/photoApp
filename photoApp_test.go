package main

import (
	"database/sql"
	"testing"
)

const dbInit = "CREATE TABLE users (id integer primary key, email text unique);\n" +
	"CREATE TABLE albums (id integer primary key, user_id integer references users(id), name text not null);\n" +
	"CREATE TABLE photos (id integer primary key, album_id integer references albums(id), user_id integer references users(id));\n" +
	"CREATE TABLE album_permissions (album_id integer references albums(id), user_id integer references users(id), unique (album_id, user_id));\n" +
	"CREATE TABLE tags (photo_id references photos(id), tagged_id references users(id), unique (photo_id, tagged_id));\n" +
	"INSERT INTO users (email) VALUES ('user1@example.com');\n" +
	"INSERT INTO users (email) VALUES ('user2@example.com');\n" +
	"INSERT INTO albums (user_id, name) VALUES (1, '1 main');\n" +
	"INSERT INTO albums (user_id, name) VALUES (2, '2 main');\n" +
	"INSERT INTO albums (user_id, name) VALUES (1, '1s Birthday!');\n" +
	"INSERT INTO photos (album_id, user_id) VALUES (1, 1);\n" +
	"INSERT INTO photos (album_id, user_id) VALUES (2, 2);\n" +
	"INSERT INTO photos (album_id, user_id) VALUES (3, 1);\n" +
	"INSERT INTO album_permissions (album_id, user_id) VALUES (1, 1);\n" +
	"INSERT INTO album_permissions (album_id, user_id) VALUES (2, 2);\n" +
	"INSERT INTO album_permissions (album_id, user_id) VALUES (3, 2);\n" +
	"INSERT INTO tags (photo_id, tagged_id) VALUES (3, 2);\n"

func TestPerm(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	check(err)
	defer db.Close()

	_, err = db.Exec(dbInit)
	check(err)

	examples := []struct {
		name    string
		albumID int64
		userID  int64
		want    bool
	}{
		{
			name:    "perm",
			albumID: 1,
			userID:  1,
			want:    true,
		},
		{
			name:    "noPerm",
			albumID: 1,
			userID:  2,
			want:    false,
		},
	}

	for _, ex := range examples {
		t.Run(ex.name, func(t *testing.T) {
			got := checkPerm(ex.albumID, ex.userID, db)
			if got != ex.want {
				t.Fatalf("got %v, want %v\n", got, ex.want)
			}
		})
	}
}

func TestTags(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	check(err)
	defer db.Close()

	_, err = db.Exec(init)
	check(err)

	examples := []struct {
		name   string
		userID int64
		want   []int64
	}{
		{
			name:   "one tag",
			userID: 2,
			want:   []int64{3},
		},
	}

	for _, ex := range examples {
		t.Run(ex.name, func(t *testing.T) {
			got := showTaggedPhotos(ex.userID, db)
			for i := range got {
				if got[i] != ex.want[i] {
					t.Fatalf("got %v, want %v\n", got[i], ex.want[i])
				}
			}

		})
	}
}
