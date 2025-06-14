package db

import (
	"os"
	"testing"

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
