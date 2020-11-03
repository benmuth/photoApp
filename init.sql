CREATE TABLE users (id INTEGER PRIMARY KEY, email TEXT UNIQUE);
CREATE TABLE albums (id INTEGER PRIMARY KEY, user_id INTEGER REFERENCES users(id), name TEXT NOT NULL);
CREATE TABLE photos (id INTEGER PRIMARY KEY, album_id INTEGER REFERENCES albums(id), user_id INTEGER REFERENCES users(id), path TEXT UNIQUE);
CREATE TABLE album_permissions (album_id INTEGER REFERENCES albums(id), user_id INTEGER REFERENCES users(id));
INSERT INTO users (email) VALUES ('user1@example.com');
INSERT INTO users (email) VALUES ('user2@example.com');
INSERT INTO albums (user_id, name) VALUES (1, '1 main');
INSERT INTO albums (user_id, name) VALUES (2, '2 main');
INSERT INTO albums (user_id, name) VALUES (1, '1s Birthday!');
INSERT INTO album_permissions (album_id, user_id) VALUES (1,1);
INSERT INTO album_permissions (album_id, user_id) VALUES (1,2);
INSERT INTO album_permissions (album_id, user_id) VALUES (2,2);
INSERT INTO photos (album_id, user_id, path) VALUES (1, 1, '/Users/moose1/Documents/photoApp/Photos/1.jpg');
INSERT INTO photos (album_id, user_id, path) VALUES (1, 1, '/Users/moose1/Documents/photoApp/Photos/2.png');

