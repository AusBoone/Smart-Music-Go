package handlers

import (
	"context"
	"encoding/base64"
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
	"Smart-Music-Go/pkg/music"
	libspotify "github.com/zmb3/spotify"
	"golang.org/x/oauth2"
)

// fakeSearcher is a stand-in for the Spotify client used by handlers. It
// returns predefined tracks and errors to simulate API responses.
type fakeSearcher struct {
	tracks []music.Track
	err    error
}

var testKey = []byte("test-key")

func (f fakeSearcher) SearchTrack(ctx context.Context, track string) ([]music.Track, error) {
	return f.tracks, f.err
}

func (f fakeSearcher) GetRecommendations(ctx context.Context, seedIDs []string) ([]music.Track, error) {
	return f.tracks, f.err
}

func TestMain(m *testing.M) {
	// change working directory to project root so template paths resolve
	os.Chdir("../..")
	os.Exit(m.Run())
}

// TestHome ensures the landing page renders successfully and includes
// basic elements such as the welcome text and search form.
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

// TestSearchFound verifies that a successful track search returns JSON
// with the expected items.
func TestSearchFound(t *testing.T) {
	track := libspotify.FullTrack{SimpleTrack: libspotify.SimpleTrack{
		Name:         "Song",
		Artists:      []libspotify.SimpleArtist{{Name: "Artist"}},
		ExternalURLs: map[string]string{"spotify": "http://example.com"},
	}}
	app := &Application{Music: fakeSearcher{tracks: []music.Track{track}}, SignKey: testKey}

	req := httptest.NewRequest(http.MethodGet, "/api/search?track=test", nil)
	rr := httptest.NewRecorder()

	app.SearchJSON(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	var res []music.Track
	if err := json.NewDecoder(rr.Body).Decode(&res); err != nil {
		t.Fatal(err)
	}
	if len(res) != 1 || res[0].Name != "Song" {
		t.Fatalf("unexpected results: %+v", res)
	}
}

// TestSearchNotFound checks that the API responds with 404 when no tracks match.
func TestSearchNotFound(t *testing.T) {
	app := &Application{Music: fakeSearcher{err: fmt.Errorf("no tracks found")}, SignKey: testKey}
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

// TestHomeTemplateError simulates a template failure and expects a 500
// response to be returned to the client.
func TestHomeTemplateError(t *testing.T) {
	app := &Application{SignKey: testKey}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	fw := &failingWriter{}

	app.Home(fw, req)

	if fw.status != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", fw.status)
	}
}

// TestFavoritesTemplateError verifies that errors rendering the favorites
// page are reported correctly.
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

// TestFavoritesJSON checks that the JSON API returns the stored favorites for a
// user.
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

// TestPlaylistsJSONAuth verifies that unauthenticated requests to the playlists
// API are rejected.
func TestPlaylistsJSONAuth(t *testing.T) {
	app := &Application{SignKey: testKey}
	req := httptest.NewRequest(http.MethodGet, "/api/playlists", nil)
	rr := httptest.NewRecorder()
	app.PlaylistsJSON(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rr.Code)
	}
}

// TestAddFavoriteAuth ensures the AddFavorite endpoint rejects requests with an
// unsigned or missing user cookie. This prevents clients from spoofing other
// user IDs.
func TestAddFavoriteAuth(t *testing.T) {
	app := &Application{SignKey: testKey}

	// Missing cookie should result in 401.
	req := httptest.NewRequest(http.MethodPost, "/favorites", strings.NewReader(`{}`))
	rr := httptest.NewRecorder()
	app.AddFavorite(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rr.Code)
	}

	// Tampered cookie should also be rejected.
	req = httptest.NewRequest(http.MethodPost, "/favorites", strings.NewReader(`{}`))
	req.AddCookie(&http.Cookie{Name: "spotify_user_id", Value: "badvalue"})
	rr = httptest.NewRecorder()
	app.AddFavorite(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401 for bad cookie, got %d", rr.Code)
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

type refreshRT struct {
	t         *testing.T
	tokenResp string
	apiResp   string
}

func (rt refreshRT) RoundTrip(req *http.Request) (*http.Response, error) {
	resp := &http.Response{StatusCode: 200, Header: make(http.Header)}
	if strings.Contains(req.URL.Path, "/api/token") {
		resp.Body = io.NopCloser(strings.NewReader(rt.tokenResp))
	} else {
		resp.Body = io.NopCloser(strings.NewReader(rt.apiResp))
	}
	return resp, nil
}

// TestPlaylistsJSONFromDB ensures a stored token is used when calling the
// playlists endpoint and that the returned JSON is decoded correctly.
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

// TestPlaylistsJSONRefresh checks that an expired token is refreshed when
// calling the playlists API and that the cookie and DB are updated.
func TestPlaylistsJSONRefresh(t *testing.T) {
	d, err := db.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	expired := &oauth2.Token{AccessToken: "old", RefreshToken: "ref", TokenType: "Bearer", Expiry: time.Now().Add(-time.Hour)}
	if err := d.SaveToken("u", expired); err != nil {
		t.Fatal(err)
	}
	auth := libspotify.NewAuthenticator("http://example.com/callback")
	auth.SetAuthInfo("id", "secret")
	tokenJSON := `{"access_token":"new","token_type":"Bearer","expires_in":3600,"refresh_token":"ref"}`
	client := &http.Client{Transport: refreshRT{tokenResp: tokenJSON, apiResp: `{"items":[]}`}}
	setAuthClient(&auth, client)
	app := &Application{DB: d, Authenticator: auth, SignKey: testKey}
	req := httptest.NewRequest(http.MethodGet, "/api/playlists", nil)
	req.AddCookie(&http.Cookie{Name: "spotify_user_id", Value: signValue("u", testKey)})
	rr := httptest.NewRecorder()
	app.PlaylistsJSON(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	tok, err := d.GetToken("u")
	if err != nil || tok.AccessToken != "new" {
		t.Fatalf("token not refreshed: %+v %v", tok, err)
	}
	var found bool
	for _, c := range rr.Result().Cookies() {
		if c.Name == "spotify_token" {
			found = true
			if v, ok := verifyValue(c.Value, testKey); ok {
				t2, err2 := decodeToken(v)
				if err2 != nil || t2.AccessToken != "new" {
					t.Errorf("cookie not updated")
				}
			} else {
				t.Errorf("bad signature")
			}
		}
	}
	if !found {
		t.Errorf("token cookie not set")
	}
}

// TestSearchJSONRefresh performs a search using an expired token and confirms
// that the refresh flow updates the stored token.
func TestSearchJSONRefresh(t *testing.T) {
	d, err := db.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	expired := &oauth2.Token{AccessToken: "old", RefreshToken: "ref", TokenType: "Bearer", Expiry: time.Now().Add(-time.Hour)}
	if err := d.SaveToken("u", expired); err != nil {
		t.Fatal(err)
	}
	fs := fakeSearcher{tracks: []music.Track{{SimpleTrack: libspotify.SimpleTrack{Name: "Song"}}}}
	auth := libspotify.NewAuthenticator("http://example.com/callback")
	auth.SetAuthInfo("id", "secret")
	tokenJSON := `{"access_token":"new","token_type":"Bearer","expires_in":3600,"refresh_token":"ref"}`
	rt := refreshRT{tokenResp: tokenJSON, apiResp: `{"tracks":{"items":[]}}`}
	client := &http.Client{Transport: rt}
	setAuthClient(&auth, client)
	app := &Application{Music: fs, DB: d, Authenticator: auth, SignKey: testKey}
	b, _ := json.Marshal(expired)
	req := httptest.NewRequest(http.MethodGet, "/api/search?track=song", nil)
	req.AddCookie(&http.Cookie{Name: "spotify_token", Value: signValue(base64.StdEncoding.EncodeToString(b), testKey)})
	req.AddCookie(&http.Cookie{Name: "spotify_user_id", Value: signValue("u", testKey)})
	rr := httptest.NewRecorder()
	app.SearchJSON(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", rr.Code)
	}
	tok, _ := d.GetToken("u")
	if tok.AccessToken != "new" {
		t.Errorf("token not refreshed in db")
	}
}

// TestAddHistoryAndCollections covers the collaborative playlist handlers and
// history recording endpoints. It ensures database rows are created and proper
// status codes are returned.
func TestAddHistoryAndCollections(t *testing.T) {
	d, err := db.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	app := &Application{DB: d, SignKey: testKey}

	// Record a play event via the history endpoint.
	histReq := httptest.NewRequest(http.MethodPost, "/api/history", strings.NewReader(`{"track_id":"1","artist_name":"Artist"}`))
	histReq.AddCookie(&http.Cookie{Name: "spotify_user_id", Value: signValue("u", testKey)})
	rr := httptest.NewRecorder()
	app.AddHistoryJSON(rr, histReq)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201 got %d", rr.Code)
	}
	counts, _ := d.TopTracksSince("u", time.Now().Add(-time.Hour))
	if len(counts) != 1 || counts[0].TrackID != "1" {
		t.Fatalf("history not recorded: %+v", counts)
	}

	// Create a new collection then add a track and a user.
	colReq := httptest.NewRequest(http.MethodPost, "/api/collections", nil)
	colReq.AddCookie(&http.Cookie{Name: "spotify_user_id", Value: signValue("u", testKey)})
	rr = httptest.NewRecorder()
	app.CreateCollectionJSON(rr, colReq)
	if rr.Code != http.StatusOK {
		t.Fatalf("create collection status %d", rr.Code)
	}
	var res map[string]string
	json.NewDecoder(rr.Body).Decode(&res)
	colID := res["id"]

	trackReq := httptest.NewRequest(http.MethodPost, "/api/collections/"+colID+"/tracks", strings.NewReader(`{"track_id":"1","track_name":"Song","artist_name":"Artist"}`))
	trackReq.AddCookie(&http.Cookie{Name: "spotify_user_id", Value: signValue("u", testKey)})
	rr = httptest.NewRecorder()
	app.AddCollectionTrackJSON(rr, trackReq)
	if rr.Code != http.StatusCreated {
		t.Fatalf("add track status %d", rr.Code)
	}

	tracks, _ := d.ListCollectionTracks(colID)
	if len(tracks) != 1 || tracks[0].TrackID != "1" {
		t.Fatalf("track not stored: %+v", tracks)
	}

	userReq := httptest.NewRequest(http.MethodPost, "/api/collections/"+colID+"/users", strings.NewReader(`{"user_id":"other"}`))
	userReq.AddCookie(&http.Cookie{Name: "spotify_user_id", Value: signValue("u", testKey)})
	rr = httptest.NewRecorder()
	app.AddCollectionUserJSON(rr, userReq)
	if rr.Code != http.StatusCreated {
		t.Fatalf("add user status %d", rr.Code)
	}
}
