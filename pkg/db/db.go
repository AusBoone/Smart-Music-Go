package db

import (
	"database/sql"
	"encoding/json"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/oauth2"
)

// DB wraps a sql.DB connection and exposes helper methods for the
// application's persistence layer.
type DB struct {
	*sql.DB
}

// New opens the SQLite database file at path, creates required tables
// on first run and returns a DB value.
func New(path string) (*DB, error) {
	// Open or create the SQLite database file.
	d, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS tokens (user_id TEXT PRIMARY KEY, token TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS favorites (id INTEGER PRIMARY KEY AUTOINCREMENT, user_id TEXT, track_id TEXT, track_name TEXT, artist_name TEXT)`,
	}
	// Execute the schema creation statements. Errors here likely mean the
	// database file is not writable.
	for _, s := range stmts {
		if _, err := d.Exec(s); err != nil {
			d.Close()
			return nil, fmt.Errorf("init db: %w", err)
		}
	}
	return &DB{d}, nil
}

// SaveToken persists the OAuth token for the given userID.  If a token
// already exists it is replaced.
func (db *DB) SaveToken(userID string, token *oauth2.Token) error {
	// Serialize the oauth2 token to JSON before storing it.
	b, err := json.Marshal(token)
	if err != nil {
		return err
	}
	_, err = db.Exec(`INSERT INTO tokens(user_id, token) VALUES(?, ?) ON CONFLICT(user_id) DO UPDATE SET token=excluded.token`, userID, string(b))
	return err
}

// GetToken retrieves the OAuth token stored for userID and unmarshals it
// from JSON.  The returned token includes the refresh token if one was
// originally saved.
func (db *DB) GetToken(userID string) (*oauth2.Token, error) {
	var data string
	if err := db.QueryRow(`SELECT token FROM tokens WHERE user_id=?`, userID).Scan(&data); err != nil {
		return nil, err
	}
	var tok oauth2.Token
	if err := json.Unmarshal([]byte(data), &tok); err != nil {
		return nil, err
	}
	return &tok, nil
}

// AddFavorite inserts a track into the favorites table for userID. The
// trackID, trackName and artistName parameters correspond to the
// Spotify track information being saved.
func (db *DB) AddFavorite(userID, trackID, trackName, artistName string) error {
	_, err := db.Exec(`INSERT INTO favorites(user_id, track_id, track_name, artist_name) VALUES(?, ?, ?, ?)`, userID, trackID, trackName, artistName)
	return err
}

// Favorite represents a track saved by a user.
type Favorite struct {
	TrackID    string
	TrackName  string
	ArtistName string
}

// ListFavorites retrieves all favorites stored for the provided userID.
func (db *DB) ListFavorites(userID string) ([]Favorite, error) {
	// Query all favorites for the given user ordered by insertion time.
	rows, err := db.Query(`SELECT track_id, track_name, artist_name FROM favorites WHERE user_id=? ORDER BY id DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var fs []Favorite
	for rows.Next() {
		var f Favorite
		if err := rows.Scan(&f.TrackID, &f.TrackName, &f.ArtistName); err != nil {
			return nil, err
		}
		fs = append(fs, f)
	}
	// rows.Err returns the first error encountered while iterating.
	return fs, rows.Err()
}
