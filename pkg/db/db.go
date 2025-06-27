// Package db provides the persistence layer used by the application. It wraps
// a SQLite database and exposes helper methods for storing OAuth tokens and
// user favorites. The package is intentionally small to keep the example
// simple; callers are expected to open a single DB instance using New and reuse
// it for all operations.

package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/oauth2"
)

// DB wraps a sql.DB connection and exposes helper methods for the
// application's persistence layer.
type DB struct {
	*sql.DB
}

// New opens the SQLite database located at path. If the file does not
// exist it is created along with the required schema. The returned DB
// value wraps the sql.DB connection for use by the rest of the
// application.
func New(path string) (*DB, error) {
	// Open or create the SQLite database file.
	d, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS tokens (user_id TEXT PRIMARY KEY, token TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS favorites (id INTEGER PRIMARY KEY AUTOINCREMENT, user_id TEXT, track_id TEXT, track_name TEXT, artist_name TEXT)`,
		`CREATE TABLE IF NOT EXISTS history (id INTEGER PRIMARY KEY AUTOINCREMENT, user_id TEXT, track_id TEXT, artist_name TEXT, played_at TIMESTAMP)`,
		`CREATE TABLE IF NOT EXISTS collections (id TEXT PRIMARY KEY, owner TEXT)`,
		`CREATE TABLE IF NOT EXISTS collection_tracks (collection_id TEXT, track_id TEXT, track_name TEXT, artist_name TEXT)`,
		`CREATE TABLE IF NOT EXISTS collection_users (collection_id TEXT, user_id TEXT)`,
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
	// Upsert the token so subsequent logins replace any existing value.
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
	// Insert the favorite using the supplied track metadata. The ID is
	// auto-incremented so we only store the user association and track
	// details.
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
// Results are returned in reverse insertion order so the most recently
// saved tracks appear first.
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

// AddHistory inserts a listening event for the given user. playedAt should be
// the time the track was played and is stored as a timestamp.
func (db *DB) AddHistory(userID, trackID, artistName string, playedAt time.Time) error {
	_, err := db.Exec(`INSERT INTO history(user_id, track_id, artist_name, played_at) VALUES(?,?,?,?)`, userID, trackID, artistName, playedAt)
	return err
}

// ArtistCount represents how many times an artist was played.
type ArtistCount struct {
	Artist string
	Count  int
}

// TopArtistsSince returns the most played artists since the provided time.
func (db *DB) TopArtistsSince(userID string, since time.Time) ([]ArtistCount, error) {
	rows, err := db.Query(`SELECT artist_name, COUNT(*) c FROM history WHERE user_id=? AND played_at>=? GROUP BY artist_name ORDER BY c DESC`, userID, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var res []ArtistCount
	for rows.Next() {
		var ac ArtistCount
		if err := rows.Scan(&ac.Artist, &ac.Count); err != nil {
			return nil, err
		}
		res = append(res, ac)
	}
	return res, rows.Err()
}
