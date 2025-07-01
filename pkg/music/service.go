// Package music defines generic interfaces and data structures for
// interacting with music providers. Implementations can wrap Spotify,
// YouTube or any other service. By depending on this package the rest of
// the application can remain agnostic about the underlying platform.
//
// Track is currently an alias of spotify.FullTrack so handlers and
// templates operate on familiar fields (Name, Album, Artists etc). Other
// services should populate these fields where possible.
package music

import (
	"context"

	libspotify "github.com/zmb3/spotify"
)

// Track represents a track returned by a music service. For compatibility
// with the existing templates it mirrors spotify.FullTrack.
type Track = libspotify.FullTrack

// Service exposes searching and recommendation capabilities. Additional
// features can be implemented by concrete services.
type Service interface {
	// SearchTrack returns tracks matching the query string. The context is
	// used for request cancellation and timeout propagation. An error is
	// returned when the service encounters a failure or no tracks are found.
	SearchTrack(ctx context.Context, query string) ([]Track, error)

	// GetRecommendations returns tracks related to the provided seed IDs.
	// The context controls request cancellation. At least one seed must be
	// supplied or an error is returned.
	GetRecommendations(ctx context.Context, seedIDs []string) ([]Track, error)
}
