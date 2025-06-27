// Package youtube implements the music.Service interface using the
// YouTube Data API. Only the endpoints required by the application are
// supported. An API key must be provided when constructing the client.
//
// Network calls are performed using the provided http.Client allowing
// callers to substitute a test client.
package youtube

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	libspotify "github.com/zmb3/spotify"

	"Smart-Music-Go/pkg/music"
)

// Client provides access to the YouTube Data API.
type Client struct {
	Key    string
	Client *http.Client
}

// ensure Client implements the music.Service interface.
var _ music.Service = (*Client)(nil)

// SearchTrack queries the YouTube search API and converts results into
// music.Track values. Only the first page of results is returned.
func (c *Client) SearchTrack(q string) ([]music.Track, error) {
	if c.Client == nil {
		c.Client = http.DefaultClient
	}
	u := "https://www.googleapis.com/youtube/v3/search"
	params := url.Values{
		"part":       {"snippet"},
		"type":       {"video"},
		"maxResults": {"5"},
		"q":          {q},
		"key":        {c.Key},
	}
	resp, err := c.Client.Get(u + "?" + params.Encode())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("youtube search error: %s", resp.Status)
	}
	var body struct {
		Items []struct {
			ID struct {
				VideoID string `json:"videoId"`
			} `json:"id"`
			Snippet struct {
				Title        string `json:"title"`
				ChannelTitle string `json:"channelTitle"`
				Thumbnails   struct {
					Default struct{ URL string }
				}
			} `json:"snippet"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}
	if len(body.Items) == 0 {
		return nil, fmt.Errorf("no tracks found")
	}
	tracks := make([]music.Track, len(body.Items))
	for i, item := range body.Items {
		tracks[i] = libspotify.FullTrack{
			SimpleTrack: libspotify.SimpleTrack{
				ID:           libspotify.ID(item.ID.VideoID),
				Name:         item.Snippet.Title,
				Artists:      []libspotify.SimpleArtist{{Name: item.Snippet.ChannelTitle}},
				ExternalURLs: map[string]string{"youtube": "https://youtu.be/" + item.ID.VideoID},
			},
		}
	}
	return tracks, nil
}

// GetRecommendations fetches related videos for the given seed video ID.
func (c *Client) GetRecommendations(seeds []string) ([]music.Track, error) {
	if len(seeds) == 0 {
		return nil, fmt.Errorf("no seed ids provided")
	}
	if c.Client == nil {
		c.Client = http.DefaultClient
	}
	seed := seeds[0]
	u := "https://www.googleapis.com/youtube/v3/search"
	params := url.Values{
		"part":             {"snippet"},
		"type":             {"video"},
		"relatedToVideoId": {seed},
		"maxResults":       {"5"},
		"key":              {c.Key},
	}
	resp, err := c.Client.Get(u + "?" + params.Encode())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("youtube recommendation error: %s", resp.Status)
	}
	var body struct {
		Items []struct {
			ID      struct{ VideoID string } `json:"id"`
			Snippet struct {
				Title        string `json:"title"`
				ChannelTitle string `json:"channelTitle"`
				Thumbnails   struct{ Default struct{ URL string } }
			} `json:"snippet"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}
	if len(body.Items) == 0 {
		return nil, fmt.Errorf("no recommendations found")
	}
	tracks := make([]music.Track, len(body.Items))
	for i, item := range body.Items {
		tracks[i] = libspotify.FullTrack{
			SimpleTrack: libspotify.SimpleTrack{
				ID:           libspotify.ID(item.ID.VideoID),
				Name:         item.Snippet.Title,
				Artists:      []libspotify.SimpleArtist{{Name: item.Snippet.ChannelTitle}},
				ExternalURLs: map[string]string{"youtube": "https://youtu.be/" + item.ID.VideoID},
			},
		}
	}
	return tracks, nil
}
