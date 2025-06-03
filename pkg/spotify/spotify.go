// This file will contain the code to interact with the Spotify API.

package spotify

import (
	"context"
	"fmt"
	"github.com/zmb3/spotify"
	"golang.org/x/oauth2/clientcredentials"
)

// SpotifyClient is a wrapper around the Spotify API client
type SpotifyClient struct {
	Client spotify.Client
}

// NewSpotifyClient creates a new Spotify API client with client credentials
func NewSpotifyClient(clientID string, clientSecret string) (SpotifyClient, error) {
	config := &clientcredentials.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     spotify.TokenURL,
	}

	token, err := config.Token(context.Background())
	if err != nil {
		return SpotifyClient{}, err
	}

	client := spotify.Authenticator{}.NewClient(token)
	return SpotifyClient{Client: client}, nil
}

// SearchTrack searches for a track on Spotify
// This function performs a search on Spotify for the given track and returns the first result.
// If no tracks are found, it returns an error.
func (sc *SpotifyClient) SearchTrack(track string) (spotify.FullTrack, error) {
	results, err := sc.Client.Search(track, spotify.SearchTypeTrack)
	if err != nil {
		return spotify.FullTrack{}, err
	}

	if results.Tracks != nil && len(results.Tracks.Tracks) > 0 {
		return results.Tracks.Tracks[0], nil
	}

	return spotify.FullTrack{}, fmt.Errorf("no tracks found")
}
