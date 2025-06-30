// Package db provides the persistence layer used by the application. It wraps
// a SQLite database and exposes helper methods for storing OAuth tokens and
// user favorites. The package is intentionally small to keep the example
// simple; callers are expected to open a single DB instance using New and reuse
// it for all operations. Recent updates add collaborative playlist helpers and
// monthly listening summaries.

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

// TrackCount represents how many times a specific track was played.
type TrackCount struct {
	TrackID string
	Count   int
}

// TopTracksSince returns the most played tracks since the given time.
func (db *DB) TopTracksSince(userID string, since time.Time) ([]TrackCount, error) {
	rows, err := db.Query(`SELECT track_id, COUNT(*) c FROM history WHERE user_id=? AND played_at>=? GROUP BY track_id ORDER BY c DESC`, userID, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var res []TrackCount
	for rows.Next() {
		var tc TrackCount
		if err := rows.Scan(&tc.TrackID, &tc.Count); err != nil {
			return nil, err
		}
		res = append(res, tc)
	}
	return res, rows.Err()
}

// CreateCollection inserts a new collaborative playlist owned by the specified user.
func (db *DB) CreateCollection(owner string) (string, error) {
	id := fmt.Sprintf("c_%d", time.Now().UnixNano())
	if _, err := db.Exec(`INSERT INTO collections(id, owner) VALUES(?, ?)`, id, owner); err != nil {
		return "", err
	}
	if _, err := db.Exec(`INSERT INTO collection_users(collection_id, user_id) VALUES(?, ?)`, id, owner); err != nil {
		return "", err
	}
	return id, nil
}

// AddTrackToCollection saves a track in the specified collection.
func (db *DB) AddTrackToCollection(colID, trackID, trackName, artistName string) error {
	_, err := db.Exec(`INSERT INTO collection_tracks(collection_id, track_id, track_name, artist_name) VALUES(?,?,?,?)`, colID, trackID, trackName, artistName)
	return err
}

// CollectionTrack represents a track entry within a collection.
type CollectionTrack struct {
	TrackID    string
	TrackName  string
	ArtistName string
}

// ListCollectionTracks returns all tracks stored in the given collection.
func (db *DB) ListCollectionTracks(colID string) ([]CollectionTrack, error) {
	rows, err := db.Query(`SELECT track_id, track_name, artist_name FROM collection_tracks WHERE collection_id=?`, colID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var res []CollectionTrack
	for rows.Next() {
		var ct CollectionTrack
		if err := rows.Scan(&ct.TrackID, &ct.TrackName, &ct.ArtistName); err != nil {
			return nil, err
		}
		res = append(res, ct)
	}
	return res, rows.Err()
}

// AddUserToCollection grants access to an existing collection for a user.
// It inserts a row into the collection_users table linking the user
// to the playlist. Duplicate entries are ignored.
func (db *DB) AddUserToCollection(colID, userID string) error {
	_, err := db.Exec(`INSERT INTO collection_users(collection_id, user_id) SELECT ?, ? WHERE NOT EXISTS (SELECT 1 FROM collection_users WHERE collection_id=? AND user_id=?)`, colID, userID, colID, userID)
	return err
}

// MonthCount groups play count totals by month in YYYY-MM format.
type MonthCount struct {
	Month string
	Count int
}

// MonthlyPlayCountsSince aggregates listening history per month starting from
// the provided time. Results are ordered chronologically.
func (db *DB) MonthlyPlayCountsSince(userID string, since time.Time) ([]MonthCount, error) {
	rows, err := db.Query(`SELECT strftime('%Y-%m', played_at) m, COUNT(*) c FROM history WHERE user_id=? AND played_at>=? GROUP BY m ORDER BY m`, userID, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var res []MonthCount
	for rows.Next() {
		var mc MonthCount
		if err := rows.Scan(&mc.Month, &mc.Count); err != nil {
			return nil, err
		}
		res = append(res, mc)
	}
	return res, rows.Err()
}
