// Command web initializes the Smart-Music-Go application and starts the HTTP
// server. Configuration is provided via environment variables for Spotify API
// credentials and database location. The server listens on the port specified
// by the PORT environment variable (default 4000) and serves both HTML pages and
// a JSON API. Recent additions include monthly
// insights and collaborative playlist routes.

package main

import (
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
	"strings"
	"time"

	libspotify "github.com/zmb3/spotify"

	"Smart-Music-Go/pkg/db"
	"Smart-Music-Go/pkg/handlers"
	"Smart-Music-Go/pkg/music"
	"Smart-Music-Go/pkg/soundcloud"
	"Smart-Music-Go/pkg/spotify"
	"Smart-Music-Go/pkg/youtube"
)

// main configures application dependencies and starts the HTTP server.
func main() {
	// Initialize a new http.ServeMux which will route incoming HTTP
	// requests to the appropriate handler based on the URL path.
	mux := http.NewServeMux()

	// Load Spotify credentials from environment variables. These are
	// required for authenticating with the Spotify API. If they are not
	// provided the server cannot run.
	clientID := os.Getenv("SPOTIFY_CLIENT_ID")
	clientSecret := os.Getenv("SPOTIFY_CLIENT_SECRET")
	// SPOTIFY_REDIRECT_URL must match the callback configured in the
	// Spotify developer dashboard. When unset we fall back to the local
	// development address.
	redirectURL := os.Getenv("SPOTIFY_REDIRECT_URL")
	if redirectURL == "" {
		redirectURL = "http://localhost:4000/callback"
	}
	signingKey := os.Getenv("SIGNING_KEY")
	if signingKey == "" {
		log.Fatal("SIGNING_KEY must be set")
	}
	// Validate that we have credentials. Without them the application is
	// unable to talk to Spotify so we exit early.
	if clientID == "" || clientSecret == "" {
		log.Fatal("SPOTIFY_CLIENT_ID and SPOTIFY_CLIENT_SECRET must be set")
	}

	// Initialize a Spotify client which will be used for unauthenticated
	// API requests (searching without a user token). This client is always
	// created as it may be needed for login flows even when another music
	// service is active.
	sc, err := spotify.NewSpotifyClient(clientID, clientSecret)
	if err != nil {
		log.Fatalf("spotify client init: %v", err)
	}
	var musicService music.Service = sc
	svc := os.Getenv("MUSIC_SERVICE")
	switch svc {
	case "youtube":
		key := os.Getenv("YOUTUBE_API_KEY")
		if key == "" {
			log.Fatal("YOUTUBE_API_KEY must be set when MUSIC_SERVICE=youtube")
		}
		musicService = &youtube.Client{Key: key, Client: &http.Client{Timeout: 10 * time.Second}}
	case "soundcloud":
		id := os.Getenv("SOUNDCLOUD_CLIENT_ID")
		if id == "" {
			log.Fatal("SOUNDCLOUD_CLIENT_ID must be set when MUSIC_SERVICE=soundcloud")
		}
		musicService = &soundcloud.Client{ClientID: id, HTTP: &http.Client{Timeout: 10 * time.Second}}
	case "aggregate":
		yKey := os.Getenv("YOUTUBE_API_KEY")
		scID := os.Getenv("SOUNDCLOUD_CLIENT_ID")
		if yKey == "" || scID == "" {
			log.Fatal("YOUTUBE_API_KEY and SOUNDCLOUD_CLIENT_ID must be set for aggregate service")
		}
		musicService = music.Aggregator{
			Services: []music.Service{
				sc,
				&youtube.Client{Key: yKey, Client: &http.Client{Timeout: 10 * time.Second}},
				&soundcloud.Client{ClientID: scID, HTTP: &http.Client{Timeout: 10 * time.Second}},
			},
			MaxConcurrent: 2,
		}
	}
	// The authenticator handles the OAuth flow for user specific
	// operations (like listing playlists). It needs the client credentials
	// and redirect URL configured above.
	auth := libspotify.NewAuthenticator(redirectURL, libspotify.ScopePlaylistReadPrivate)
	auth.SetAuthInfo(clientID, clientSecret)
	// DATABASE_PATH allows the SQLite file to be customised. It defaults
	// to a file named smartmusic.db in the working directory.
	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		dbPath = "smartmusic.db"
	}
	// Open the SQLite database which stores user tokens and favorites.
	database, err := db.New(dbPath)
	if err != nil {
		log.Fatalf("db init: %v", err)
	}
	defer database.Close()

	// Create the application struct which bundles the dependencies used by
	// our HTTP handlers.
	app := &handlers.Application{Music: musicService, SpotifyClient: sc, Authenticator: auth, DB: database, SignKey: []byte(signingKey)}

	// Register the application routes. Static assets are served from the
	// ui directory and all API endpoints are implemented in handlers.
	mux.HandleFunc("/", app.Home)
	mux.HandleFunc("/search", app.Search)
	mux.HandleFunc("/api/search", app.SearchJSON)
	mux.HandleFunc("/recommendations", app.Recommendations)
	mux.HandleFunc("/api/recommendations", app.RecommendationsJSON)
	mux.HandleFunc("/api/recommendations/mood", app.RecommendationsMood)
	mux.HandleFunc("/login", app.Login)
	mux.HandleFunc("/logout", app.Logout)
	mux.HandleFunc("/api/recommendations/advanced", app.RecommendationsAdvanced)
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
	mux.HandleFunc("/api/history", app.AddHistoryJSON)
	mux.HandleFunc("/api/collections", app.CreateCollectionJSON)
	mux.HandleFunc("/api/collections/", func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/tracks"):
			if r.Method == http.MethodPost {
				app.AddCollectionTrackJSON(w, r)
			} else {
				app.ListCollectionTracksJSON(w, r)
			}
		case strings.HasSuffix(r.URL.Path, "/users"):
			if r.Method == http.MethodPost {
				app.AddCollectionUserJSON(w, r)
			} else {
				http.NotFound(w, r)
			}
		default:
			http.NotFound(w, r)
		}
	})
	mux.HandleFunc("/api/insights", app.InsightsJSON)
	mux.HandleFunc("/api/insights/tracks", app.InsightsTracksJSON)
	mux.HandleFunc("/api/insights/monthly", app.InsightsMonthlyJSON)
	mux.Handle("/metrics", promhttp.Handler())
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./ui/static"))))
	mux.Handle("/app/", http.StripPrefix("/app/", http.FileServer(http.Dir("./ui/frontend/dist"))))

	// Determine the port to listen on. PORT may be set by the environment
	// (for example on cloud platforms). A leading colon is optional.
	port := os.Getenv("PORT")
	if port == "" {
		port = "4000"
	}
	if !strings.HasPrefix(port, ":") {
		port = ":" + port
	}

	// Finally start the HTTP server. ListenAndServe blocks and only returns
	// an error if the server fails to start or encounters a fatal error.
	if err := http.ListenAndServe(port, mux); err != nil {
		log.Fatalf("http server error: %v", err)
	}
}
