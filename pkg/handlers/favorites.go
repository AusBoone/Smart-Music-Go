// Package handlers groups HTTP handlers for Smart-Music-Go. This file focuses
// on endpoints that manage user favorites, both the HTML page and the JSON API.
// Recent updates add CSV export so favorites can be shared easily. Splitting
// favorites logic keeps the handler implementations concise.

package handlers

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"html/template"
	"net/http"
)

// AddFavorite accepts a JSON payload describing a track and saves it to the
// current user's favorites list. The user ID is retrieved from a signed cookie.
func (app *Application) AddFavorite(w http.ResponseWriter, r *http.Request) {
	userID, ok := app.requireUser(w, r)
	if !ok {
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
	if err := app.DB.AddFavorite(r.Context(), userID, req.TrackID, req.TrackName, req.ArtistName); err != nil {
		http.Error(w, "failed to save favorite", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

// DeleteFavorite removes a track from the current user's favorites list. It
// expects a JSON body containing a track_id field. Missing or unsigned cookies
// result in a 401 response. A 404 status is returned when the favorite does not
// exist.
func (app *Application) DeleteFavorite(w http.ResponseWriter, r *http.Request) {
	userID, ok := app.requireUser(w, r)
	if !ok {
		return
	}
	if app.DB == nil {
		http.Error(w, "db not configured", http.StatusInternalServerError)
		return
	}
	var req struct {
		TrackID string `json:"track_id"`
	}
	if err := decodeJSON(r, &req); err != nil {
		respondJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.TrackID == "" {
		respondJSONError(w, http.StatusBadRequest, "track_id is required")
		return
	}
	var err error
	err = app.DB.DeleteFavorite(r.Context(), userID, req.TrackID)
	if err == sql.ErrNoRows {
		http.Error(w, "favorite not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "failed to delete favorite", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Favorites renders an HTML page listing tracks the user has marked as
// favorites.
func (app *Application) Favorites(w http.ResponseWriter, r *http.Request) {
	userID, ok := app.requireUser(w, r)
	if !ok {
		return
	}
	if app.DB == nil {
		http.Error(w, "db not configured", http.StatusInternalServerError)
		return
	}
	favs, err := app.DB.ListFavorites(r.Context(), userID)
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
	userID, ok := app.requireUser(w, r)
	if !ok {
		return
	}
	if app.DB == nil {
		http.Error(w, "db not configured", http.StatusInternalServerError)
		return
	}
	favs, err := app.DB.ListFavorites(r.Context(), userID)
	if err != nil {
		http.Error(w, "failed to load favorites", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(favs); err != nil {
		log.WithError(err).Error("encode favorites response")
	}
}

// FavoritesCSV exports the user's favorites in CSV format so they can be
// imported into other tools. The first row contains column headers.
func (app *Application) FavoritesCSV(w http.ResponseWriter, r *http.Request) {
	userID, ok := app.requireUser(w, r)
	if !ok {
		return
	}
	if app.DB == nil {
		http.Error(w, "db not configured", http.StatusInternalServerError)
		return
	}
	favs, err := app.DB.ListFavorites(r.Context(), userID)
	if err != nil {
		http.Error(w, "failed to load favorites", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/csv")
	enc := csv.NewWriter(w)
	if err := enc.Write([]string{"track_id", "track_name", "artist_name"}); err != nil {
		log.WithError(err).Error("csv header write")
		http.Error(w, "csv error", http.StatusInternalServerError)
		return
	}
	for _, f := range favs {
		if err := enc.Write([]string{f.TrackID, f.TrackName, f.ArtistName}); err != nil {
			log.WithError(err).Error("csv row write")
			http.Error(w, "csv error", http.StatusInternalServerError)
			return
		}
	}
	if err := enc.Flush(); err != nil {
		log.WithError(err).Error("csv flush")
	}
}
