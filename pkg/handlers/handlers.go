// Package handlers implements the HTTP endpoints for the Smart-Music-Go
// application. Handlers glue the web server to the Spotify client and
// persistence layer. Each function maps to a specific route and is designed
// to be used with the standard net/http mux. Templates located under
// ui/templates are loaded on demand. The functions here assume that template
// files exist and will return HTTP 500 when they cannot be loaded.
//
// The comments throughout this file document the reasoning behind key logic
// such as token refresh and cookie signing so developers unfamiliar with the
// project can quickly understand the flow.
//
// Updates: extended mood-based recommendation parameters, monthly insights API
// and collection collaboration endpoints were added to further separate the
// application from a basic Spotify wrapper.

package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	libspotify "github.com/zmb3/spotify"
	"golang.org/x/oauth2"

	"Smart-Music-Go/pkg/db"
	"Smart-Music-Go/pkg/music"
)

// Application struct holds dependencies for HTTP handlers.
type Application struct {
	// Music is the active music service implementation used for searching
	// and recommendations. It abstracts Spotify, YouTube or any future
	// provider behind a single interface.
	Music music.Service

	// SpotifyClient is optional and provides access to Spotify specific
	// features such as audio analysis used for mood-based playlists.
	// SpotifyClient exposes Spotify specific APIs used for advanced
	// recommendations. Tests replace this with a lightweight mock.
	SpotifyClient interface {
		GetRecommendationsWithAttrs(ctx context.Context, seedIDs []string, attrs *libspotify.TrackAttributes) ([]music.Track, error)
		GetAudioFeatures(ids ...string) ([]*libspotify.AudioFeatures, error)
	}

	Authenticator libspotify.Authenticator
	DB            *db.DB
	SignKey       []byte
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

	// Perform the search using the configured music service. This typically
	// uses client-credential flow and does not require a user session.
	results, err := app.Music.SearchTrack(r.Context(), track)
	var userID string
	if uc, errCookie := r.Cookie("spotify_user_id"); errCookie == nil {
		if v, ok := verifyValue(uc.Value, app.SignKey); ok {
			userID = v
		}
	}
	if c, errCookie := r.Cookie("spotify_token"); errCookie == nil {
		if v, ok := verifyValue(c.Value, app.SignKey); ok {
			if t, errTok := decodeToken(v); errTok == nil {
				if t, _ = app.refreshIfExpired(w, r, userID, t); t != nil {
					client := app.Authenticator.NewClient(t)
					sr, errSearch := client.Search(track, libspotify.SearchTypeTrack)
					if errSearch == nil && sr.Tracks != nil && len(sr.Tracks.Tracks) > 0 {
						results = sr.Tracks.Tracks
						err = nil
					}
				}
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

// Recommendations renders an HTML page listing recommended tracks for a
// provided seed track ID.
func (app *Application) Recommendations(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("track_id")
	if id == "" {
		http.Error(w, "missing track_id", http.StatusBadRequest)
		return
	}
	tracks, err := app.Music.GetRecommendations(r.Context(), []string{id})
	if err != nil {
		if err.Error() == "no recommendations found" {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else {
			http.Error(w, "recommendation error", http.StatusInternalServerError)
		}
		return
	}
	tmpl, err := template.ParseFiles("ui/templates/recommendations.html")
	if err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
		return
	}
	if err := tmpl.Execute(w, tracks); err != nil {
		http.Error(w, "render error", http.StatusInternalServerError)
	}
}

// RecommendationsMood returns recommendations using mood based on tempo,
// energy and danceability attributes. The track_id parameter is required.
// Optional query parameters min_tempo/max_tempo, min_energy/max_energy and
// min_dance/max_dance allow fine tuning. This handler only works when a
// Spotify client is configured.
func (app *Application) RecommendationsMood(w http.ResponseWriter, r *http.Request) {
	if app.SpotifyClient == nil {
		http.Error(w, "spotify service required", http.StatusNotImplemented)
		return
	}
	id := r.URL.Query().Get("track_id")
	if id == "" {
		http.Error(w, "missing track_id", http.StatusBadRequest)
		return
	}
	minTempo, _ := strconv.ParseFloat(r.URL.Query().Get("min_tempo"), 64)
	maxTempo, _ := strconv.ParseFloat(r.URL.Query().Get("max_tempo"), 64)
	minEnergy, _ := strconv.ParseFloat(r.URL.Query().Get("min_energy"), 64)
	maxEnergy, _ := strconv.ParseFloat(r.URL.Query().Get("max_energy"), 64)
	minDance, _ := strconv.ParseFloat(r.URL.Query().Get("min_dance"), 64)
	maxDance, _ := strconv.ParseFloat(r.URL.Query().Get("max_dance"), 64)
	feats, err := app.SpotifyClient.GetAudioFeatures(id)
	if err != nil || len(feats) == 0 {
		http.Error(w, "audio feature error", http.StatusInternalServerError)
		return
	}
	if minTempo == 0 && maxTempo == 0 {
		// Default to +-10 BPM around the seed track tempo
		minTempo = float64(feats[0].Tempo) - 10
		maxTempo = float64(feats[0].Tempo) + 10
	}
	attrs := libspotify.NewTrackAttributes().MinTempo(minTempo).MaxTempo(maxTempo)
	if minEnergy > 0 {
		attrs = attrs.MinEnergy(minEnergy)
	}
	if maxEnergy > 0 {
		attrs = attrs.MaxEnergy(maxEnergy)
	}
	if minDance > 0 {
		attrs = attrs.MinDanceability(minDance)
	}
	if maxDance > 0 {
		attrs = attrs.MaxDanceability(maxDance)
	}
	tracks, err := app.SpotifyClient.GetRecommendationsWithAttrs(r.Context(), []string{id}, attrs)
	if err != nil {
		if err.Error() == "no recommendations found" {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else {
			http.Error(w, "recommendation error", http.StatusInternalServerError)
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tracks)
}

// SearchJSON handles the /api/search endpoint and writes search results as
// JSON.  Parameters mirror those of http.HandlerFunc.
func (app *Application) SearchJSON(w http.ResponseWriter, r *http.Request) {
	// Grab the requested track from the query string.
	track := r.URL.Query().Get("track")
	// Start with a search using the active music service.
	results, err := app.Music.SearchTrack(r.Context(), track)
	var userID string
	if uc, errCookie := r.Cookie("spotify_user_id"); errCookie == nil {
		if v, ok := verifyValue(uc.Value, app.SignKey); ok {
			userID = v
		}
	}
	if c, errCookie := r.Cookie("spotify_token"); errCookie == nil {
		if v, ok := verifyValue(c.Value, app.SignKey); ok {
			if t, errTok := decodeToken(v); errTok == nil {
				if t, _ = app.refreshIfExpired(w, r, userID, t); t != nil {
					client := app.Authenticator.NewClient(t)
					sr, errSearch := client.Search(track, libspotify.SearchTypeTrack)
					if errSearch == nil && sr.Tracks != nil && len(sr.Tracks.Tracks) > 0 {
						results = sr.Tracks.Tracks
						err = nil
					}
				}
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

// Recommendations handles /api/recommendations and returns track suggestions as JSON.
func (app *Application) RecommendationsJSON(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("track_id")
	if id == "" {
		http.Error(w, "missing track_id", http.StatusBadRequest)
		return
	}
	tracks, err := app.Music.GetRecommendations(r.Context(), []string{id})
	if err != nil {
		if err.Error() == "no recommendations found" {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else {
			http.Error(w, "recommendation error", http.StatusInternalServerError)
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tracks)
}

// Playlists renders an HTML page listing the logged-in user's playlists. It
// requires a valid authentication cookie.
func (app *Application) Playlists(w http.ResponseWriter, r *http.Request) {
	userCookie, err := r.Cookie("spotify_user_id")
	if err != nil {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	if v, ok := verifyValue(userCookie.Value, app.SignKey); ok {
		userCookie.Value = v
	} else {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	var token *oauth2.Token
	if c, err := r.Cookie("spotify_token"); err == nil {
		if v, ok := verifyValue(c.Value, app.SignKey); ok {
			if t, errTok := decodeToken(v); errTok == nil {
				token, _ = app.refreshIfExpired(w, r, userCookie.Value, t)
			}
		}
	}
	if token == nil && app.DB != nil {
		if t, errTok := app.DB.GetToken(userCookie.Value); errTok == nil {
			token, _ = app.refreshIfExpired(w, r, userCookie.Value, t)
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
	if v, ok := verifyValue(userCookie.Value, app.SignKey); ok {
		userCookie.Value = v
	} else {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	var token *oauth2.Token
	if c, err := r.Cookie("spotify_token"); err == nil {
		if v, ok := verifyValue(c.Value, app.SignKey); ok {
			if t, errTok := decodeToken(v); errTok == nil {
				token, _ = app.refreshIfExpired(w, r, userCookie.Value, t)
			}
		}
	}
	if token == nil && app.DB != nil {
		if t, errTok := app.DB.GetToken(userCookie.Value); errTok == nil {
			token, _ = app.refreshIfExpired(w, r, userCookie.Value, t)
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
// AddHistoryJSON records that a track was played. It expects a JSON body with
// track_id and artist_name fields. The current timestamp is stored.
func (app *Application) AddHistoryJSON(w http.ResponseWriter, r *http.Request) {
	userCookie, err := r.Cookie("spotify_user_id")
	if err != nil {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	if v, ok := verifyValue(userCookie.Value, app.SignKey); ok {
		userCookie.Value = v
	} else {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	if app.DB == nil {
		http.Error(w, "db not configured", http.StatusInternalServerError)
		return
	}
	var req struct {
		TrackID    string `json:"track_id"`
		ArtistName string `json:"artist_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if err := app.DB.AddHistory(userCookie.Value, req.TrackID, req.ArtistName, time.Now()); err != nil {
		http.Error(w, "failed to save history", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

// CreateCollectionJSON starts a new collaborative playlist owned by the user.
func (app *Application) CreateCollectionJSON(w http.ResponseWriter, r *http.Request) {
	userCookie, err := r.Cookie("spotify_user_id")
	if err != nil {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	if v, ok := verifyValue(userCookie.Value, app.SignKey); ok {
		userCookie.Value = v
	} else {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	if app.DB == nil {
		http.Error(w, "db not configured", http.StatusInternalServerError)
		return
	}
	id, err := app.DB.CreateCollection(userCookie.Value)
	if err != nil {
		http.Error(w, "failed to create collection", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"id": id})
}

// AddCollectionTrackJSON appends a track to an existing collection.
func (app *Application) AddCollectionTrackJSON(w http.ResponseWriter, r *http.Request) {
	userCookie, err := r.Cookie("spotify_user_id")
	if err != nil {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	if v, ok := verifyValue(userCookie.Value, app.SignKey); ok {
		userCookie.Value = v
	} else {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	colID := strings.TrimPrefix(r.URL.Path, "/api/collections/")
	colID = strings.TrimSuffix(colID, "/tracks")
	if colID == "" {
		http.Error(w, "missing collection id", http.StatusBadRequest)
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
	if err := app.DB.AddTrackToCollection(colID, req.TrackID, req.TrackName, req.ArtistName); err != nil {
		http.Error(w, "failed to add track", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

// AddCollectionUserJSON associates another user with the specified collection
// so they can contribute tracks. The body must include a user_id field.
func (app *Application) AddCollectionUserJSON(w http.ResponseWriter, r *http.Request) {
	userCookie, err := r.Cookie("spotify_user_id")
	if err != nil {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	if v, ok := verifyValue(userCookie.Value, app.SignKey); ok {
		userCookie.Value = v
	} else {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return
	}
	colID := strings.TrimPrefix(r.URL.Path, "/api/collections/")
	colID = strings.TrimSuffix(colID, "/users")
	if colID == "" {
		http.Error(w, "missing collection id", http.StatusBadRequest)
		return
	}
	var req struct {
		UserID string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.UserID == "" {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if app.DB == nil {
		http.Error(w, "db not configured", http.StatusInternalServerError)
		return
	}
	if err := app.DB.AddUserToCollection(colID, req.UserID); err != nil {
		http.Error(w, "failed to add user", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

// ListCollectionTracksJSON returns tracks for a given collection.
func (app *Application) ListCollectionTracksJSON(w http.ResponseWriter, r *http.Request) {
	colID := strings.TrimPrefix(r.URL.Path, "/api/collections/")
	colID = strings.TrimSuffix(colID, "/tracks")
	if colID == "" {
		http.Error(w, "missing collection id", http.StatusBadRequest)
		return
	}
	if app.DB == nil {
		http.Error(w, "db not configured", http.StatusInternalServerError)
		return
	}
	tracks, err := app.DB.ListCollectionTracks(colID)
	if err != nil {
		http.Error(w, "failed to list tracks", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tracks)
}

// RecommendationsAdvanced allows specifying additional audio features for recommendations.
// Supported query parameters include min_energy and min_valence.
func (app *Application) RecommendationsAdvanced(w http.ResponseWriter, r *http.Request) {
	if app.SpotifyClient == nil {
		http.Error(w, "spotify service required", http.StatusNotImplemented)
		return
	}
	id := r.URL.Query().Get("track_id")
	if id == "" {
		http.Error(w, "missing track_id", http.StatusBadRequest)
		return
	}
	attrs := libspotify.NewTrackAttributes()
	if v, err := strconv.ParseFloat(r.URL.Query().Get("min_energy"), 64); err == nil && v > 0 {
		attrs = attrs.MinEnergy(v)
	}
	if v, err := strconv.ParseFloat(r.URL.Query().Get("min_valence"), 64); err == nil && v > 0 {
		attrs = attrs.MinValence(v)
	}
	tracks, err := app.SpotifyClient.GetRecommendationsWithAttrs(r.Context(), []string{id}, attrs)
	if err != nil {
		if err.Error() == "no recommendations found" {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else {
			http.Error(w, "recommendation error", http.StatusInternalServerError)
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tracks)
}
