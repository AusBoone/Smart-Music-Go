// Package handlers provides HTTP handlers for Smart-Music-Go. This file contains
// endpoints that expose listening insights such as top artists and tracks.

package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

// InsightsJSON returns the most played artists for the last week.
func (app *Application) InsightsJSON(w http.ResponseWriter, r *http.Request) {
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
	since := time.Now().AddDate(0, 0, -7)
	res, err := app.DB.TopArtistsSince(r.Context(), userCookie.Value, since)
	if err != nil {
		http.Error(w, "failed to load insights", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(res); err != nil {
		log.WithError(err).Error("encode insights artists")
	}
}

// InsightsTracksJSON returns the most played tracks for a configurable period
// controlled by the 'days' query parameter.
func (app *Application) InsightsTracksJSON(w http.ResponseWriter, r *http.Request) {
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
	days, _ := strconv.Atoi(r.URL.Query().Get("days"))
	if days <= 0 {
		days = 7
	}
	since := time.Now().AddDate(0, 0, -days)
	res, err := app.DB.TopTracksSince(r.Context(), userCookie.Value, since)
	if err != nil {
		http.Error(w, "failed to load insights", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(res); err != nil {
		log.WithError(err).Error("encode insights tracks")
	}
}

// InsightsMonthlyJSON groups play counts by month starting from an optional
// 'since' query parameter (YYYY-MM-DD). If omitted a one year lookback is used.
func (app *Application) InsightsMonthlyJSON(w http.ResponseWriter, r *http.Request) {
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
	sinceStr := r.URL.Query().Get("since")
	since := time.Now().AddDate(-1, 0, 0)
	if sinceStr != "" {
		if t, err := time.Parse("2006-01-02", sinceStr); err == nil {
			since = t
		}
	}
	res, err := app.DB.MonthlyPlayCountsSince(r.Context(), userCookie.Value, since)
	if err != nil {
		http.Error(w, "failed to load insights", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(res); err != nil {
		log.WithError(err).Error("encode insights monthly")
	}
}
