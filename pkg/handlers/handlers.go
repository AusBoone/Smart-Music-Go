// This file will contain the HTTP handlers that respond to web requests.

package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"

	libspotify "github.com/zmb3/spotify"
	"golang.org/x/oauth2"

	"Smart-Music-Go/pkg/db"
	"Smart-Music-Go/pkg/spotify"
)

// Application struct holds dependencies for HTTP handlers.
type Application struct {
	Spotify       spotify.TrackSearcher
	Authenticator libspotify.Authenticator
	DB            *db.DB
}

// Home renders the landing page.  It shows the search form where users can
// enter a track name.
func (app *Application) Home(w http.ResponseWriter, r *http.Request) {
	// Load the HTML template for the home page.
	// Reload the template to display the search results. In a larger
	// application you might parse templates once during startup instead.
	tmpl, err := template.ParseFiles("ui/templates/index.html")
	if err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
		return
	}
	if err := tmpl.Execute(w, nil); err != nil {
		// Log the error but return a generic message to the client
		// so we do not expose internal details.
		log.Printf("home template execute: %v", err)
		http.Error(w, "render error", http.StatusInternalServerError)
	}
}

// Search handles the /search endpoint.  It looks up the "track" query
// parameter and renders search results.  The ResponseWriter and Request are
// the standard HTTP interfaces used by net/http.
func (app *Application) Search(w http.ResponseWriter, r *http.Request) {
	// Extract the "track" query parameter from the URL. If it is empty the
	// user submitted the form without entering a search term.
	track := r.URL.Query().Get("track")

	// Perform the search using the application level Spotify client. This
	// uses the client credentials flow which does not require a user
	// session.
	results, err := app.Spotify.SearchTrack(track)
	if c, errCookie := r.Cookie("spotify_token"); errCookie == nil {
		// If the user has authenticated, retry the search with their
		// personal token to return personalized results.
		if t, errTok := decodeToken(c.Value); errTok == nil {
			client := app.Authenticator.NewClient(t)
			sr, errSearch := client.Search(track, libspotify.SearchTypeTrack)
			if errSearch == nil && sr.Tracks != nil && len(sr.Tracks.Tracks) > 0 {
				results = sr.Tracks.Tracks
				err = nil
			}
		}
	}

	// The SearchTrack function returns the tracks found and an error. If no
	// tracks match, the error will be "no tracks found" which we treat
	// differently from other failures.
	if err != nil {
		// If the error is "no tracks found", respond with a user-friendly message
		if err.Error() == "no tracks found" {
			fmt.Fprintf(w, "No tracks found for '%s'", track)
		} else {
			// If a different error occurs, respond with a generic server error message
			http.Error(w, "An error occurred while searching for tracks", http.StatusInternalServerError)
		}
		// Stop processing the request
		return
	}

	// Parse the index template used for both home and search results
	// The ParseFiles function returns a *Template and an error
	// If an error occurs while loading the template, it will be a different error
	tmpl, err := template.ParseFiles("ui/templates/index.html")
	if err != nil {
		// If an error occurs while loading the template, respond with a generic server error message
		http.Error(w, "An error occurred while loading the template", http.StatusInternalServerError)
		// Stop processing the request
		return
	}

	// Render the template with the search results
	// The Execute function writes the rendered template to the http.ResponseWriter
	// If an error occurs while rendering the template, it will be a different error
	// Build the template data structure. Using an anonymous struct keeps
	// things simple for this small application.
	data := struct {
		Results []libspotify.FullTrack
	}{Results: results}

	err = tmpl.Execute(w, data)
	if err != nil {
		// If an error occurs while rendering the template, respond with a generic server error message
		http.Error(w, "An error occurred while rendering the template", http.StatusInternalServerError)
		// Stop processing the request
		return
	}
}

// SearchJSON handles the /api/search endpoint and writes search results as
// JSON.  Parameters mirror those of http.HandlerFunc.
func (app *Application) SearchJSON(w http.ResponseWriter, r *http.Request) {
	// Grab the requested track from the query string.
	track := r.URL.Query().Get("track")
	// Start with a search using the application client.
	results, err := app.Spotify.SearchTrack(track)
	if c, errCookie := r.Cookie("spotify_token"); errCookie == nil {
		// If the user is logged in, attempt the search again using
		// their access token for potentially richer results.
		if t, errTok := decodeToken(c.Value); errTok == nil {
			client := app.Authenticator.NewClient(t)
			sr, errSearch := client.Search(track, libspotify.SearchTypeTrack)
			if errSearch == nil && sr.Tracks != nil && len(sr.Tracks.Tracks) > 0 {
				results = sr.Tracks.Tracks
				err = nil
			}
		}
	}
	if err != nil {
		if err.Error() == "no tracks found" {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else {
			http.Error(w, "search error", http.StatusInternalServerError)
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	// Encode the results to JSON for the client.
	json.NewEncoder(w).Encode(results)
}

// decodeToken converts the cookie stored token back into an oauth2.Token. It
// expects the cookie value to be base64 encoded JSON.
func decodeToken(v string) (*oauth2.Token, error) {
	// Decode the base64 encoded string stored in the cookie back into the
	// JSON representation of the token.
	data, err := base64.StdEncoding.DecodeString(v)
	if err != nil {
		return nil, err
	}
	// Unmarshal the JSON into the oauth2.Token struct.
	var t oauth2.Token
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

// Login begins the OAuth flow by redirecting the user to Spotify's
// authorization page.  The ResponseWriter and Request come from net/http.
func (app *Application) Login(w http.ResponseWriter, r *http.Request) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		http.Error(w, "failed to generate state", http.StatusInternalServerError)
		return
	}
	state := base64.RawURLEncoding.EncodeToString(b)
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil,
	})
	http.Redirect(w, r, app.Authenticator.AuthURL(state), http.StatusFound)
}

// OAuthCallback completes the OAuth flow.  It exchanges the authorization code
// for a token and stores it in secure cookies.
func (app *Application) OAuthCallback(w http.ResponseWriter, r *http.Request) {
	// Retrieve and verify the state cookie to protect against CSRF attacks.
	c, err := r.Cookie("oauth_state")
	if err != nil {
		http.Error(w, "state mismatch", http.StatusBadRequest)
		return
	}
	state := c.Value
	if r.URL.Query().Get("state") != state {
		http.Error(w, "state mismatch", http.StatusBadRequest)
		return
	}
	// Delete the state cookie as it is no longer needed.
	http.SetCookie(w, &http.Cookie{Name: "oauth_state", Path: "/", MaxAge: -1})

	// Exchange the authorization code for an access token.
	token, err := app.Authenticator.Token(state, r)
	if err != nil {
		http.Error(w, "authentication failed", http.StatusInternalServerError)
		return
	}
	// Look up the current Spotify user so we can persist their token.
	client := app.Authenticator.NewClient(token)
	user, err := client.CurrentUser()
	if err == nil && app.DB != nil {
		app.DB.SaveToken(user.ID, token)
	}
	// Store the token and user ID in cookies for later requests.
	b, _ := json.Marshal(token)
	cookie := &http.Cookie{
		Name:     "spotify_token",
		Value:    base64.StdEncoding.EncodeToString(b),
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil,
	}
	http.SetCookie(w, cookie)
	if user != nil {
		http.SetCookie(w, &http.Cookie{Name: "spotify_user_id", Value: user.ID, Path: "/"})
	}
	http.Redirect(w, r, "/", http.StatusFound)
}

// Playlists renders an HTML page listing the logged-in user's playlists. It
// requires a valid authentication cookie.
func (app *Application) Playlists(w http.ResponseWriter, r *http.Request) {
	userCookie, err := r.Cookie("spotify_user_id")
	if err != nil {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	var token *oauth2.Token
	if c, err := r.Cookie("spotify_token"); err == nil {
		if t, errTok := decodeToken(c.Value); errTok == nil && t.Valid() {
			token = t
		}
	}
	if token == nil && app.DB != nil {
		if t, errTok := app.DB.GetToken(userCookie.Value); errTok == nil {
			token = t
		} else {
			http.Error(w, "authentication required", http.StatusUnauthorized)
			return
		}
	}
	// Use the authenticated client to fetch the user's playlists.
	client := app.Authenticator.NewClient(token)
	playlists, err := client.CurrentUsersPlaylists()
	if err != nil {
		http.Error(w, "failed to fetch playlists", http.StatusInternalServerError)
		return
	}
	// Load the template that displays the playlists.
	tmpl, err := template.ParseFiles("ui/templates/playlists.html")
	if err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
		return
	}
	// Write the rendered HTML to the response writer.
	if err := tmpl.Execute(w, playlists); err != nil {
		http.Error(w, "render error", http.StatusInternalServerError)
	}
}

// PlaylistsJSON handles /api/playlists and returns the playlists encoded as
// JSON.
func (app *Application) PlaylistsJSON(w http.ResponseWriter, r *http.Request) {
	userCookie, err := r.Cookie("spotify_user_id")
	if err != nil {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	var token *oauth2.Token
	if c, err := r.Cookie("spotify_token"); err == nil {
		if t, errTok := decodeToken(c.Value); errTok == nil && t.Valid() {
			token = t
		}
	}
	if token == nil && app.DB != nil {
		if t, errTok := app.DB.GetToken(userCookie.Value); errTok == nil {
			token = t
		} else {
			http.Error(w, "authentication required", http.StatusUnauthorized)
			return
		}
	}
	client := app.Authenticator.NewClient(token)
	playlists, err := client.CurrentUsersPlaylists()
	if err != nil {
		http.Error(w, "failed to fetch playlists", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(playlists)
}

// AddFavorite accepts a JSON payload describing a track and stores it in the
// logged-in user's favorites list.
func (app *Application) AddFavorite(w http.ResponseWriter, r *http.Request) {
	// Identify the current user via the ID stored in a cookie.
	userCookie, err := r.Cookie("spotify_user_id")
	if err != nil {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	var req struct {
		TrackID    string `json:"track_id"`
		TrackName  string `json:"track_name"`
		ArtistName string `json:"artist_name"`
	}
	// Decode the JSON payload describing the track to favorite.
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if app.DB == nil {
		http.Error(w, "db not configured", http.StatusInternalServerError)
		return
	}
	// Store the favorite in the database keyed by the user ID.
	if err := app.DB.AddFavorite(userCookie.Value, req.TrackID, req.TrackName, req.ArtistName); err != nil {
		http.Error(w, "failed to save favorite", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

// Favorites renders an HTML page listing tracks the user has saved as
// favorites.
func (app *Application) Favorites(w http.ResponseWriter, r *http.Request) {
	// Obtain the user ID from the cookie so we can query their favorites.
	userCookie, err := r.Cookie("spotify_user_id")
	if err != nil {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	if app.DB == nil {
		http.Error(w, "db not configured", http.StatusInternalServerError)
		return
	}
	// Look up the user's favorites from the database.
	favs, err := app.DB.ListFavorites(userCookie.Value)
	if err != nil {
		http.Error(w, "failed to load favorites", http.StatusInternalServerError)
		return
	}
	// Load the HTML template that displays the favorites.
	tmpl, err := template.ParseFiles("ui/templates/favorites.html")
	if err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
		return
	}
	if err := tmpl.Execute(w, favs); err != nil {
		log.Printf("favorites template execute: %v", err)
		http.Error(w, "render error", http.StatusInternalServerError)
	}
}

// FavoritesJSON serves the /api/favorites endpoint and returns the favorites
// as JSON.
func (app *Application) FavoritesJSON(w http.ResponseWriter, r *http.Request) {
	// Authenticate and find the user's favorites, returning them as JSON.
	userCookie, err := r.Cookie("spotify_user_id")
	if err != nil {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	if app.DB == nil {
		http.Error(w, "db not configured", http.StatusInternalServerError)
		return
	}
	favs, err := app.DB.ListFavorites(userCookie.Value)
	if err != nil {
		http.Error(w, "failed to load favorites", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(favs)
}
