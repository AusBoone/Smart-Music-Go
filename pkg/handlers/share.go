// Package handlers includes HTTP handlers for Smart-Music-Go. This file contains
// the endpoints responsible for creating and displaying shareable links for
// tracks and playlists. Each share is stored with a short ID allowing anyone
// with the link to view the content without authentication. The pages rendered
// are deliberately minimal so they can be embedded in social media previews.
package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strings"
)

// AddShareTrack creates a shareable link for a track. It requires the caller to
// be authenticated with Google so the share can be associated with a user.
// The request body should contain JSON fields `track_id`, `track_name` and
// `artist_name`. On success a JSON object containing the full URL is returned.
func (app *Application) AddShareTrack(w http.ResponseWriter, r *http.Request) {
	_, ok := app.requireGoogleUser(w, r)
	if !ok {
		return
	}
	if app.DB == nil {
		http.Error(w, "db not configured", http.StatusInternalServerError)
		return
	}
	var req struct {
		TrackID    string `json:"track_id"`
		TrackName  string `json:"track_name"`
		ArtistName string `json:"artist_name"`
	}
	if err := decodeJSON(r, &req); err != nil {
		respondJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.TrackID == "" || req.TrackName == "" || req.ArtistName == "" {
		respondJSONError(w, http.StatusBadRequest, "track_id, track_name and artist_name are required")
		return
	}
	// Persist the share details and generate a short identifier. This ID is
	// used by the view handler to retrieve the record.
	id, err := app.DB.CreateShareTrack(r.Context(), req.TrackID, req.TrackName, req.ArtistName)
	if err != nil {
		http.Error(w, "failed to store share", http.StatusInternalServerError)
		return
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	url := fmt.Sprintf("%s://%s/share/%s", scheme, r.Host, id)
	json.NewEncoder(w).Encode(map[string]string{"url": url})
}

// ShareTrack renders a small HTML page describing a shared track. The share ID
// is extracted from the path `/share/{id}`. If the ID is missing from the
// database a 404 response is returned.
func (app *Application) ShareTrack(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/share/")
	if app.DB == nil {
		http.Error(w, "db not configured", http.StatusInternalServerError)
		return
	}
	// Retrieve the track metadata using the ID from the URL.
	st, err := app.DB.GetShareTrack(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			http.NotFound(w, r)
		} else {
			http.Error(w, "failed to load share", http.StatusInternalServerError)
		}
		return
	}
	tmpl, err := template.ParseFiles("ui/templates/share.html")
	if err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
		return
	}
	data := struct {
		TrackName  string
		ArtistName string
		TrackID    string
	}{st.TrackName, st.ArtistName, st.TrackID}
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "render error", http.StatusInternalServerError)
	}
}

// AddSharePlaylist generates a share link for the given playlist. The handler
// expects `playlist_id` and `playlist_name` in the request body and requires a
// signed-in Google user. A JSON response containing the URL is returned.
func (app *Application) AddSharePlaylist(w http.ResponseWriter, r *http.Request) {
	_, ok := app.requireGoogleUser(w, r)
	if !ok {
		return
	}
	if app.DB == nil {
		http.Error(w, "db not configured", http.StatusInternalServerError)
		return
	}
	var req struct {
		PlaylistID   string `json:"playlist_id"`
		PlaylistName string `json:"playlist_name"`
	}
	if err := decodeJSON(r, &req); err != nil {
		respondJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.PlaylistID == "" || req.PlaylistName == "" {
		respondJSONError(w, http.StatusBadRequest, "playlist_id and playlist_name are required")
		return
	}
	id, err := app.DB.CreateSharePlaylist(r.Context(), req.PlaylistID, req.PlaylistName)
	if err != nil {
		http.Error(w, "failed to store share", http.StatusInternalServerError)
		return
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	url := fmt.Sprintf("%s://%s/share/playlist/%s", scheme, r.Host, id)
	json.NewEncoder(w).Encode(map[string]string{"url": url})
}

// SharePlaylist displays an HTML page describing a shared playlist. The ID is
// taken from the path `/share/playlist/{id}`. Missing entries result in a 404
// response to keep links unguessable.
func (app *Application) SharePlaylist(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/share/playlist/")
	if app.DB == nil {
		http.Error(w, "db not configured", http.StatusInternalServerError)
		return
	}
	// Look up the playlist referenced by the share ID. Errors include
	// sql.ErrNoRows when the link does not exist.
	sp, err := app.DB.GetSharePlaylist(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			http.NotFound(w, r)
		} else {
			http.Error(w, "failed to load share", http.StatusInternalServerError)
		}
		return
	}
	tmpl, err := template.ParseFiles("ui/templates/share_playlist.html")
	if err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
		return
	}
	data := struct {
		PlaylistName string
		PlaylistID   string
	}{sp.PlaylistName, sp.PlaylistID}
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "render error", http.StatusInternalServerError)
	}
}
