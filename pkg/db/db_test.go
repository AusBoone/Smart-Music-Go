package db

import (
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

	if err := d.AddFavorite("u", "1", "Song", "Artist"); err != nil {
		t.Fatal(err)
	}
	favs, err := d.ListFavorites("u")
	if err != nil {
		t.Fatal(err)
	}
	if len(favs) != 1 || favs[0].TrackID != "1" {
		t.Fatalf("unexpected favorites: %+v", favs)
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
	if err := d.SaveToken("u", tok); err != nil {
		t.Fatal(err)
	}
	got, err := d.GetToken("u")
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
	if err := d.AddHistory("u", "1", "Artist", now); err != nil {
		t.Fatal(err)
	}
	if err := d.AddHistory("u", "2", "Artist", now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	artists, err := d.TopArtistsSince("u", now.Add(-time.Hour))
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
	d.AddHistory("u", "1", "Artist", now)
	d.AddHistory("u", "1", "Artist", now.Add(time.Minute))
	tracks, err := d.TopTracksSince("u", now.Add(-time.Hour))
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
	id, err := d.CreateCollection("u")
	if err != nil || id == "" {
		t.Fatalf("create collection failed: %v", err)
	}
	if err := d.AddTrackToCollection(id, "1", "Song", "Artist"); err != nil {
		t.Fatal(err)
	}
	tracks, err := d.ListCollectionTracks(id)
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
	id, _ := d.CreateCollection("owner")
	if err := d.AddUserToCollection(id, "user2"); err != nil {
		t.Fatal(err)
	}
	// Insert again to verify the IGNORE behaviour does not error
	if err := d.AddUserToCollection(id, "user2"); err != nil {
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

// TestMonthlyPlayCountsSince verifies monthly aggregation of history data.
func TestMonthlyPlayCountsSince(t *testing.T) {
	d, err := New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	base := time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC)
	d.AddHistory("u", "1", "Artist", base)
	d.AddHistory("u", "2", "Artist", base.AddDate(0, 1, 0))
	counts, err := d.MonthlyPlayCountsSince("u", base.AddDate(0, -1, 0))
	if err != nil {
		t.Fatal(err)
	}
	if len(counts) != 2 || counts[0].Count != 1 || counts[1].Count != 1 {
		t.Fatalf("unexpected counts %+v", counts)
	}
}
