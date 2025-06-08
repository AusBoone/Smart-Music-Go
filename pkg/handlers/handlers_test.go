package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"Smart-Music-Go/pkg/db"
	libspotify "github.com/zmb3/spotify"
)

type fakeSearcher struct {
	tracks []libspotify.FullTrack
	err    error
}

func (f fakeSearcher) SearchTrack(track string) ([]libspotify.FullTrack, error) {
	return f.tracks, f.err
}

func TestMain(m *testing.M) {
	// change working directory to project root so template paths resolve
	os.Chdir("../..")
	os.Exit(m.Run())
}

func TestHome(t *testing.T) {
	app := &Application{}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	app.Home(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "Welcome to Smart-Music-Go!") {
		t.Errorf("response missing welcome message")
	}
	if !strings.Contains(body, "<form") {
		t.Errorf("response missing form")
	}
}

func TestSearchFound(t *testing.T) {
	track := libspotify.FullTrack{SimpleTrack: libspotify.SimpleTrack{
		Name:         "Song",
		Artists:      []libspotify.SimpleArtist{{Name: "Artist"}},
		ExternalURLs: map[string]string{"spotify": "http://example.com"},
	}}
	app := &Application{Spotify: fakeSearcher{tracks: []libspotify.FullTrack{track}}}

	req := httptest.NewRequest(http.MethodGet, "/api/search?track=test", nil)
	rr := httptest.NewRecorder()

	app.SearchJSON(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	var res []libspotify.FullTrack
	if err := json.NewDecoder(rr.Body).Decode(&res); err != nil {
		t.Fatal(err)
	}
	if len(res) != 1 || res[0].Name != "Song" {
		t.Fatalf("unexpected results: %+v", res)
	}
}

func TestSearchNotFound(t *testing.T) {
	app := &Application{Spotify: fakeSearcher{err: fmt.Errorf("no tracks found")}}
	req := httptest.NewRequest(http.MethodGet, "/api/search?track=missing", nil)
	rr := httptest.NewRecorder()

	app.SearchJSON(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", rr.Code)
	}
}

// failingWriter returns an error on the first Write call to trigger template failures.
type failingWriter struct {
	header http.Header
	status int
	wrote  bool
}

func (w *failingWriter) Header() http.Header {
	if w.header == nil {
		w.header = make(http.Header)
	}
	return w.header
}

func (w *failingWriter) WriteHeader(code int) {
	w.status = code
}

func (w *failingWriter) Write(b []byte) (int, error) {
	if !w.wrote {
		w.wrote = true
		return 0, fmt.Errorf("write error")
	}
	return len(b), nil
}

func TestHomeTemplateError(t *testing.T) {
	app := &Application{}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	fw := &failingWriter{}

	app.Home(fw, req)

	if fw.status != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", fw.status)
	}
}

func TestFavoritesTemplateError(t *testing.T) {
	d, err := db.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	app := &Application{DB: d}
	req := httptest.NewRequest(http.MethodGet, "/favorites", nil)
	req.AddCookie(&http.Cookie{Name: "spotify_user_id", Value: "u"})
	fw := &failingWriter{}

	app.Favorites(fw, req)

	if fw.status != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", fw.status)
	}
}

func TestFavoritesJSON(t *testing.T) {
	d, err := db.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	if err := d.AddFavorite("u", "1", "Song", "Artist"); err != nil {
		t.Fatal(err)
	}
	app := &Application{DB: d}
	req := httptest.NewRequest(http.MethodGet, "/api/favorites", nil)
	req.AddCookie(&http.Cookie{Name: "spotify_user_id", Value: "u"})
	rr := httptest.NewRecorder()
	app.FavoritesJSON(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	var favs []db.Favorite
	if err := json.NewDecoder(rr.Body).Decode(&favs); err != nil {
		t.Fatal(err)
	}
	if len(favs) != 1 || favs[0].TrackID != "1" {
		t.Fatalf("unexpected favorites: %+v", favs)
	}
}

func TestPlaylistsJSONAuth(t *testing.T) {
	app := &Application{}
	req := httptest.NewRequest(http.MethodGet, "/api/playlists", nil)
	rr := httptest.NewRecorder()
	app.PlaylistsJSON(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rr.Code)
	}
}
