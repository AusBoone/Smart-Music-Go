package clientcredentials

import (
	"context"
	"golang.org/x/oauth2"
)

type Config struct {
	ClientID     string
	ClientSecret string
	TokenURL     string
}

func (c *Config) Token(ctx context.Context) (*oauth2.Token, error) {
	return &oauth2.Token{AccessToken: "stub"}, nil
}
