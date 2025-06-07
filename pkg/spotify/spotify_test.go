package spotify

import (
	"errors"
	"testing"

	libspotify "github.com/zmb3/spotify"
)

type fakeSearcher struct {
	lastQuery string
	lastType  libspotify.SearchType
	result    *libspotify.SearchResult
	err       error
}

func (f *fakeSearcher) Search(query string, t libspotify.SearchType) (*libspotify.SearchResult, error) {
	f.lastQuery = query
	f.lastType = t
	return f.result, f.err
}

func TestSearchTrackFound(t *testing.T) {
	track := libspotify.FullTrack{SimpleTrack: libspotify.SimpleTrack{Name: "Song"}}
	sr := &libspotify.SearchResult{Tracks: &libspotify.FullTrackPage{Tracks: []libspotify.FullTrack{track}}}
	fs := &fakeSearcher{result: sr}
	sc := &SpotifyClient{client: fs}

	got, err := sc.SearchTrack("q")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].Name != "Song" {
		t.Errorf("unexpected result: %+v", got)
	}
	if fs.lastQuery != "q" || fs.lastType != libspotify.SearchTypeTrack {
		t.Errorf("Search called with %s %v", fs.lastQuery, fs.lastType)
	}
}

func TestSearchTrackNotFound(t *testing.T) {
	sr := &libspotify.SearchResult{Tracks: &libspotify.FullTrackPage{}}
	sc := &SpotifyClient{client: &fakeSearcher{result: sr}}

	_, err := sc.SearchTrack("missing")
	if err == nil || err.Error() != "no tracks found" {
		t.Fatalf("expected no tracks found error, got %v", err)
	}
}

func TestSearchTrackError(t *testing.T) {
	fs := &fakeSearcher{err: errors.New("boom")}
	sc := &SpotifyClient{client: fs}

	_, err := sc.SearchTrack("fail")
	if err == nil || err.Error() != "boom" {
		t.Fatalf("expected boom error, got %v", err)
	}
}
