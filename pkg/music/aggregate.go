// Package music provides interfaces for interacting with music services.
// This file implements an aggregation service which combines multiple
// providers to broaden search results and recommendations.
package music

import "sync"

// Aggregator queries each configured Service and merges the results.
// It is useful when the application wants to search across multiple
// providers (e.g. Spotify and YouTube) simultaneously.
type Aggregator struct {
	Services []Service
}

// SearchTrack returns the union of results from all underlying services.
// Duplicates are removed based on track ID. Failure of one service does not
// prevent results from others.
func (a Aggregator) SearchTrack(q string) ([]Track, error) {
	if len(a.Services) == 0 {
		return nil, nil
	}
	var wg sync.WaitGroup
	resCh := make(chan []Track, len(a.Services))
	for _, svc := range a.Services {
		svc := svc
		wg.Add(1)
		go func() {
			defer wg.Done()
			tracks, _ := svc.SearchTrack(q)
			resCh <- tracks
		}()
	}
	wg.Wait()
	close(resCh)
	seen := make(map[string]struct{})
	var merged []Track
	for r := range resCh {
		for _, t := range r {
			id := string(t.ID)
			if _, ok := seen[id]; !ok {
				seen[id] = struct{}{}
				merged = append(merged, t)
			}
		}
	}
	return merged, nil
}

// GetRecommendations merges recommendations from all services. Only the first
// seed ID is passed through to providers that do not support multiple seeds.
func (a Aggregator) GetRecommendations(seedIDs []string) ([]Track, error) {
	if len(a.Services) == 0 {
		return nil, nil
	}
	var wg sync.WaitGroup
	resCh := make(chan []Track, len(a.Services))
	for _, svc := range a.Services {
		svc := svc
		wg.Add(1)
		go func() {
			defer wg.Done()
			tracks, _ := svc.GetRecommendations(seedIDs)
			resCh <- tracks
		}()
	}
	wg.Wait()
	close(resCh)
	seen := make(map[string]struct{})
	var merged []Track
	for r := range resCh {
		for _, t := range r {
			id := string(t.ID)
			if _, ok := seen[id]; !ok {
				seen[id] = struct{}{}
				merged = append(merged, t)
			}
		}
	}
	return merged, nil
}
