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

// Home is a simple handler function which writes a response.
// This will display a form on the home page where users can enter a track name and click on the "Search" button to search for the track.
func (app *Application) Home(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("ui/templates/index.html")
	if err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
		return
	}
	if err := tmpl.Execute(w, nil); err != nil {
		log.Printf("home template execute: %v", err)
		http.Error(w, "render error", http.StatusInternalServerError)
	}
}

/* In this function, we're getting the track query parameter from the request,
creating a new Spotify client, searching for the track, and printing the name of the first track found.
You'll need to replace "your-client-id" and "your-client-secret" with your actual Spotify application's client ID and secret.

This will display the name of the track, the name of the artist, and a link to listen to the track on Spotify.

This is a very basic implementation and there's a lot more you can do.
For example, you could add pagination to display more search results,
add more details about the tracks, handle errors more gracefully,
add a login system to allow users to save their favorite tracks, and much more.
The possibilities are endless! */

// Search is a handler function which will be used to handle search requests.
func (app *Application) Search(w http.ResponseWriter, r *http.Request) {
	// Get the query parameter for the track from the URL
	track := r.URL.Query().Get("track")

	// Try to use an authenticated client if a token cookie is present
	results, err := app.Spotify.SearchTrack(track)
	if c, errCookie := r.Cookie("spotify_token"); errCookie == nil {
		if t, errTok := decodeToken(c.Value); errTok == nil {
			client := app.Authenticator.NewClient(t)
			sr, errSearch := client.Search(track, libspotify.SearchTypeTrack)
			if errSearch == nil && sr.Tracks != nil && len(sr.Tracks.Tracks) > 0 {
				results = sr.Tracks.Tracks
				err = nil
			}
		}
	}

	// The SearchTrack function returns the tracks found and an error
	// If no tracks are found, the error will be "no tracks found"
	// If an error occurs during the search, it will be a different error
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

// decodeToken decodes a base64 encoded oauth2 token stored in a cookie
func decodeToken(v string) (*oauth2.Token, error) {
	data, err := base64.StdEncoding.DecodeString(v)
	if err != nil {
		return nil, err
	}
	var t oauth2.Token
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

// Login redirects the user to Spotify's OAuth authorization page
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

// OAuthCallback handles the redirect from Spotify and stores the token in a secure cookie
func (app *Application) OAuthCallback(w http.ResponseWriter, r *http.Request) {
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
	http.SetCookie(w, &http.Cookie{Name: "oauth_state", Path: "/", MaxAge: -1})

	token, err := app.Authenticator.Token(state, r)
	if err != nil {
		http.Error(w, "authentication failed", http.StatusInternalServerError)
		return
	}
	client := app.Authenticator.NewClient(token)
	user, err := client.CurrentUser()
	if err == nil && app.DB != nil {
		app.DB.SaveToken(user.ID, token)
	}
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

// Playlists displays the current user's playlists using an authenticated token
func (app *Application) Playlists(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("spotify_token")
	if err != nil {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	token, err := decodeToken(c.Value)
	if err != nil {
		http.Error(w, "invalid token", http.StatusBadRequest)
		return
	}
	client := app.Authenticator.NewClient(token)
	playlists, err := client.CurrentUsersPlaylists()
	if err != nil {
		http.Error(w, "failed to fetch playlists", http.StatusInternalServerError)
		return
	}
	tmpl, err := template.ParseFiles("ui/templates/playlists.html")
	if err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
		return
	}
	if err := tmpl.Execute(w, playlists); err != nil {
		http.Error(w, "render error", http.StatusInternalServerError)
	}
}

// AddFavorite saves a track to the user's favorites.
func (app *Application) AddFavorite(w http.ResponseWriter, r *http.Request) {
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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if app.DB == nil {
		http.Error(w, "db not configured", http.StatusInternalServerError)
		return
	}
	if err := app.DB.AddFavorite(userCookie.Value, req.TrackID, req.TrackName, req.ArtistName); err != nil {
		http.Error(w, "failed to save favorite", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

// Favorites displays the user's saved favorite tracks.
func (app *Application) Favorites(w http.ResponseWriter, r *http.Request) {
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
