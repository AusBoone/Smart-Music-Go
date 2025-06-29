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
