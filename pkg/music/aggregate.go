// Package music provides interfaces for interacting with music services.
// This file implements an aggregation service which combines multiple
// providers to broaden search results and recommendations.
//
// Update: error handling now surfaces an error when all configured
// services fail. Previously failures were swallowed and an empty slice
// was returned, making diagnosis difficult.
package music

import (
	"context"
	"sync"
)

// Aggregator queries each configured Service and merges the results.
// It is useful when the application wants to search across multiple
// providers (e.g. Spotify and YouTube) simultaneously.
type Aggregator struct {
	Services      []Service
	MaxConcurrent int
}

// SearchTrack returns the union of results from all underlying services.
// Duplicates are removed based on track ID. Failure of one service does not
// prevent results from others.
func (a Aggregator) SearchTrack(ctx context.Context, q string) ([]Track, error) {
	if len(a.Services) == 0 {
		return nil, nil
	}
	type result struct {
		tracks []Track
		err    error
	}
	var wg sync.WaitGroup
	resCh := make(chan result, len(a.Services))
	sem := make(chan struct{}, a.concurrency())
	for _, svc := range a.Services {
		svc := svc
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			tracks, err := svc.SearchTrack(ctx, q)
			<-sem
			resCh <- result{tracks: tracks, err: err}
		}()
	}
	wg.Wait()
	close(resCh)
	seen := make(map[string]struct{})
	var merged []Track
	var firstErr error
	successes := 0
	for r := range resCh {
		if r.err != nil {
			if firstErr == nil {
				firstErr = r.err
			}
			continue
		}
		successes++
		for _, t := range r.tracks {
			id := string(t.ID)
			if _, ok := seen[id]; !ok {
				seen[id] = struct{}{}
				merged = append(merged, t)
			}
		}
	}
	if successes == 0 && firstErr != nil {
		return nil, firstErr
	}
	return merged, nil
}

// concurrency returns the number of goroutines to use when querying services.
// If MaxConcurrent is not set or less than 1, it defaults to the number of services.
func (a Aggregator) concurrency() int {
	if a.MaxConcurrent > 0 {
		return a.MaxConcurrent
	}
	return len(a.Services)
}

// GetRecommendations merges recommendations from all services. Only the first
// seed ID is passed through to providers that do not support multiple seeds.
func (a Aggregator) GetRecommendations(ctx context.Context, seedIDs []string) ([]Track, error) {
	if len(a.Services) == 0 {
		return nil, nil
	}
	type result struct {
		tracks []Track
		err    error
	}
	var wg sync.WaitGroup
	resCh := make(chan result, len(a.Services))
	sem := make(chan struct{}, a.concurrency())
	for _, svc := range a.Services {
		svc := svc
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			tracks, err := svc.GetRecommendations(ctx, seedIDs)
			<-sem
			resCh <- result{tracks: tracks, err: err}
		}()
	}
	wg.Wait()
	close(resCh)
	seen := make(map[string]struct{})
	var merged []Track
	var firstErr error
	successes := 0
	for r := range resCh {
		if r.err != nil {
			if firstErr == nil {
				firstErr = r.err
			}
			continue
		}
		successes++
		for _, t := range r.tracks {
			id := string(t.ID)
			if _, ok := seen[id]; !ok {
				seen[id] = struct{}{}
				merged = append(merged, t)
			}
		}
	}
	if successes == 0 && firstErr != nil {
		return nil, firstErr
	}
	return merged, nil
}
