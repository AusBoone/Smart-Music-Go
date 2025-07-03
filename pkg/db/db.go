// Package db provides the persistence layer used by the application. It wraps
// a SQLite database and exposes helper methods for storing OAuth tokens and
// user favorites. The package is intentionally small to keep the example
// simple; callers are expected to open a single DB instance using New and reuse
// it for all operations. Recent updates add collaborative playlist helpers,
// monthly listening summaries, favorite deletion helpers and automatic
// de-duplication of favorites.

package db

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
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
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_fav_user_track ON favorites(user_id, track_id)`,
		`CREATE TABLE IF NOT EXISTS history (id INTEGER PRIMARY KEY AUTOINCREMENT, user_id TEXT, track_id TEXT, artist_name TEXT, played_at TIMESTAMP)`,
		`CREATE TABLE IF NOT EXISTS collections (id TEXT PRIMARY KEY, owner TEXT)`,
		`CREATE TABLE IF NOT EXISTS collection_tracks (collection_id TEXT, track_id TEXT, track_name TEXT, artist_name TEXT)`,
		`CREATE TABLE IF NOT EXISTS collection_users (collection_id TEXT, user_id TEXT)`,
		`CREATE TABLE IF NOT EXISTS shares (id TEXT PRIMARY KEY, track_id TEXT, track_name TEXT, artist_name TEXT)`,
		`CREATE TABLE IF NOT EXISTS playlist_shares (id TEXT PRIMARY KEY, playlist_id TEXT, playlist_name TEXT)`,
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
func (db *DB) SaveToken(ctx context.Context, userID string, token *oauth2.Token) error {
	// Serialize the oauth2 token to JSON before storing it.
	b, err := json.Marshal(token)
	if err != nil {
		return err
	}
	// Upsert the token so subsequent logins replace any existing value.
	_, err = db.ExecContext(ctx, `INSERT INTO tokens(user_id, token) VALUES(?, ?) ON CONFLICT(user_id) DO UPDATE SET token=excluded.token`, userID, string(b))
	return err
}

// GetToken retrieves the OAuth token stored for userID and unmarshals it
// from JSON.  The returned token includes the refresh token if one was
// originally saved.
func (db *DB) GetToken(ctx context.Context, userID string) (*oauth2.Token, error) {
	var data string
	if err := db.QueryRowContext(ctx, `SELECT token FROM tokens WHERE user_id=?`, userID).Scan(&data); err != nil {
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
// Spotify track information being saved. Duplicate entries for the
// same user and track are ignored so favorites remain unique.
func (db *DB) AddFavorite(ctx context.Context, userID, trackID, trackName, artistName string) error {
	// Insert the favorite using the supplied track metadata. The ID is
	// auto-incremented so we only store the user association and track
	// details.
	_, err := db.ExecContext(ctx, `INSERT OR IGNORE INTO favorites(user_id, track_id, track_name, artist_name) VALUES(?, ?, ?, ?)`, userID, trackID, trackName, artistName)
	return err
}

// DeleteFavorite removes a track from the user's favorites list. sql.ErrNoRows
// is returned when the specified favorite does not exist which allows callers
// to respond with a 404.
func (db *DB) DeleteFavorite(ctx context.Context, userID, trackID string) error {
	res, err := db.ExecContext(ctx, `DELETE FROM favorites WHERE user_id=? AND track_id=?`, userID, trackID)
	if err != nil {
		return err
	}
	if n, err := res.RowsAffected(); err == nil && n == 0 {
		return sql.ErrNoRows
	}
	return nil
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
func (db *DB) ListFavorites(ctx context.Context, userID string) ([]Favorite, error) {
	// Query all favorites for the given user ordered by insertion time.
	rows, err := db.QueryContext(ctx, `SELECT track_id, track_name, artist_name FROM favorites WHERE user_id=? ORDER BY id DESC`, userID)
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
func (db *DB) AddHistory(ctx context.Context, userID, trackID, artistName string, playedAt time.Time) error {
	_, err := db.ExecContext(ctx, `INSERT INTO history(user_id, track_id, artist_name, played_at) VALUES(?,?,?,?)`, userID, trackID, artistName, playedAt)
	return err
}

// ArtistCount represents how many times an artist was played.
type ArtistCount struct {
	Artist string
	Count  int
}

// TopArtistsSince returns the most played artists since the provided time.
func (db *DB) TopArtistsSince(ctx context.Context, userID string, since time.Time) ([]ArtistCount, error) {
	rows, err := db.QueryContext(ctx, `SELECT artist_name, COUNT(*) c FROM history WHERE user_id=? AND played_at>=? GROUP BY artist_name ORDER BY c DESC`, userID, since)
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

// ShareTrack holds information about a track shared with a unique link.
type ShareTrack struct {
	TrackID    string
	TrackName  string
	ArtistName string
}

// SharePlaylist represents a playlist that can be accessed via a short ID.
type SharePlaylist struct {
	PlaylistID   string
	PlaylistName string
}

// randomString returns a URL-safe base64 string with n bytes of entropy. It is
// used for generating non-guessable IDs.
func randomString(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// CreateShareTrack generates a unique ID and stores the track metadata so users
// can share it via a short URL. The ID is returned to the caller for link
// construction.
func (db *DB) CreateShareTrack(ctx context.Context, trackID, trackName, artistName string) (string, error) {
	id, err := randomString(12)
	if err != nil {
		return "", err
	}
	_, err = db.ExecContext(ctx, `INSERT INTO shares(id, track_id, track_name, artist_name) VALUES(?,?,?,?)`, id, trackID, trackName, artistName)
	if err != nil {
		return "", err
	}
	return id, nil
}

// CreateSharePlaylist stores the playlist details under a random ID so it can
// be shared via URL. The generated ID is returned to the caller.
func (db *DB) CreateSharePlaylist(ctx context.Context, playlistID, playlistName string) (string, error) {
	id, err := randomString(12)
	if err != nil {
		return "", err
	}
	_, err = db.ExecContext(ctx, `INSERT INTO playlist_shares(id, playlist_id, playlist_name) VALUES(?,?,?)`, id, playlistID, playlistName)
	if err != nil {
		return "", err
	}
	return id, nil
}

// GetSharePlaylist retrieves the playlist metadata for a share ID.
func (db *DB) GetSharePlaylist(ctx context.Context, id string) (SharePlaylist, error) {
	var sp SharePlaylist
	err := db.QueryRowContext(ctx, `SELECT playlist_id, playlist_name FROM playlist_shares WHERE id=?`, id).Scan(&sp.PlaylistID, &sp.PlaylistName)
	if err != nil {
		return SharePlaylist{}, err
	}
	return sp, nil
}

// GetShareTrack looks up the track referenced by a share ID. sql.ErrNoRows is
// returned if the ID does not exist.
func (db *DB) GetShareTrack(ctx context.Context, id string) (ShareTrack, error) {
	var st ShareTrack
	err := db.QueryRowContext(ctx, `SELECT track_id, track_name, artist_name FROM shares WHERE id=?`, id).Scan(&st.TrackID, &st.TrackName, &st.ArtistName)
	if err != nil {
		return ShareTrack{}, err
	}
	return st, nil
}

// TopTracksSince returns the most played tracks since the given time.
func (db *DB) TopTracksSince(ctx context.Context, userID string, since time.Time) ([]TrackCount, error) {
	rows, err := db.QueryContext(ctx, `SELECT track_id, COUNT(*) c FROM history WHERE user_id=? AND played_at>=? GROUP BY track_id ORDER BY c DESC`, userID, since)
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
func (db *DB) CreateCollection(ctx context.Context, owner string) (string, error) {
	id := fmt.Sprintf("c_%d", time.Now().UnixNano())
	if _, err := db.ExecContext(ctx, `INSERT INTO collections(id, owner) VALUES(?, ?)`, id, owner); err != nil {
		return "", err
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO collection_users(collection_id, user_id) VALUES(?, ?)`, id, owner); err != nil {
		return "", err
	}
	return id, nil
}

// AddTrackToCollection saves a track in the specified collection.
func (db *DB) AddTrackToCollection(ctx context.Context, colID, trackID, trackName, artistName string) error {
	_, err := db.ExecContext(ctx, `INSERT INTO collection_tracks(collection_id, track_id, track_name, artist_name) VALUES(?,?,?,?)`, colID, trackID, trackName, artistName)
	return err
}

// CollectionTrack represents a track entry within a collection.
type CollectionTrack struct {
	TrackID    string
	TrackName  string
	ArtistName string
}

// ListCollectionTracks returns all tracks stored in the given collection.
func (db *DB) ListCollectionTracks(ctx context.Context, colID string) ([]CollectionTrack, error) {
	rows, err := db.QueryContext(ctx, `SELECT track_id, track_name, artist_name FROM collection_tracks WHERE collection_id=?`, colID)
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
func (db *DB) AddUserToCollection(ctx context.Context, colID, userID string) error {
	_, err := db.ExecContext(ctx, `INSERT INTO collection_users(collection_id, user_id) SELECT ?, ? WHERE NOT EXISTS (SELECT 1 FROM collection_users WHERE collection_id=? AND user_id=?)`, colID, userID, colID, userID)
	return err
}

// MonthCount groups play count totals by month in YYYY-MM format.
type MonthCount struct {
	Month string
	Count int
}

// MonthlyPlayCountsSince aggregates listening history per month starting from
// the provided time. Results are ordered chronologically.
func (db *DB) MonthlyPlayCountsSince(ctx context.Context, userID string, since time.Time) ([]MonthCount, error) {
	rows, err := db.QueryContext(ctx, `SELECT strftime('%Y-%m', played_at) m, COUNT(*) c FROM history WHERE user_id=? AND played_at>=? GROUP BY m ORDER BY m`, userID, since)
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
