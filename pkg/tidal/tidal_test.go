package tidal

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

type rt struct {
	status int
	body   string
}

func (r rt) RoundTrip(*http.Request) (*http.Response, error) {
	rec := httptest.NewRecorder()
	if r.body != "" {
		rec.WriteString(r.body)
	}
	rec.WriteHeader(r.status)
	return rec.Result(), nil
}

// TestSearchTrack verifies JSON decoding and error handling.
func TestSearchTrack(t *testing.T) {
	data := `{"tracks":{"items":[{"id":1,"title":"Song","artist":{"name":"Art"}}]}}`
	c := &Client{Token: "t", HTTP: &http.Client{Transport: rt{status: 200, body: data}}}
	res, err := c.SearchTrack(context.Background(), "q")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res) != 1 || res[0].Name != "Song" {
		t.Fatalf("unexpected result %+v", res)
	}
}

func TestSearchTrackStatus(t *testing.T) {
	c := &Client{Token: "t", HTTP: &http.Client{Transport: rt{status: 500}}}
	if _, err := c.SearchTrack(context.Background(), "q"); err == nil {
		t.Fatal("expected error")
	}
}
