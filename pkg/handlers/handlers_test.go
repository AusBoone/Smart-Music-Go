package handlers

import (
	"context"
	"crypto/tls"
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

// fakeSpotify implements the minimal subset of the Spotify client used by
// RecommendationsAdvanced. It allows injecting predefined results without
// hitting the real API during tests.
type fakeSpotify struct {
	tracks []music.Track
	err    error
}

func (f fakeSpotify) GetAudioFeatures(ids ...string) ([]*libspotify.AudioFeatures, error) {
	return nil, nil
}

var testKey = []byte("test-key")

// addCSRF attaches a fixed CSRF token header and cookie to the request so
// handlers requiring the token succeed in tests.
func addCSRF(req *http.Request) {
	token := "token"
	req.AddCookie(&http.Cookie{Name: "csrf_token", Value: token})
	req.Header.Set("X-CSRF-Token", token)
}

func (f fakeSearcher) SearchTrack(ctx context.Context, track string) ([]music.Track, error) {
	return f.tracks, f.err
}

func (f fakeSearcher) GetRecommendations(ctx context.Context, seedIDs []string) ([]music.Track, error) {
	return f.tracks, f.err
}

func (f fakeSpotify) GetRecommendationsWithAttrs(ctx context.Context, seedIDs []string, attrs *libspotify.TrackAttributes) ([]music.Track, error) {
	return f.tracks, f.err
}

func TestMain(m *testing.M) {
	// change working directory to project root so template paths resolve
	if err := os.Chdir("../.."); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

// TestEncodeTokenAttributes ensures cookies created by encodeToken include the
// expected security attributes so the browser enforces them correctly.
func TestEncodeTokenAttributes(t *testing.T) {
	app := &Application{SignKey: testKey}
	tok := &oauth2.Token{AccessToken: "a"}
	c := app.encodeToken(tok, true)
	if !c.Secure {
		t.Errorf("Secure not set")
	}
	if c.SameSite != http.SameSiteLaxMode {
		t.Errorf("SameSite not Lax")
	}
}

// TestLoginCookieAttributes verifies the oauth_state cookie has secure options
// when starting the OAuth flow.
func TestLoginCookieAttributes(t *testing.T) {
	app := &Application{Authenticator: libspotify.NewAuthenticator("http://example.com"), SignKey: testKey}
	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	rr := httptest.NewRecorder()
	app.Login(rr, req)
	found := false
	for _, c := range rr.Result().Cookies() {
		if c.Name == "oauth_state" {
			found = true
			if c.SameSite != http.SameSiteLaxMode {
				t.Errorf("oauth_state cookie SameSite not set")
			}
		}
	}
	if !found {
		t.Errorf("oauth_state cookie not set")
	}
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
	if err := d.AddFavorite(context.Background(), "u", "1", "Song", "Artist"); err != nil {
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

	// Valid cookie but missing CSRF token should fail with 403.
	req = httptest.NewRequest(http.MethodPost, "/favorites", strings.NewReader(`{}`))
	req.AddCookie(&http.Cookie{Name: "spotify_user_id", Value: signValue("u", testKey)})
	rr = httptest.NewRecorder()
	app.AddFavorite(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 for missing csrf, got %d", rr.Code)
	}
}

// TestAddFavoriteValidation verifies that missing fields return a 400 error.
func TestAddFavoriteValidation(t *testing.T) {
	d, err := db.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	app := &Application{DB: d, SignKey: testKey}
	req := httptest.NewRequest(http.MethodPost, "/favorites", strings.NewReader(`{}`))
	req.AddCookie(&http.Cookie{Name: "spotify_user_id", Value: signValue("u", testKey)})
	addCSRF(req)
	rr := httptest.NewRecorder()
	app.AddFavorite(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", rr.Code)
	}
}

// TestDeleteFavorite verifies that a stored favorite can be removed through the
// API and that deleting a non-existent favorite returns 404.
func TestDeleteFavorite(t *testing.T) {
	d, err := db.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	d.AddFavorite(context.Background(), "u", "1", "Song", "Artist")
	app := &Application{DB: d, SignKey: testKey}
	req := httptest.NewRequest(http.MethodDelete, "/api/favorites", strings.NewReader(`{"track_id":"1"}`))
	req.AddCookie(&http.Cookie{Name: "spotify_user_id", Value: signValue("u", testKey)})
	addCSRF(req)
	rr := httptest.NewRecorder()
	app.DeleteFavorite(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204 got %d", rr.Code)
	}
	// verify gone
	favs, _ := d.ListFavorites(context.Background(), "u")
	if len(favs) != 0 {
		t.Fatalf("favorite not removed: %+v", favs)
	}

	req = httptest.NewRequest(http.MethodDelete, "/api/favorites", strings.NewReader(`{"track_id":"x"}`))
	req.AddCookie(&http.Cookie{Name: "spotify_user_id", Value: signValue("u", testKey)})
	addCSRF(req)
	rr = httptest.NewRecorder()
	app.DeleteFavorite(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 got %d", rr.Code)
	}
}

// TestFavoritesCSV verifies that favorites can be exported as CSV.
func TestFavoritesCSV(t *testing.T) {
	d, err := db.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	d.AddFavorite(context.Background(), "u", "1", "Song", "Artist")
	app := &Application{DB: d, SignKey: testKey}
	req := httptest.NewRequest(http.MethodGet, "/api/favorites.csv", nil)
	req.AddCookie(&http.Cookie{Name: "spotify_user_id", Value: signValue("u", testKey)})
	rr := httptest.NewRecorder()
	app.FavoritesCSV(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "track_id,track_name,artist_name") {
		t.Fatalf("unexpected csv output: %s", body)
	}
}

// TestGoogleLoginCookie ensures the Google login handler sets a signed state cookie.
func TestGoogleLoginCookie(t *testing.T) {
	app := &Application{GoogleOAuth: &oauth2.Config{ClientID: "id", ClientSecret: "secret", RedirectURL: "http://example.com"}, SignKey: testKey}
	req := httptest.NewRequest(http.MethodGet, "/login/google", nil)
	rr := httptest.NewRecorder()
	app.GoogleLogin(rr, req)
	found := false
	for _, c := range rr.Result().Cookies() {
		if c.Name == "google_state" {
			found = true
			if c.SameSite != http.SameSiteLaxMode {
				t.Errorf("google_state cookie SameSite not set")
			}
		}
	}
	if !found {
		t.Errorf("google_state cookie not set")
	}
}

// TestGoogleCallbackCookie verifies that GoogleCallback stores a secure cookie
// after successful authentication.
func TestGoogleCallbackCookie(t *testing.T) {
	// Fake OAuth2 server responding with a token.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/token" {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"access_token":"t","token_type":"Bearer"}`)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	cfg := &oauth2.Config{
		ClientID:     "id",
		ClientSecret: "secret",
		Endpoint: oauth2.Endpoint{
			AuthURL:  ts.URL + "/auth",
			TokenURL: ts.URL + "/token",
		},
		RedirectURL: "http://example.com",
	}

	// HTTP client intercepts the userinfo request.
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host == "www.googleapis.com" {
			res := `{"id":"gid"}`
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(res)), Header: http.Header{"Content-Type": {"application/json"}}}, nil
		}
		return http.DefaultTransport.RoundTrip(r)
	})}

	app := &Application{GoogleOAuth: cfg, SignKey: testKey}

	state := "abc"
	req := httptest.NewRequest(http.MethodGet, "/google/callback?state="+state+"&code=x", nil)
	req.AddCookie(&http.Cookie{Name: "google_state", Value: signValue(state, testKey)})
	req = req.WithContext(context.WithValue(req.Context(), oauth2.HTTPClient, client))

	rr := httptest.NewRecorder()
	app.GoogleCallback(rr, req)

	found := false
	var csrfFound bool
	for _, c := range rr.Result().Cookies() {
		if c.Name == "google_user_id" {
			found = true
			if !c.HttpOnly || c.Secure != false {
				t.Errorf("cookie flags not set")
			}
		} else if c.Name == "csrf_token" {
			csrfFound = true
		}
	}
	if !found {
		t.Fatalf("google_user_id cookie not set")
	}
	if !csrfFound {
		t.Errorf("csrf_token cookie not set")
	}
}

// TestAddShareTrack verifies a share link is created and returned as JSON.
func TestAddShareTrack(t *testing.T) {
	d, err := db.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	app := &Application{DB: d, SignKey: testKey}
	body := strings.NewReader(`{"track_id":"1","track_name":"Song","artist_name":"Artist"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/share/track", body)
	req.AddCookie(&http.Cookie{Name: "google_user_id", Value: signValue("g", testKey)})
	addCSRF(req)
	rr := httptest.NewRecorder()
	app.AddShareTrack(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d", rr.Code)
	}
	var res map[string]string
	json.NewDecoder(rr.Body).Decode(&res)
	if !strings.Contains(res["url"], "/share/") {
		t.Fatalf("unexpected url %v", res)
	}
}

// TestAddSharePlaylist ensures playlists can be shared via the API.
func TestAddSharePlaylist(t *testing.T) {
	d, err := db.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	app := &Application{DB: d, SignKey: testKey}
	body := strings.NewReader(`{"playlist_id":"pl","playlist_name":"MyList"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/share/playlist", body)
	req.AddCookie(&http.Cookie{Name: "google_user_id", Value: signValue("g", testKey)})
	addCSRF(req)
	rr := httptest.NewRecorder()
	app.AddSharePlaylist(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d", rr.Code)
	}
	var res map[string]string
	json.NewDecoder(rr.Body).Decode(&res)
	if !strings.Contains(res["url"], "/share/playlist/") {
		t.Fatalf("unexpected url %v", res)
	}
}

// TestSecurityHeaders verifies the middleware adds expected headers.
func TestSecurityHeaders(t *testing.T) {
	h := SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.TLS = &tls.ConnectionState{}
	h.ServeHTTP(rr, req)
	if rr.Header().Get("Content-Security-Policy") == "" {
		t.Fatalf("CSP header missing")
	}
	if rr.Header().Get("X-Frame-Options") != "DENY" {
		t.Fatalf("frame header wrong")
	}
	if rr.Header().Get("Strict-Transport-Security") == "" {
		t.Fatalf("HSTS header missing")
	}
}

// TestAddHistoryValidation ensures missing fields cause a 400 response.
func TestAddHistoryValidation(t *testing.T) {
	d, err := db.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	app := &Application{DB: d, SignKey: testKey}
	req := httptest.NewRequest(http.MethodPost, "/api/history", strings.NewReader(`{}`))
	req.AddCookie(&http.Cookie{Name: "spotify_user_id", Value: signValue("u", testKey)})
	addCSRF(req)
	rr := httptest.NewRecorder()
	app.AddHistoryJSON(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", rr.Code)
	}
}

// TestAddCollectionTrackValidation verifies that invalid JSON payloads
// for AddCollectionTrackJSON return a 400 status.
func TestAddCollectionTrackValidation(t *testing.T) {
	d, err := db.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	app := &Application{DB: d, SignKey: testKey}
	req := httptest.NewRequest(http.MethodPost, "/api/collections/abc/tracks", strings.NewReader("{"))
	req.AddCookie(&http.Cookie{Name: "spotify_user_id", Value: signValue("u", testKey)})
	addCSRF(req)
	rr := httptest.NewRecorder()
	app.AddCollectionTrackJSON(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", rr.Code)
	}
}

// TestAddCollectionUserValidation ensures missing fields return 400.
func TestAddCollectionUserValidation(t *testing.T) {
	d, err := db.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	app := &Application{DB: d, SignKey: testKey}
	req := httptest.NewRequest(http.MethodPost, "/api/collections/abc/users", strings.NewReader(`{}`))
	req.AddCookie(&http.Cookie{Name: "spotify_user_id", Value: signValue("u", testKey)})
	addCSRF(req)
	rr := httptest.NewRecorder()
	app.AddCollectionUserJSON(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", rr.Code)
	}
}

type rt struct{ data string }

func (r rt) RoundTrip(req *http.Request) (*http.Response, error) {
	resp := &http.Response{StatusCode: 200, Header: make(http.Header)}
	resp.Body = io.NopCloser(strings.NewReader(r.data))
	return resp, nil
}

// roundTripFunc allows custom HTTP behaviour inside tests without defining a
// full struct type each time.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func setAuthClient(a *libspotify.Authenticator, c *http.Client) {
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, c)
	v := reflect.ValueOf(a).Elem().FieldByName("context")
	reflect.NewAt(v.Type(), unsafe.Pointer(v.UnsafeAddr())).Elem().Set(reflect.ValueOf(ctx))
}

type refreshRT struct {
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
	if err := d.SaveToken(context.Background(), "u", tok); err != nil {
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
	if err := d.SaveToken(context.Background(), "u", expired); err != nil {
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
	tok, err := d.GetToken(context.Background(), "u")
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
	if err := d.SaveToken(context.Background(), "u", expired); err != nil {
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
	tok, _ := d.GetToken(context.Background(), "u")
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
	addCSRF(histReq)
	rr := httptest.NewRecorder()
	app.AddHistoryJSON(rr, histReq)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201 got %d", rr.Code)
	}
	counts, _ := d.TopTracksSince(context.Background(), "u", time.Now().Add(-time.Hour))
	if len(counts) != 1 || counts[0].TrackID != "1" {
		t.Fatalf("history not recorded: %+v", counts)
	}

	// Create a new collection then add a track and a user.
	colReq := httptest.NewRequest(http.MethodPost, "/api/collections", nil)
	colReq.AddCookie(&http.Cookie{Name: "spotify_user_id", Value: signValue("u", testKey)})
	addCSRF(colReq)
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
	addCSRF(trackReq)
	rr = httptest.NewRecorder()
	app.AddCollectionTrackJSON(rr, trackReq)
	if rr.Code != http.StatusCreated {
		t.Fatalf("add track status %d", rr.Code)
	}

	tracks, _ := d.ListCollectionTracks(context.Background(), colID)
	if len(tracks) != 1 || tracks[0].TrackID != "1" {
		t.Fatalf("track not stored: %+v", tracks)
	}

	userReq := httptest.NewRequest(http.MethodPost, "/api/collections/"+colID+"/users", strings.NewReader(`{"user_id":"other"}`))
	userReq.AddCookie(&http.Cookie{Name: "spotify_user_id", Value: signValue("u", testKey)})
	addCSRF(userReq)
	rr = httptest.NewRecorder()
	app.AddCollectionUserJSON(rr, userReq)
	if rr.Code != http.StatusCreated {
		t.Fatalf("add user status %d", rr.Code)
	}
}

// TestInsightsEndpoints verifies that the insights handlers return JSON summaries
// from the database. It covers InsightsMonthlyJSON and InsightsTracksJSON with
// a simple in-memory database populated with history entries.
func TestInsightsEndpoints(t *testing.T) {
	d, err := db.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	// Two history records in different months for aggregation
	d.AddHistory(context.Background(), "u", "1", "Artist", now)
	d.AddHistory(context.Background(), "u", "2", "Artist", now.AddDate(0, 1, 0))
	app := &Application{DB: d, SignKey: testKey}

	// Monthly endpoint
	req := httptest.NewRequest(http.MethodGet, "/api/insights/monthly?since=2024-01-01", nil)
	req.AddCookie(&http.Cookie{Name: "spotify_user_id", Value: signValue("u", testKey)})
	rr := httptest.NewRecorder()
	app.InsightsMonthlyJSON(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("monthly status %d", rr.Code)
	}
	var months []db.MonthCount
	if err := json.NewDecoder(rr.Body).Decode(&months); err != nil {
		t.Fatal(err)
	}
	if len(months) != 2 {
		t.Fatalf("expected 2 months got %d", len(months))
	}

	// Tracks endpoint with custom days parameter
	req = httptest.NewRequest(http.MethodGet, "/api/insights/tracks?days=800", nil)
	req.AddCookie(&http.Cookie{Name: "spotify_user_id", Value: signValue("u", testKey)})
	rr = httptest.NewRecorder()
	app.InsightsTracksJSON(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("tracks status %d", rr.Code)
	}
	var tracks []db.TrackCount
	if err := json.NewDecoder(rr.Body).Decode(&tracks); err != nil {
		t.Fatal(err)
	}
	if len(tracks) != 2 {
		t.Fatalf("expected 2 track counts got %d", len(tracks))
	}
}

// TestRecommendationsAdvanced checks that advanced recommendations honour query
// parameters and return the Spotify client's result.
func TestRecommendationsAdvanced(t *testing.T) {
	fs := fakeSpotify{tracks: []music.Track{{SimpleTrack: libspotify.SimpleTrack{ID: "1"}}}}
	app := &Application{SpotifyClient: fs, SignKey: testKey}
	req := httptest.NewRequest(http.MethodGet, "/api/recommendations/advanced?track_id=1&min_energy=0.5", nil)
	rr := httptest.NewRecorder()
	app.RecommendationsAdvanced(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status %d", rr.Code)
	}
	var res []music.Track
	if err := json.NewDecoder(rr.Body).Decode(&res); err != nil {
		t.Fatal(err)
	}
	if len(res) != 1 || string(res[0].ID) != "1" {
		t.Fatalf("unexpected response %+v", res)
	}
}
