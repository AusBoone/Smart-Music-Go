package youtube

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
	rec.WriteHeader(r.status)
	rec.WriteString(r.body)
	return rec.Result(), nil
}

// TestSearchTrackSuccess verifies JSON is parsed into Track values.
func TestSearchTrackSuccess(t *testing.T) {
	data := `{"items":[{"id":{"videoId":"abc"},"snippet":{"title":"Song","channelTitle":"Artist"}}]}`
	c := &Client{Key: "k", Client: &http.Client{Transport: rt{status: 200, body: data}}}
	tracks, err := c.SearchTrack(context.Background(), "q")
	if err != nil || len(tracks) != 1 || tracks[0].Name != "Song" {
		t.Fatalf("unexpected result: %v %+v", err, tracks)
	}
}

// TestSearchTrackStatusError ensures non-200 responses are returned as errors.
func TestSearchTrackStatusError(t *testing.T) {
	c := &Client{Key: "k", Client: &http.Client{Transport: rt{status: 500}}}
	_, err := c.SearchTrack(context.Background(), "q")
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestGetRecommendations verifies recommendations are parsed correctly.
func TestGetRecommendations(t *testing.T) {
	data := `{"items":[{"id":{"videoId":"xyz"},"snippet":{"title":"Rec","channelTitle":"Chan"}}]}`
	c := &Client{Key: "k", Client: &http.Client{Transport: rt{status: 200, body: data}}}
	tracks, err := c.GetRecommendations(context.Background(), []string{"seed"})
	if err != nil || len(tracks) != 1 || tracks[0].Name != "Rec" {
		t.Fatalf("unexpected result: %v %+v", err, tracks)
	}
}
