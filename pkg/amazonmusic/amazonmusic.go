// Package amazonmusic is a placeholder implementation of music.Service.
// Amazon Music does not currently expose a public API so this client simply
// returns an error. The file demonstrates how an additional provider could be
// integrated once an official API becomes available.
package amazonmusic

import (
	"context"
	"fmt"

	"Smart-Music-Go/pkg/music"
)

// Client satisfies the music.Service interface but does not perform any calls.
type Client struct{}

// Compile-time check.
var _ music.Service = (*Client)(nil)

// SearchTrack always returns an error indicating the API is unavailable. It
// satisfies the interface so callers can compile even though the feature is
// missing.
func (c *Client) SearchTrack(context.Context, string) ([]music.Track, error) {
	return nil, fmt.Errorf("amazon music api not implemented")
}

// GetRecommendations always returns an error for the same reason. Once Amazon
// exposes a recommendations API this function would call it and translate the
// response into music.Track values.
func (c *Client) GetRecommendations(context.Context, []string) ([]music.Track, error) {
	return nil, fmt.Errorf("amazon music api not implemented")
}
