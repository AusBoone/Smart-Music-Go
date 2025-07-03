package db

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

// TestAddAndListFavorites verifies that favorites can be persisted and
// subsequently retrieved from the database.
func TestAddAndListFavorites(t *testing.T) {
	d, err := New("test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		d.Close()
		os.Remove("test.db")
	}()

	if err := d.AddFavorite(context.Background(), "u", "1", "Song", "Artist"); err != nil {
		t.Fatal(err)
	}
	favs, err := d.ListFavorites(context.Background(), "u")
	if err != nil {
		t.Fatal(err)
	}
	if len(favs) != 1 || favs[0].TrackID != "1" {
		t.Fatalf("unexpected favorites: %+v", favs)
	}
}

// TestDeleteFavorite verifies that DeleteFavorite removes the record and
// returns sql.ErrNoRows when attempting to delete a missing entry.
func TestDeleteFavorite(t *testing.T) {
	d, err := New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	if err := d.AddFavorite(context.Background(), "u", "1", "Song", "Artist"); err != nil {
		t.Fatal(err)
	}
	if err := d.DeleteFavorite(context.Background(), "u", "1"); err != nil {
		t.Fatalf("delete failed: %v", err)
	}
	favs, _ := d.ListFavorites(context.Background(), "u")
	if len(favs) != 0 {
		t.Fatalf("favorite not removed: %+v", favs)
	}
	if err := d.DeleteFavorite(context.Background(), "u", "missing"); err != sql.ErrNoRows {
		t.Fatalf("expected ErrNoRows, got %v", err)
	}
}

// TestAddFavoriteDedup ensures inserting the same track twice does not create
// duplicate rows thanks to the UNIQUE index and INSERT OR IGNORE.
func TestAddFavoriteDedup(t *testing.T) {
	d, err := New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	if err := d.AddFavorite(context.Background(), "u", "1", "Song", "Artist"); err != nil {
		t.Fatal(err)
	}
	if err := d.AddFavorite(context.Background(), "u", "1", "Song", "Artist"); err != nil {
		t.Fatal(err)
	}
	favs, err := d.ListFavorites(context.Background(), "u")
	if err != nil {
		t.Fatal(err)
	}
	if len(favs) != 1 {
		t.Fatalf("expected one favorite, got %d", len(favs))
	}
}

// TestSaveAndGetToken ensures that OAuth tokens are stored and retrieved
// without modification.
func TestSaveAndGetToken(t *testing.T) {
	d, err := New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	tok := &oauth2.Token{AccessToken: "abc", RefreshToken: "refresh"}
	if err := d.SaveToken(context.Background(), "u", tok); err != nil {
		t.Fatal(err)
	}
	got, err := d.GetToken(context.Background(), "u")
	if err != nil {
		t.Fatal(err)
	}
	if got.AccessToken != tok.AccessToken {
		t.Fatalf("expected %s got %s", tok.AccessToken, got.AccessToken)
	}
	if got.RefreshToken != tok.RefreshToken {
		t.Fatalf("expected refresh %s got %s", tok.RefreshToken, got.RefreshToken)
	}
}

// TestHistory verifies that listening events can be stored and summarized.
func TestHistory(t *testing.T) {
	d, err := New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	now := time.Now()
	if err := d.AddHistory(context.Background(), "u", "1", "Artist", now); err != nil {
		t.Fatal(err)
	}
	if err := d.AddHistory(context.Background(), "u", "2", "Artist", now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	artists, err := d.TopArtistsSince(context.Background(), "u", now.Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(artists) != 1 || artists[0].Artist != "Artist" || artists[0].Count != 2 {
		t.Fatalf("unexpected summary: %+v", artists)
	}
}

// TestTopTracksSince verifies the track summary query returns counts.
func TestTopTracksSince(t *testing.T) {
	d, err := New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	now := time.Now()
	d.AddHistory(context.Background(), "u", "1", "Artist", now)
	d.AddHistory(context.Background(), "u", "1", "Artist", now.Add(time.Minute))
	tracks, err := d.TopTracksSince(context.Background(), "u", now.Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(tracks) != 1 || tracks[0].TrackID != "1" || tracks[0].Count != 2 {
		t.Fatalf("unexpected summary: %+v", tracks)
	}
}

// TestCollections verifies creating a collection and adding tracks.
func TestCollections(t *testing.T) {
	d, err := New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	id, err := d.CreateCollection(context.Background(), "u")
	if err != nil || id == "" {
		t.Fatalf("create collection failed: %v", err)
	}
	if err := d.AddTrackToCollection(context.Background(), id, "1", "Song", "Artist"); err != nil {
		t.Fatal(err)
	}
	tracks, err := d.ListCollectionTracks(context.Background(), id)
	if err != nil || len(tracks) != 1 {
		t.Fatalf("list tracks failed: %v %v", err, tracks)
	}
	if tracks[0].TrackID != "1" {
		t.Fatalf("unexpected track %+v", tracks[0])
	}
}

// TestAddUserToCollection ensures additional users can be associated with a collection.
func TestAddUserToCollection(t *testing.T) {
	d, err := New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	id, _ := d.CreateCollection(context.Background(), "owner")
	if err := d.AddUserToCollection(context.Background(), id, "user2"); err != nil {
		t.Fatal(err)
	}
	// Insert again to verify the IGNORE behaviour does not error
	if err := d.AddUserToCollection(context.Background(), id, "user2"); err != nil {
		t.Fatal(err)
	}
	// Ensure user was added by querying list of users
	rows, err := d.Query(`SELECT COUNT(*) FROM collection_users WHERE collection_id=? AND user_id=?`, id, "user2")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var c int
	if rows.Next() {
		rows.Scan(&c)
	}
	if c != 1 {
		t.Fatalf("expected 1 user row got %d", c)
	}
}

// TestShareTrack verifies that a track can be stored and retrieved using a
// generated share ID.
func TestShareTrack(t *testing.T) {
	d, err := New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	id, err := d.CreateShareTrack(context.Background(), "1", "Song", "Artist")
	if err != nil || len(id) < 10 {
		t.Fatalf("create share failed: %v", err)
	}
	st, err := d.GetShareTrack(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if st.TrackID != "1" || st.TrackName != "Song" {
		t.Fatalf("unexpected share %+v", st)
	}
}

// TestSharePlaylist verifies playlists can be shared and retrieved by ID.
func TestSharePlaylist(t *testing.T) {
	d, err := New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	id, err := d.CreateSharePlaylist(context.Background(), "pl1", "My Mix")
	if err != nil || id == "" {
		t.Fatalf("create share failed: %v", err)
	}
	sp, err := d.GetSharePlaylist(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if sp.PlaylistID != "pl1" || sp.PlaylistName != "My Mix" {
		t.Fatalf("unexpected share %+v", sp)
	}
}

// TestMonthlyPlayCountsSince verifies monthly aggregation of history data.
func TestMonthlyPlayCountsSince(t *testing.T) {
	d, err := New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	base := time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC)
	d.AddHistory(context.Background(), "u", "1", "Artist", base)
	d.AddHistory(context.Background(), "u", "2", "Artist", base.AddDate(0, 1, 0))
	counts, err := d.MonthlyPlayCountsSince(context.Background(), "u", base.AddDate(0, -1, 0))
	if err != nil {
		t.Fatal(err)
	}
	if len(counts) != 2 || counts[0].Count != 1 || counts[1].Count != 1 {
		t.Fatalf("unexpected counts %+v", counts)
	}
}
