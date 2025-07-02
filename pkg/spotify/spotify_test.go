package spotify

import (
	"context"
	"errors"
	"testing"

	libspotify "github.com/zmb3/spotify"
)

type fakeSearcher struct {
	lastQuery string
	lastType  libspotify.SearchType
	result    *libspotify.SearchResult
	recSeeds  libspotify.Seeds
	recAttrs  *libspotify.TrackAttributes
	recs      *libspotify.Recommendations
	featIDs   []libspotify.ID
	feats     []*libspotify.AudioFeatures
	err       error
}

func (f *fakeSearcher) GetAudioFeatures(ids ...libspotify.ID) ([]*libspotify.AudioFeatures, error) {
	f.featIDs = ids
	return f.feats, f.err
}

func (f *fakeSearcher) GetRecommendations(seeds libspotify.Seeds, attrs *libspotify.TrackAttributes, opt *libspotify.Options) (*libspotify.Recommendations, error) {
	f.recSeeds = seeds
	f.recAttrs = attrs
	return f.recs, f.err
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

	got, err := sc.SearchTrack(context.Background(), "q")
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

	_, err := sc.SearchTrack(context.Background(), "missing")
	if err == nil || err.Error() != "no tracks found" {
		t.Fatalf("expected no tracks found error, got %v", err)
	}
}

func TestSearchTrackError(t *testing.T) {
	fs := &fakeSearcher{err: errors.New("boom")}
	sc := &SpotifyClient{client: fs}

	_, err := sc.SearchTrack(context.Background(), "fail")
	if err == nil || err.Error() != "boom" {
		t.Fatalf("expected boom error, got %v", err)
	}
}

// TestGetAudioFeatures verifies that track IDs are passed through and features
// are returned in the same order.
func TestGetAudioFeatures(t *testing.T) {
	feats := []*libspotify.AudioFeatures{{Tempo: 120}}
	fs := &fakeSearcher{feats: feats}
	sc := &SpotifyClient{client: fs}

	got, err := sc.GetAudioFeatures("1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fs.featIDs) != 1 || fs.featIDs[0] != "1" {
		t.Errorf("ids not forwarded: %+v", fs.featIDs)
	}
	if got[0].Tempo != 120 {
		t.Errorf("wrong features")
	}
}

// TestGetRecommendationsWithAttrs checks that attributes are forwarded and
// empty results produce an error.
func TestGetRecommendationsWithAttrs(t *testing.T) {
	rec := &libspotify.Recommendations{Tracks: []libspotify.SimpleTrack{{ID: "2"}}}
	attrs := libspotify.NewTrackAttributes().MinEnergy(0.5)
	fs := &fakeSearcher{recs: rec}
	sc := &SpotifyClient{client: fs}

	got, err := sc.GetRecommendationsWithAttrs(context.Background(), []string{"1"}, attrs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fs.recAttrs != attrs || len(fs.recSeeds.Tracks) != 1 {
		t.Errorf("attributes or seeds not passed")
	}
	if len(got) != 1 || got[0].ID != "2" {
		t.Errorf("unexpected tracks: %+v", got)
	}

	// empty results should return an error
	fs.recs = &libspotify.Recommendations{}
	_, err = sc.GetRecommendationsWithAttrs(context.Background(), []string{"1"}, attrs)
	if err == nil || err.Error() != "no recommendations found" {
		t.Errorf("expected empty error, got %v", err)
	}
}
