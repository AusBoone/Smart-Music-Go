package main

import (
	"Smart-Music-Go/pkg/db"
	"Smart-Music-Go/pkg/handlers"
	"Smart-Music-Go/pkg/music"
	libspotify "github.com/zmb3/spotify"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// fakeSearcher implements the TrackSearcher interface for tests. It returns
// pre-defined results and errors so handlers can be tested without hitting the
// real Spotify API.
type fakeSearcher struct {
	tracks []music.Track
	err    error
}

func (f fakeSearcher) SearchTrack(track string) ([]music.Track, error) {
	return f.tracks, f.err
}

func (f fakeSearcher) GetRecommendations(seedIDs []string) ([]music.Track, error) {
	return f.tracks, f.err
}

// TestMain changes the working directory so templates resolve correctly when
// tests are run from the package directory.
func TestMain(m *testing.M) {
	os.Chdir("../..")
	os.Exit(m.Run())
}

// newServer creates an HTTP server with all routes registered using in-memory
// dependencies so the endpoints can be exercised in tests.
func newServer() *httptest.Server {
	fs := fakeSearcher{tracks: []music.Track{
		{SimpleTrack: libspotify.SimpleTrack{Name: "Song", Artists: []libspotify.SimpleArtist{{Name: "Artist"}}, ExternalURLs: map[string]string{"spotify": "http://example.com"}}},
	}}
	auth := libspotify.NewAuthenticator("http://example.com/callback")
	auth.SetAuthInfo("id", "secret")
	database, _ := db.New(":memory:")
	app := &handlers.Application{Music: fs, Authenticator: auth, DB: database}
	mux := http.NewServeMux()
	mux.HandleFunc("/", app.Home)
	mux.HandleFunc("/search", app.Search)
	mux.HandleFunc("/recommendations", app.Recommendations)
	mux.HandleFunc("/login", app.Login)
	mux.HandleFunc("/callback", app.OAuthCallback)
	mux.HandleFunc("/playlists", app.Playlists)
	mux.HandleFunc("/favorites", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			app.AddFavorite(w, r)
		} else {
			app.Favorites(w, r)
		}
	})
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./ui/static"))))
	mux.Handle("/app/", http.StripPrefix("/app/", http.FileServer(http.Dir("./ui/frontend/dist"))))
	return httptest.NewServer(mux)
}

// TestSearchEndpoint exercises the HTML search handler and checks that the
// rendered page includes the expected heading.
func TestSearchEndpoint(t *testing.T) {
	srv := newServer()
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/search?track=test")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 got %d", resp.StatusCode)
	}
	data, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(data), "Search Results") {
		t.Errorf("unexpected body %s", data)
	}
}

// TestLoginEndpoint verifies that the login handler redirects the user to the
// Spotify authorization endpoint.
func TestLoginEndpoint(t *testing.T) {
	srv := newServer()
	defer srv.Close()
	client := &http.Client{CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Get(srv.URL + "/login")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusFound {
		t.Fatalf("expected 302 got %d", resp.StatusCode)
	}
	loc := resp.Header.Get("Location")
	if !strings.Contains(loc, "accounts.spotify.com") {
		t.Errorf("unexpected redirect %s", loc)
	}
}

// TestPlaylistsUnauthenticated ensures the playlists page rejects requests that
// have not completed the OAuth flow.
func TestPlaylistsUnauthenticated(t *testing.T) {
	srv := newServer()
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/playlists")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", resp.StatusCode)
	}
}
