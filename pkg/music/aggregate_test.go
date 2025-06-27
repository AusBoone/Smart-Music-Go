package music

import (
	libspotify "github.com/zmb3/spotify"
	"testing"
)

// TestAggregatorMerge ensures that results from multiple services are combined
// and duplicates removed.
func TestAggregatorMerge(t *testing.T) {
	svc1 := fakeService{tracks: []Track{newTrack("1")}}
	svc2 := fakeService{tracks: []Track{newTrack("2"), newTrack("1")}}
	agg := Aggregator{Services: []Service{svc1, svc2}}
	res, _ := agg.SearchTrack("x")
	if len(res) != 2 {
		t.Fatalf("expected 2 results got %d", len(res))
	}
}

type fakeService struct{ tracks []Track }

func (f fakeService) SearchTrack(string) ([]Track, error)          { return f.tracks, nil }
func (f fakeService) GetRecommendations([]string) ([]Track, error) { return f.tracks, nil }

func newTrack(id string) Track {
	return Track{SimpleTrack: libspotify.SimpleTrack{ID: libspotify.ID(id)}}
}
