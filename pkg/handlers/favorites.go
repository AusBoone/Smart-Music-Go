// Package handlers groups HTTP handlers for Smart-Music-Go. This file focuses
// on endpoints that manage user favorites, both the HTML page and the JSON API.
// Splitting favorites logic keeps the handler implementations concise.

package handlers

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"html/template"
	"net/http"
)

// AddFavorite accepts a JSON payload describing a track and saves it to the
// current user's favorites list. The user ID is retrieved from a signed cookie.
func (app *Application) AddFavorite(w http.ResponseWriter, r *http.Request) {
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
	if app.DB == nil {
		http.Error(w, "db not configured", http.StatusInternalServerError)
		return
	}
	if err := app.DB.AddFavorite(r.Context(), userCookie.Value, req.TrackID, req.TrackName, req.ArtistName); err != nil {
		http.Error(w, "failed to save favorite", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

// Favorites renders an HTML page listing tracks the user has marked as
// favorites.
func (app *Application) Favorites(w http.ResponseWriter, r *http.Request) {
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
	favs, err := app.DB.ListFavorites(r.Context(), userCookie.Value)
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
		log.WithError(err).Error("favorites template execute")
		http.Error(w, "render error", http.StatusInternalServerError)
	}
}

// FavoritesJSON returns the user's favorites as JSON for consumption by the
// frontend.
func (app *Application) FavoritesJSON(w http.ResponseWriter, r *http.Request) {
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
	favs, err := app.DB.ListFavorites(r.Context(), userCookie.Value)
	if err != nil {
		http.Error(w, "failed to load favorites", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(favs)
}
