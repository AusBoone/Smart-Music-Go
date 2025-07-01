package music

import (
	"context"
	libspotify "github.com/zmb3/spotify"
	"testing"
)

// TestAggregatorMerge ensures that results from multiple services are combined
// and duplicates removed.
func TestAggregatorMerge(t *testing.T) {
	svc1 := fakeService{tracks: []Track{newTrack("1")}}
	svc2 := fakeService{tracks: []Track{newTrack("2"), newTrack("1")}}
	agg := Aggregator{Services: []Service{svc1, svc2}}
	res, _ := agg.SearchTrack(context.Background(), "x")
	if len(res) != 2 {
		t.Fatalf("expected 2 results got %d", len(res))
	}
}

// TestAggregatorGetRecommendations verifies that GetRecommendations merges
// results from multiple services and eliminates duplicate track IDs. Each fake
// service returns overlapping recommendations to ensure the merge logic works
// correctly.
func TestAggregatorGetRecommendations(t *testing.T) {
	svc1 := fakeService{tracks: []Track{newTrack("a"), newTrack("b")}}
	svc2 := fakeService{tracks: []Track{newTrack("b"), newTrack("c")}}
	agg := Aggregator{Services: []Service{svc1, svc2}}
	res, err := agg.GetRecommendations(context.Background(), []string{"seed"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res) != 3 {
		t.Fatalf("expected 3 unique tracks got %d", len(res))
	}
	got := map[string]bool{}
	for _, t := range res {
		got[string(t.ID)] = true
	}
	for _, id := range []string{"a", "b", "c"} {
		if !got[id] {
			t.Errorf("missing %s in results", id)
		}
	}
}

type fakeService struct{ tracks []Track }

func (f fakeService) SearchTrack(context.Context, string) ([]Track, error) { return f.tracks, nil }
func (f fakeService) GetRecommendations(context.Context, []string) ([]Track, error) {
	return f.tracks, nil
}

func newTrack(id string) Track {
	return Track{SimpleTrack: libspotify.SimpleTrack{ID: libspotify.ID(id)}}
}
