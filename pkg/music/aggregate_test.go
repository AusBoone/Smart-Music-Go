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
	agg := Aggregator{Services: []Service{svc1, svc2}, MaxConcurrent: 1}
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
	agg := Aggregator{Services: []Service{svc1, svc2}, MaxConcurrent: 1}
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

// TestAggregatorNoServices returns nil results when no services are configured.
func TestAggregatorNoServices(t *testing.T) {
	agg := Aggregator{}
	res, err := agg.SearchTrack(context.Background(), "q")
	if err != nil || len(res) != 0 {
		t.Fatalf("expected empty results, got %v %v", res, err)
	}
	recs, err := agg.GetRecommendations(context.Background(), []string{"x"})
	if err != nil || len(recs) != 0 {
		t.Fatalf("expected empty recommendations, got %v %v", recs, err)
	}
}

// TestAggregatorAllErrors ensures an error is returned when every service fails.
func TestAggregatorAllErrors(t *testing.T) {
	svc := failingService{err: context.DeadlineExceeded}
	agg := Aggregator{Services: []Service{svc, svc}}
	if _, err := agg.SearchTrack(context.Background(), "q"); err == nil {
		t.Fatal("expected error when all services fail")
	}
	if _, err := agg.GetRecommendations(context.Background(), []string{"x"}); err == nil {
		t.Fatal("expected error when all services fail")
	}
}

type fakeService struct{ tracks []Track }

func (f fakeService) SearchTrack(context.Context, string) ([]Track, error) { return f.tracks, nil }
func (f fakeService) GetRecommendations(context.Context, []string) ([]Track, error) {
	return f.tracks, nil
}

// failingService always returns the provided error. It is used to simulate
// complete service outages when testing the aggregator's error handling.
type failingService struct{ err error }

func (f failingService) SearchTrack(context.Context, string) ([]Track, error) { return nil, f.err }
func (f failingService) GetRecommendations(context.Context, []string) ([]Track, error) {
	return nil, f.err
}

func newTrack(id string) Track {
	return Track{SimpleTrack: libspotify.SimpleTrack{ID: libspotify.ID(id)}}
}
