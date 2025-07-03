// Package tidal implements the music.Service interface using the public Tidal
// API. Only basic track search is supported. A token is required which can be
// obtained from the Tidal web player. The client does not perform authentication
// itself.
package tidal

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

// Client queries the Tidal API. A valid Token from the Tidal web player is
// required. If HTTP is nil SearchTrack creates a client with a 10 second
// timeout. CountryCode controls localisation and defaults to "US".
type Client struct {
	Token       string
	CountryCode string
	HTTP        *http.Client
}

// Ensure interface compliance.
var _ music.Service = (*Client)(nil)

// SearchTrack executes a search against Tidal. It returns up to five tracks
// matching q or an error when the request fails. IDs are prefixed with
// "tidal-" to avoid clashing with Spotify IDs used elsewhere.
func (c *Client) SearchTrack(ctx context.Context, q string) ([]music.Track, error) {
	if c.Token == "" {
		return nil, fmt.Errorf("token required")
	}
	if c.HTTP == nil {
		c.HTTP = &http.Client{Timeout: 10 * time.Second}
	}
	cc := c.CountryCode
	if cc == "" {
		cc = "US"
	}
	params := url.Values{
		"query":       {q},
		"limit":       {"5"},
		"offset":      {"0"},
		"countryCode": {cc},
		"token":       {c.Token},
	}
	u := "https://api.tidal.com/v1/search/tracks?" + params.Encode()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tidal search error: %s", resp.Status)
	}
	var body struct {
		Tracks struct {
			Items []struct {
				ID     int64  `json:"id"`
				Title  string `json:"title"`
				Artist struct {
					Name string `json:"name"`
				} `json:"artist"`
			} `json:"items"`
		} `json:"tracks"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		// Invalid JSON means the API changed or the token is invalid.
		return nil, err
	}
	if len(body.Tracks.Items) == 0 {
		// No results is treated as an error for parity with other services.
		return nil, fmt.Errorf("no tracks found")
	}
	tracks := make([]music.Track, len(body.Tracks.Items))
	for i, item := range body.Tracks.Items {
		// Only minimal metadata is extracted to keep the interface consistent.
		tracks[i] = libspotify.FullTrack{
			SimpleTrack: libspotify.SimpleTrack{
				ID:      libspotify.ID(fmt.Sprintf("tidal-%d", item.ID)),
				Name:    item.Title,
				Artists: []libspotify.SimpleArtist{{Name: item.Artist.Name}},
				ExternalURLs: map[string]string{
					"tidal": fmt.Sprintf("https://tidal.com/browse/track/%d", item.ID),
				},
			},
		}
	}
	return tracks, nil
}

// GetRecommendations is not implemented for Tidal since the public API does not
// expose a recommendation endpoint. It exists to satisfy the Service interface.
func (c *Client) GetRecommendations(ctx context.Context, seeds []string) ([]music.Track, error) {
	return nil, fmt.Errorf("recommendations not supported")
}
