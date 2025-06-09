// This file will contain the code to interact with the Spotify API.

package spotify

import (
	"context"
	"fmt"

	"github.com/zmb3/spotify"
	"golang.org/x/oauth2/clientcredentials"
)

// searcher defines the subset of the spotify.Client used by this package.
// It allows the concrete client to be replaced in tests.
type searcher interface {
	Search(query string, t spotify.SearchType) (*spotify.SearchResult, error)
}

// SpotifyClient wraps the official Spotify client providing higher level
// helper methods.
type SpotifyClient struct {
	client searcher
}

// TrackSearcher describes the ability to search for tracks.
type TrackSearcher interface {
	SearchTrack(track string) ([]spotify.FullTrack, error)
}

// Compile-time interface check ensuring SpotifyClient implements TrackSearcher.
var _ TrackSearcher = (*SpotifyClient)(nil)

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
func (sc *SpotifyClient) SearchTrack(track string) ([]spotify.FullTrack, error) {
	// Use the wrapped client to search for the track name.
	results, err := sc.client.Search(track, spotify.SearchTypeTrack)
	if err != nil {
		return nil, err
	}

	if results.Tracks != nil && len(results.Tracks.Tracks) > 0 {
		return results.Tracks.Tracks, nil
	}

	// Indicate to callers that nothing matched the query.
	return nil, fmt.Errorf("no tracks found")
}
