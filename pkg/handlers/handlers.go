// This file will contain the HTTP handlers that respond to web requests.

package handlers

import (
	"fmt"
	"html/template"
	"net/http"
	"os"

	"Smart-Music-Go/pkg/spotify"
)

// Application struct to hold the methods for routes
type Application struct{}

// Home is a simple handler function which writes a response.
// This will display a form on the home page where users can enter a track name and click on the "Search" button to search for the track.
func (app *Application) Home(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, `
		<h1>Welcome to Smart-Music-Go!</h1>
		<form action="/search" method="get">
			<input type="text" name="track" placeholder="Enter a track name">
			<button type="submit">Search</button>
		</form>
	`)
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

	// Read credentials from environment variables
	clientID := os.Getenv("SPOTIFY_CLIENT_ID")
	clientSecret := os.Getenv("SPOTIFY_CLIENT_SECRET")
	if clientID == "" || clientSecret == "" {
		http.Error(w, "Spotify credentials not configured", http.StatusInternalServerError)
		return
	}

	// Create a new Spotify client using the credentials
	sc, err := spotify.NewSpotifyClient(clientID, clientSecret)
	if err != nil {
		http.Error(w, "Failed to authenticate with Spotify", http.StatusInternalServerError)
		return
	}

	// Use the Spotify client to search for the track
	// The SearchTrack function returns the first track found and an error
	// If no tracks are found, the error will be "no tracks found"
	// If an error occurs during the search, it will be a different error
	result, err := sc.SearchTrack(track)
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

	// Parse the "search_results.html" template
	// The ParseFiles function returns a *Template and an error
	// If an error occurs while loading the template, it will be a different error
	tmpl, err := template.ParseFiles("ui/templates/search_results.html")
	if err != nil {
		// If an error occurs while loading the template, respond with a generic server error message
		http.Error(w, "An error occurred while loading the template", http.StatusInternalServerError)
		// Stop processing the request
		return
	}

	// Render the template with the search results
	// The Execute function writes the rendered template to the http.ResponseWriter
	// If an error occurs while rendering the template, it will be a different error
	err = tmpl.Execute(w, result)
	if err != nil {
		// If an error occurs while rendering the template, respond with a generic server error message
		http.Error(w, "An error occurred while rendering the template", http.StatusInternalServerError)
		// Stop processing the request
		return
	}
}
