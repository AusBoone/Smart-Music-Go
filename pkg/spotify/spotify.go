// This file will contain the code to interact with the Spotify API.

package spotify

import (
	"context"
	"fmt"

	"github.com/zmb3/spotify"
	"golang.org/x/oauth2/clientcredentials"
)

// SpotifyClient is a wrapper around the Spotify API client
type searcher interface {
	Search(query string, t spotify.SearchType) (*spotify.SearchResult, error)
}

// SpotifyClient is a wrapper around the Spotify API client
type SpotifyClient struct {
	client searcher
}

// TrackSearcher describes the ability to search for tracks.
type TrackSearcher interface {
	SearchTrack(track string) ([]spotify.FullTrack, error)
}

var _ TrackSearcher = (*SpotifyClient)(nil)

// NewSpotifyClient creates a new Spotify API client with client credentials
func NewSpotifyClient(clientID string, clientSecret string) (*SpotifyClient, error) {
	config := &clientcredentials.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     spotify.TokenURL,
	}

	token, err := config.Token(context.Background())
	if err != nil {
		return nil, err
	}

	c := spotify.Authenticator{}.NewClient(token)
	return &SpotifyClient{client: &c}, nil
}

// SearchTrack searches for a track on Spotify.
// It performs a search for the given track and returns all results.
// If no tracks are found, it returns an error.
func (sc *SpotifyClient) SearchTrack(track string) ([]spotify.FullTrack, error) {
	results, err := sc.client.Search(track, spotify.SearchTypeTrack)
	if err != nil {
		return nil, err
	}

	if results.Tracks != nil && len(results.Tracks.Tracks) > 0 {
		return results.Tracks.Tracks, nil
	}

	return nil, fmt.Errorf("no tracks found")
}
