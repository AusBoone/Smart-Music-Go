package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	libspotify "github.com/zmb3/spotify"
)

type fakeSearcher struct {
	track libspotify.FullTrack
	err   error
}

func (f fakeSearcher) SearchTrack(track string) (libspotify.FullTrack, error) {
	return f.track, f.err
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
	app := &Application{Spotify: fakeSearcher{track: track}}

	req := httptest.NewRequest(http.MethodGet, "/search?track=test", nil)
	rr := httptest.NewRecorder()

	app.Search(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "Search Results") ||
		!strings.Contains(body, "Song") ||
		!strings.Contains(body, "Artist") {
		t.Errorf("unexpected body: %s", body)
	}
}

func TestSearchNotFound(t *testing.T) {
	app := &Application{Spotify: fakeSearcher{err: fmt.Errorf("no tracks found")}}
	req := httptest.NewRequest(http.MethodGet, "/search?track=missing", nil)
	rr := httptest.NewRecorder()

	app.Search(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "No tracks found for 'missing'") {
		t.Errorf("unexpected response: %s", rr.Body.String())
	}
}
