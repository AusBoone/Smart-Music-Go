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
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"Smart-Music-Go/pkg/amazonmusic"
	"Smart-Music-Go/pkg/applemusic"
	"Smart-Music-Go/pkg/db"
	"Smart-Music-Go/pkg/handlers"
	"Smart-Music-Go/pkg/music"
	"Smart-Music-Go/pkg/soundcloud"
	"Smart-Music-Go/pkg/spotify"
	"Smart-Music-Go/pkg/tidal"
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
	case "applemusic":
		musicService = &applemusic.Client{HTTP: &http.Client{Timeout: 10 * time.Second}}
	case "tidal":
		token := os.Getenv("TIDAL_TOKEN")
		if token == "" {
			log.Fatal("TIDAL_TOKEN must be set when MUSIC_SERVICE=tidal")
		}
		cc := os.Getenv("TIDAL_COUNTRY_CODE")
		musicService = &tidal.Client{Token: token, CountryCode: cc, HTTP: &http.Client{Timeout: 10 * time.Second}}
	case "amazon":
		musicService = &amazonmusic.Client{}
	case "aggregate":
		var services []music.Service
		services = append(services, sc)
		if key := os.Getenv("YOUTUBE_API_KEY"); key != "" {
			services = append(services, &youtube.Client{Key: key, Client: &http.Client{Timeout: 10 * time.Second}})
		}
		if id := os.Getenv("SOUNDCLOUD_CLIENT_ID"); id != "" {
			services = append(services, &soundcloud.Client{ClientID: id, HTTP: &http.Client{Timeout: 10 * time.Second}})
		}
		services = append(services, &applemusic.Client{HTTP: &http.Client{Timeout: 10 * time.Second}})
		if token := os.Getenv("TIDAL_TOKEN"); token != "" {
			services = append(services, &tidal.Client{Token: token, CountryCode: os.Getenv("TIDAL_COUNTRY_CODE"), HTTP: &http.Client{Timeout: 10 * time.Second}})
		}
		services = append(services, &amazonmusic.Client{})
		musicService = music.Aggregator{Services: services, MaxConcurrent: 3}
	}
	// The authenticator handles the OAuth flow for user specific
	// operations (like listing playlists). It needs the client credentials
	// and redirect URL configured above.
	auth := libspotify.NewAuthenticator(redirectURL, libspotify.ScopePlaylistReadPrivate)
	auth.SetAuthInfo(clientID, clientSecret)

	googleID := os.Getenv("GOOGLE_CLIENT_ID")
	googleSecret := os.Getenv("GOOGLE_CLIENT_SECRET")
	googleRedirect := os.Getenv("GOOGLE_REDIRECT_URL")
	if googleRedirect == "" {
		googleRedirect = "http://localhost:4000/google/callback"
	}
	var googleConf *oauth2.Config
	if googleID != "" && googleSecret != "" {
		googleConf = &oauth2.Config{
			ClientID:     googleID,
			ClientSecret: googleSecret,
			RedirectURL:  googleRedirect,
			Scopes:       []string{"https://www.googleapis.com/auth/userinfo.profile"},
			Endpoint:     google.Endpoint,
		}
	}
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
	// Close the database when the server shuts down and log any error.
	defer func() {
		if err := database.Close(); err != nil {
			log.WithError(err).Error("database close")
		}
	}()

	// Create the application struct which bundles the dependencies used by
	// our HTTP handlers.
	app := &handlers.Application{Music: musicService, SpotifyClient: sc, Authenticator: auth, GoogleOAuth: googleConf, DB: database, SignKey: []byte(signingKey)}

	// Register the application routes. Static assets are served from the
	// ui directory and all API endpoints are implemented in handlers.
	mux.HandleFunc("/", app.Home)
	mux.HandleFunc("/search", app.Search)
	mux.HandleFunc("/api/search", app.SearchJSON)
	mux.HandleFunc("/recommendations", app.Recommendations)
	mux.HandleFunc("/api/recommendations", app.RecommendationsJSON)
	mux.HandleFunc("/api/recommendations/mood", app.RecommendationsMood)
	mux.HandleFunc("/login", app.Login)
	mux.HandleFunc("/login/google", app.GoogleLogin)
	mux.HandleFunc("/logout", app.Logout)
	mux.HandleFunc("/api/recommendations/advanced", app.RecommendationsAdvanced)
	mux.HandleFunc("/callback", app.OAuthCallback)
	mux.HandleFunc("/google/callback", app.GoogleCallback)
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
		switch r.Method {
		case http.MethodPost:
			app.AddFavorite(w, r)
		case http.MethodDelete:
			app.DeleteFavorite(w, r)
		default:
			app.FavoritesJSON(w, r)
		}
	})
	mux.HandleFunc("/api/favorites.csv", app.FavoritesCSV)
	mux.HandleFunc("/api/share/track", app.AddShareTrack)
	mux.HandleFunc("/api/share/playlist", app.AddSharePlaylist)
	mux.HandleFunc("/share/", app.ShareTrack)
	mux.HandleFunc("/share/playlist/", app.SharePlaylist)
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
	handler := handlers.SecurityHeaders(mux)

	// Finally start the HTTP server. ListenAndServe blocks and only returns
	// an error if the server fails to start or encounters a fatal error.
	if err := http.ListenAndServe(port, handler); err != nil {
		log.Fatalf("http server error: %v", err)
	}
}
