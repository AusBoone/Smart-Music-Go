package main

// Integration tests spin up the full HTTP server with an in-memory database and
// exercise a typical flow: login callback, search, and save a favorite. These
// tests use httptest to avoid network dependencies.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"Smart-Music-Go/pkg/db"
	"Smart-Music-Go/pkg/handlers"
	"Smart-Music-Go/pkg/music"
	libspotify "github.com/zmb3/spotify"
)

type stubService struct{ tracks []music.Track }

func (s stubService) SearchTrack(context.Context, string) ([]music.Track, error) {
	return s.tracks, nil
}
func (s stubService) GetRecommendations(context.Context, []string) ([]music.Track, error) {
	return s.tracks, nil
}

// TestIntegrationSearchFavorite exercises the /api/search and /favorites routes
// end-to-end with a real database.
func TestIntegrationSearchFavorite(t *testing.T) {
	svc := stubService{tracks: []music.Track{{SimpleTrack: libspotify.SimpleTrack{ID: "1", Name: "Song", Artists: []libspotify.SimpleArtist{{Name: "Artist"}}}}}}
	auth := libspotify.NewAuthenticator("http://example.com/cb")
	auth.SetAuthInfo("id", "secret")

	database, err := db.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	app := &handlers.Application{Music: svc, Authenticator: auth, DB: database, SignKey: testKey}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/search", app.SearchJSON)
	mux.HandleFunc("/favorites", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			app.AddFavorite(w, r)
		}
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/api/search?track=song", nil)
	res, err := http.DefaultClient.Do(req)
	if err != nil || res.StatusCode != http.StatusOK {
		t.Fatalf("search failed %v %d", err, res.StatusCode)
	}
	var tracks []music.Track
	json.NewDecoder(res.Body).Decode(&tracks)
	if len(tracks) != 1 {
		t.Fatalf("expected 1 track, got %d", len(tracks))
	}

	favReqBody := `{"track_id":"1","track_name":"Song","artist_name":"Artist"}`
	favReq, _ := http.NewRequest(http.MethodPost, srv.URL+"/favorites", strings.NewReader(favReqBody))
	favReq.AddCookie(&http.Cookie{Name: "spotify_user_id", Value: sign("user")})
	favReq.AddCookie(&http.Cookie{Name: "csrf_token", Value: "t"})
	favReq.Header.Set("X-CSRF-Token", "t")
	resp, err := http.DefaultClient.Do(favReq)
	if err != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("favorite add failed %v %d", err, resp.StatusCode)
	}

	favs, err := database.ListFavorites(context.Background(), "user")
	if err != nil || len(favs) != 1 {
		t.Fatalf("favorite not stored: %v %v", err, favs)
	}
}
