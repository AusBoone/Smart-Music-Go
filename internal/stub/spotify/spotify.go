package spotify

import "golang.org/x/oauth2"

const TokenURL = "https://example.com/token"

type Authenticator struct{}

func (a Authenticator) NewClient(token *oauth2.Token) Client {
	return Client{}
}

type Client struct {
	SearchFunc func(query string, t SearchType) (*SearchResult, error)
}

func (c Client) Search(query string, t SearchType) (*SearchResult, error) {
	if c.SearchFunc != nil {
		return c.SearchFunc(query, t)
	}
	return &SearchResult{}, nil
}

type SearchType int

const SearchTypeTrack SearchType = iota

type SearchResult struct {
	Tracks *SimpleTrackPage
}

type SimpleTrackPage struct {
	Tracks []FullTrack
}

type FullTrack struct {
	Name         string
	Artists      []SimpleArtist
	ExternalURLs struct{ Spotify string }
}

type SimpleArtist struct{ Name string }
