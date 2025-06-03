package handlers_test

import (
	"bytes"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"Smart-Music-Go/pkg/handlers"
	stub "github.com/zmb3/spotify"
)

func TestHomeHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	app := &handlers.Application{}
	app.Home(rr, req)
	body := rr.Body.String()
	if !strings.Contains(body, "<form") {
		t.Errorf("expected form in response")
	}
}

func TestRenderSearchResultsTemplate(t *testing.T) {
	tmpl := template.Must(template.New("test").Parse(
		`<h1>Search Results</h1><h2>{{.Name}}</h2><p>By: {{(index .Artists 0).Name}}</p><p><a href="{{.ExternalURLs.Spotify}}">Listen on Spotify</a></p>`))

	track := stub.FullTrack{Name: "Track"}
	track.Artists = []stub.SimpleArtist{{Name: "Artist"}}
	track.ExternalURLs.Spotify = "http://example.com"

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, track); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "Track") || !strings.Contains(out, "Artist") || !strings.Contains(out, "http://example.com") {
		t.Errorf("unexpected output: %s", out)
	}
}
