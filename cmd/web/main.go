// This file will initialize our application and start the server.

package main

import (
	"log"
	"net/http"
	"os"

	libspotify "github.com/zmb3/spotify"

	"Smart-Music-Go/pkg/handlers"
	"Smart-Music-Go/pkg/spotify"
)

func main() {
	// Initialize a new http.ServeMux, which is basically a HTTP request router (or multiplexer)
	mux := http.NewServeMux()

	// Load Spotify credentials from environment variables
	clientID := os.Getenv("SPOTIFY_CLIENT_ID")
	clientSecret := os.Getenv("SPOTIFY_CLIENT_SECRET")
	redirectURL := os.Getenv("SPOTIFY_REDIRECT_URL")
	if redirectURL == "" {
		redirectURL = "http://localhost:4000/callback"
	}
	if clientID == "" || clientSecret == "" {
		log.Fatal("SPOTIFY_CLIENT_ID and SPOTIFY_CLIENT_SECRET must be set")
	}

	// Initialize a Spotify client and application with dependencies
	sc := spotify.NewSpotifyClient(clientID, clientSecret)
	auth := libspotify.NewAuthenticator(redirectURL, libspotify.ScopePlaylistReadPrivate)
	auth.SetAuthInfo(clientID, clientSecret)
	app := &handlers.Application{Spotify: sc, Authenticator: auth}

	// Register the two URL patterns and their corresponding handler functions to the router
	mux.HandleFunc("/", app.Home)
	mux.HandleFunc("/search", app.Search)
	mux.HandleFunc("/login", app.Login)
	mux.HandleFunc("/callback", app.OAuthCallback)
	mux.HandleFunc("/playlists", app.Playlists)
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./ui/static"))))

	// Start the HTTP server
	http.ListenAndServe(":4000", mux)
}
