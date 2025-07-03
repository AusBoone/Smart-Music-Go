// Package applemusic implements the music.Service interface using the public
// iTunes Search API. It allows searching for tracks without requiring user
// authentication. The zero value Client is ready for use; an http.Client with a
// reasonable timeout will be created when nil.
package applemusic

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	libspotify "github.com/zmb3/spotify"

	"Smart-Music-Go/pkg/music"
)

// Client provides access to Apple's iTunes Search API. HTTP may be nil in which
// case SearchTrack will allocate an http.Client with a 10 second timeout. The
// struct contains no other fields so the zero value is ready for use.
type Client struct {
	HTTP *http.Client
}

// Ensure interface compliance at compile time.
var _ music.Service = (*Client)(nil)

// SearchTrack queries the iTunes search API and converts results into music.Track
// values. An error is returned when no tracks are found or the request fails.
// SearchTrack queries the iTunes endpoint for the provided text. It returns a
// slice of tracks on success or an error if the request fails or no items are
// found. The IDs are prefixed with "am-" so they do not collide with Spotify
// IDs used elsewhere in the application.
func (c *Client) SearchTrack(ctx context.Context, q string) ([]music.Track, error) {
	if c.HTTP == nil {
		// Lazily create the HTTP client with a sane timeout to avoid
		// leaking connections from default client usage.
		c.HTTP = &http.Client{Timeout: 10 * time.Second}
	}
	u := "https://itunes.apple.com/search"
	params := url.Values{
		"term":   {q},
		"entity": {"song"},
		"limit":  {"5"},
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u+"?"+params.Encode(), nil)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("itunes search error: %s", resp.Status)
	}
	// Body mirrors the subset of the iTunes JSON response we care about.
	// Only a few fields are needed to construct a music.Track.
	var body struct {
		Results []struct {
			TrackID    int64  `json:"trackId"`
			TrackName  string `json:"trackName"`
			ArtistName string `json:"artistName"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}
	if len(body.Results) == 0 {
		return nil, fmt.Errorf("no tracks found")
	}
	tracks := make([]music.Track, len(body.Results))
	for i, item := range body.Results {
		// Convert each iTunes result into a structure compatible with
		// the rest of the application. Only the minimal fields are
		// filled in so templates and JSON handlers behave consistently.
		tracks[i] = libspotify.FullTrack{
			SimpleTrack: libspotify.SimpleTrack{
				ID:      libspotify.ID(fmt.Sprintf("am-%d", item.TrackID)),
				Name:    item.TrackName,
				Artists: []libspotify.SimpleArtist{{Name: item.ArtistName}},
				ExternalURLs: map[string]string{
					"applemusic": fmt.Sprintf("https://music.apple.com/track/%d", item.TrackID),
				},
			},
		}
	}
	return tracks, nil
}

// GetRecommendations is not supported by the iTunes Search API and therefore
// always returns an error. It satisfies the music.Service interface so the
// client can be used in aggregation even though this feature is missing.
func (c *Client) GetRecommendations(ctx context.Context, seeds []string) ([]music.Track, error) {
	return nil, fmt.Errorf("recommendations not supported")
}
