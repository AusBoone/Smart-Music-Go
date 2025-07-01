// Package soundcloud implements the music.Service interface using the
// SoundCloud public API. Only minimal search and recommendation features
// are provided as a demonstration; a client_id must be supplied via the
// environment or configuration.
package soundcloud

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

// Client talks to the SoundCloud API. If HTTP is nil a client with a 10 second
// timeout is used. The zero value is therefore ready for basic use.
type Client struct {
	ClientID string
	HTTP     *http.Client
}

// Ensure interface compliance at compile time.
var _ music.Service = (*Client)(nil)

// SearchTrack queries the SoundCloud search API and converts results.
func (c *Client) SearchTrack(ctx context.Context, q string) ([]music.Track, error) {
	if c.HTTP == nil {
		c.HTTP = &http.Client{Timeout: 10 * time.Second}
	}
	params := url.Values{
		"q":         {q},
		"client_id": {c.ClientID},
		"limit":     {"5"},
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://api-v2.soundcloud.com/search/tracks?"+params.Encode(), nil)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("soundcloud search error: %s", resp.Status)
	}
	var body struct {
		Collection []struct {
			ID    int64  `json:"id"`
			Title string `json:"title"`
			User  struct {
				Username string `json:"username"`
			} `json:"user"`
		} `json:"collection"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}
	if len(body.Collection) == 0 {
		return nil, fmt.Errorf("no tracks found")
	}
	tracks := make([]music.Track, len(body.Collection))
	for i, item := range body.Collection {
		tracks[i] = libspotify.FullTrack{
			SimpleTrack: libspotify.SimpleTrack{
				ID:      libspotify.ID(fmt.Sprintf("sc-%d", item.ID)),
				Name:    item.Title,
				Artists: []libspotify.SimpleArtist{{Name: item.User.Username}},
				ExternalURLs: map[string]string{
					"soundcloud": fmt.Sprintf("https://soundcloud.com/tracks/%d", item.ID),
				},
			},
		}
	}
	return tracks, nil
}

// GetRecommendations is not supported for SoundCloud and returns an error.
func (c *Client) GetRecommendations(ctx context.Context, seedIDs []string) ([]music.Track, error) {
	return nil, fmt.Errorf("recommendations not supported")
}
