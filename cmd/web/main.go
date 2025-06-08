// This file will initialize our application and start the server.

package main

import (
	"log"
	"net/http"
	"os"

	libspotify "github.com/zmb3/spotify"

	"Smart-Music-Go/pkg/db"
	"Smart-Music-Go/pkg/handlers"
	"Smart-Music-Go/pkg/spotify"
)

// main configures application dependencies and starts the HTTP server.
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
	sc, err := spotify.NewSpotifyClient(clientID, clientSecret)
	if err != nil {
		log.Fatalf("spotify client init: %v", err)
	}
	auth := libspotify.NewAuthenticator(redirectURL, libspotify.ScopePlaylistReadPrivate)
	auth.SetAuthInfo(clientID, clientSecret)
	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		dbPath = "smartmusic.db"
	}
	database, err := db.New(dbPath)
	if err != nil {
		log.Fatalf("db init: %v", err)
	}
	defer database.Close()

	app := &handlers.Application{Spotify: sc, Authenticator: auth, DB: database}

	// Register the two URL patterns and their corresponding handler functions to the router
	mux.HandleFunc("/", app.Home)
	mux.HandleFunc("/search", app.Search)
	mux.HandleFunc("/api/search", app.SearchJSON)
	mux.HandleFunc("/login", app.Login)
	mux.HandleFunc("/callback", app.OAuthCallback)
	mux.HandleFunc("/playlists", app.Playlists)
	mux.HandleFunc("/api/playlists", app.PlaylistsJSON)
	mux.HandleFunc("/favorites", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			app.AddFavorite(w, r)
		} else {
			app.Favorites(w, r)
		}
	})
	mux.HandleFunc("/api/favorites", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			app.AddFavorite(w, r)
		} else {
			app.FavoritesJSON(w, r)
		}
	})
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./ui/static"))))
	mux.Handle("/app/", http.StripPrefix("/app/", http.FileServer(http.Dir("./ui/frontend/dist"))))

	// Start the HTTP server and log any startup error
	if err := http.ListenAndServe(":4000", mux); err != nil {
		log.Fatalf("http server error: %v", err)
	}
}
