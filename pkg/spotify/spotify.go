// Package spotify wraps the official Spotify client library providing helper
// functions used by the web application. It performs authentication using the
// client credentials flow and exposes a minimal interface required by the
// handlers package. Errors are returned directly from the underlying client so
// callers can inspect them if needed.
//
// All exported methods now accept a context parameter allowing callers to
// cancel long running requests. The wrapped library does not provide context
// support so cancellation is checked explicitly before each call.

package spotify

import (
	"context"
	"fmt"

	"github.com/zmb3/spotify"
	"golang.org/x/oauth2/clientcredentials"

	"Smart-Music-Go/pkg/music"
)

// searcher defines the subset of the spotify.Client used by this package.
// It allows the concrete client to be replaced in tests.
type searcher interface {
	Search(query string, t spotify.SearchType) (*spotify.SearchResult, error)
	GetRecommendations(seeds spotify.Seeds, attrs *spotify.TrackAttributes, opt *spotify.Options) (*spotify.Recommendations, error)
	GetAudioFeatures(ids ...spotify.ID) ([]*spotify.AudioFeatures, error)
}

// SpotifyClient wraps the official Spotify client providing higher level
// helper methods.
type SpotifyClient struct {
	client searcher
}

// Compile-time interface check ensuring SpotifyClient satisfies the generic
// music.Service interface used by the rest of the application.
var _ music.Service = (*SpotifyClient)(nil)

// NewSpotifyClient authenticates using the client credentials flow and returns
// a SpotifyClient ready for API calls. clientID and clientSecret are obtained
// from the Spotify developer dashboard.
func NewSpotifyClient(clientID string, clientSecret string) (*SpotifyClient, error) {
	// Use the client credentials OAuth2 flow to obtain an application token
	// which allows searching the Spotify catalog without a user login.
	config := &clientcredentials.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     spotify.TokenURL,
	}

	token, err := config.Token(context.Background())
	if err != nil {
		return nil, err
	}

	// Create the Spotify client from the retrieved token.
	c := spotify.Authenticator{}.NewClient(token)
	return &SpotifyClient{client: &c}, nil
}

// SearchTrack queries the Spotify API for the supplied track name and returns
// all matching tracks.  A "no tracks found" error is returned when the result
// set is empty.
// SearchTrack implements music.Service by querying the Spotify API for the
// supplied track name and returning matching items.
func (sc *SpotifyClient) SearchTrack(ctx context.Context, track string) ([]music.Track, error) {
	// Use the wrapped client to search for the track name.
	// The underlying client does not accept a context, but we honour the
	// provided one by checking for cancellation before returning.
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	results, err := sc.client.Search(track, spotify.SearchTypeTrack)
	if err != nil {
		return nil, err
	}

	if results.Tracks != nil && len(results.Tracks.Tracks) > 0 {
		tracks := make([]music.Track, len(results.Tracks.Tracks))
		copy(tracks, results.Tracks.Tracks)
		return tracks, nil
	}

	// Indicate to callers that nothing matched the query.
	return nil, fmt.Errorf("no tracks found")
}

// GetRecommendations returns tracks recommended based on the provided seeds.
// At least one seed must be supplied. If no tracks are returned an error is
// generated so callers can distinguish the empty case.
// GetRecommendations implements music.Service to return tracks related to the
// provided seeds using Spotify's recommendation API.
func (sc *SpotifyClient) GetRecommendations(ctx context.Context, seedIDs []string) ([]music.Track, error) {
	seeds := spotify.Seeds{Tracks: make([]spotify.ID, len(seedIDs))}
	for i, id := range seedIDs {
		seeds.Tracks[i] = spotify.ID(id)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	recs, err := sc.client.GetRecommendations(seeds, nil, nil)
	if err != nil {
		return nil, err
	}
	if len(recs.Tracks) == 0 {
		return nil, fmt.Errorf("no recommendations found")
	}
	tracks := make([]music.Track, len(recs.Tracks))
	for i, t := range recs.Tracks {
		tracks[i] = spotify.FullTrack{SimpleTrack: t}
	}
	return tracks, nil
}

// GetAudioFeatures retrieves audio features for the specified track IDs. The
// returned slice matches the number and order of the supplied IDs.
func (sc *SpotifyClient) GetAudioFeatures(ids ...string) ([]*spotify.AudioFeatures, error) {
	spotifyIDs := make([]spotify.ID, len(ids))
	for i, id := range ids {
		spotifyIDs[i] = spotify.ID(id)
	}
	feats, err := sc.client.GetAudioFeatures(spotifyIDs...)
	if err != nil {
		return nil, err
	}
	return feats, nil
}

// GetRecommendationsWithAttrs returns recommendations using additional track
// attributes such as tempo or energy. Callers can pass nil attrs to use the
// default behaviour.
func (sc *SpotifyClient) GetRecommendationsWithAttrs(ctx context.Context, seedIDs []string, attrs *spotify.TrackAttributes) ([]music.Track, error) {
	seeds := spotify.Seeds{Tracks: make([]spotify.ID, len(seedIDs))}
	for i, id := range seedIDs {
		seeds.Tracks[i] = spotify.ID(id)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	recs, err := sc.client.GetRecommendations(seeds, attrs, nil)
	if err != nil {
		return nil, err
	}
	if len(recs.Tracks) == 0 {
		return nil, fmt.Errorf("no recommendations found")
	}
	tracks := make([]music.Track, len(recs.Tracks))
	for i, t := range recs.Tracks {
		tracks[i] = spotify.FullTrack{SimpleTrack: t}
	}
	return tracks, nil
}
