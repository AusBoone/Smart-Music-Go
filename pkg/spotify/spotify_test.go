package spotify_test

import (
	spotifypkg "Smart-Music-Go/pkg/spotify"
	stub "github.com/zmb3/spotify"
	"testing"
)

func TestNewSpotifyClient(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("NewSpotifyClient panicked: %v", r)
		}
	}()
	sc := spotifypkg.NewSpotifyClient("id", "secret")
	_ = sc
}

func TestSearchTrackFound(t *testing.T) {
	expected := "Song"
	sc := spotifypkg.SpotifyClient{Client: stub.Client{
		SearchFunc: func(q string, st stub.SearchType) (*stub.SearchResult, error) {
			return &stub.SearchResult{Tracks: &stub.SimpleTrackPage{Tracks: []stub.FullTrack{{Name: expected}}}}, nil
		},
	}}
	track, err := sc.SearchTrack("anything")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if track.Name != expected {
		t.Errorf("expected %s got %s", expected, track.Name)
	}
}

func TestSearchTrackNotFound(t *testing.T) {
	sc := spotifypkg.SpotifyClient{Client: stub.Client{}}
	_, err := sc.SearchTrack("anything")
	if err == nil {
		t.Fatal("expected error")
	}
}
