package soundcloud

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// roundTripper allows mocking HTTP responses for tests.
type roundTripper struct{ data string }

func (rt roundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	resp := httptest.NewRecorder()
	resp.WriteString(rt.data)
	return resp.Result(), nil
}

// TestSearchTrack checks that the client decodes search results.
func TestSearchTrack(t *testing.T) {
	data := `{"collection":[{"id":1,"title":"Song","user":{"username":"Artist"}}]}`
	c := &Client{ClientID: "x", HTTP: &http.Client{Transport: roundTripper{data: data}}}
	res, err := c.SearchTrack("test")
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 1 || res[0].Name != "Song" {
		t.Fatalf("unexpected result %+v", res)
	}
}
