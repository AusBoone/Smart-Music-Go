package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
	"unsafe"

	"Smart-Music-Go/pkg/db"
	libspotify "github.com/zmb3/spotify"
	"golang.org/x/oauth2"
)

type fakeSearcher struct {
	tracks []libspotify.FullTrack
	err    error
}

var testKey = []byte("test-key")

func (f fakeSearcher) SearchTrack(track string) ([]libspotify.FullTrack, error) {
	return f.tracks, f.err
}

func TestMain(m *testing.M) {
	// change working directory to project root so template paths resolve
	os.Chdir("../..")
	os.Exit(m.Run())
}

func TestHome(t *testing.T) {
	app := &Application{SignKey: testKey}
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
	app := &Application{Spotify: fakeSearcher{tracks: []libspotify.FullTrack{track}}, SignKey: testKey}

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
	app := &Application{Spotify: fakeSearcher{err: fmt.Errorf("no tracks found")}, SignKey: testKey}
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
	app := &Application{SignKey: testKey}
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
	app := &Application{DB: d, SignKey: testKey}
	req := httptest.NewRequest(http.MethodGet, "/favorites", nil)
	req.AddCookie(&http.Cookie{Name: "spotify_user_id", Value: signValue("u", testKey)})
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
	app := &Application{DB: d, SignKey: testKey}
	req := httptest.NewRequest(http.MethodGet, "/api/favorites", nil)
	req.AddCookie(&http.Cookie{Name: "spotify_user_id", Value: signValue("u", testKey)})
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
	app := &Application{SignKey: testKey}
	req := httptest.NewRequest(http.MethodGet, "/api/playlists", nil)
	rr := httptest.NewRecorder()
	app.PlaylistsJSON(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rr.Code)
	}
}

type rt struct{ data string }

func (r rt) RoundTrip(req *http.Request) (*http.Response, error) {
	resp := &http.Response{StatusCode: 200, Header: make(http.Header)}
	resp.Body = io.NopCloser(strings.NewReader(r.data))
	return resp, nil
}

func setAuthClient(a *libspotify.Authenticator, c *http.Client) {
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, c)
	v := reflect.ValueOf(a).Elem().FieldByName("context")
	reflect.NewAt(v.Type(), unsafe.Pointer(v.UnsafeAddr())).Elem().Set(reflect.ValueOf(ctx))
}

func TestPlaylistsJSONFromDB(t *testing.T) {
	d, err := db.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	tok := &oauth2.Token{AccessToken: "abc", TokenType: "Bearer", Expiry: time.Now().Add(time.Hour)}
	if err := d.SaveToken("u", tok); err != nil {
		t.Fatal(err)
	}
	auth := libspotify.NewAuthenticator("http://example.com/callback")
	auth.SetAuthInfo("id", "secret")
	client := &http.Client{Transport: rt{data: `{"items":[{"id":"1","name":"List","tracks":{"total":0}}]}`}}
	setAuthClient(&auth, client)
	app := &Application{DB: d, Authenticator: auth, SignKey: testKey}
	req := httptest.NewRequest(http.MethodGet, "/api/playlists", nil)
	req.AddCookie(&http.Cookie{Name: "spotify_user_id", Value: signValue("u", testKey)})
	rr := httptest.NewRecorder()
	app.PlaylistsJSON(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	var p libspotify.SimplePlaylistPage
	if err := json.NewDecoder(rr.Body).Decode(&p); err != nil {
		t.Fatal(err)
	}
	if len(p.Playlists) != 1 || p.Playlists[0].Name != "List" {
		t.Fatalf("unexpected playlists: %+v", p.Playlists)
	}
}
