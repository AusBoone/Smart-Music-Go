package applemusic

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// roundTripper mocks HTTP responses for tests.
type roundTripper struct {
	status int
	body   string
}

func (rt roundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	rec := httptest.NewRecorder()
	if rt.body != "" {
		rec.WriteString(rt.body)
	}
	rec.WriteHeader(rt.status)
	return rec.Result(), nil
}

// TestSearchTrackSuccess ensures JSON is parsed and converted correctly.
func TestSearchTrackSuccess(t *testing.T) {
	data := `{"results":[{"trackId":1,"trackName":"Song","artistName":"Artist"}]}`
	c := &Client{HTTP: &http.Client{Transport: roundTripper{status: 200, body: data}}}
	res, err := c.SearchTrack(context.Background(), "q")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res) != 1 || res[0].Name != "Song" {
		t.Fatalf("unexpected result %+v", res)
	}
}

// TestSearchTrackStatusError verifies non-200 responses return an error.
func TestSearchTrackStatusError(t *testing.T) {
	c := &Client{HTTP: &http.Client{Transport: roundTripper{status: 500}}}
	if _, err := c.SearchTrack(context.Background(), "q"); err == nil {
		t.Fatal("expected error")
	}
}
