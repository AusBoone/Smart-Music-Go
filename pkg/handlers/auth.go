// Package handlers contains HTTP handlers for Smart-Music-Go. This file groups
// authentication related helpers and endpoints such as the OAuth login and
// callback handlers. Keeping these routines separate from the main handlers
// file improves readability and maintainability. CSRF protection is implemented
// using a random token stored in a cookie which clients must echo back in the
// `X-CSRF-Token` header for all state changing requests.

package handlers

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"golang.org/x/oauth2"
)

// signValue computes an HMAC signature for value and appends it using the
// format value|signature. The signature is base64 URL encoded so it can be
// safely stored in cookies.
func signValue(value string, key []byte) string {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(value))
	sig := mac.Sum(nil)
	return value + "|" + base64.RawURLEncoding.EncodeToString(sig)
}

// verifyValue checks the HMAC signature appended to signed. It returns the
// original value and true when the signature matches the provided key.
func verifyValue(signed string, key []byte) (string, bool) {
	parts := strings.Split(signed, "|")
	if len(parts) != 2 {
		return "", false
	}
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(parts[0]))
	expected := mac.Sum(nil)
	sig, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || !hmac.Equal(expected, sig) {
		return "", false
	}
	return parts[0], true
}

// setCSRFToken generates a new random token and sets it in a cookie. The token
// is returned so handlers can also include it in the response body if needed.
// The cookie is not HttpOnly so client-side scripts can read the value and
// attach it to subsequent requests.
func setCSRFToken(w http.ResponseWriter, secure bool) (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := base64.RawURLEncoding.EncodeToString(b)
	http.SetCookie(w, &http.Cookie{
		Name:     "csrf_token",
		Value:    token,
		Path:     "/",
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
	return token, nil
}

// verifyCSRF compares the X-CSRF-Token header with the csrf_token cookie. The
// comparison is constant time to avoid timing attacks. It returns true when the
// values match and are present.
func verifyCSRF(r *http.Request) bool {
	c, err := r.Cookie("csrf_token")
	if err != nil {
		return false
	}
	header := r.Header.Get("X-CSRF-Token")
	if header == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(c.Value), []byte(header)) == 1
}

// userFromCookie returns the verified Spotify user ID from the request cookie.
// An error is returned when the cookie is missing or has been tampered with.
func (app *Application) userFromCookie(r *http.Request) (string, error) {
	c, err := r.Cookie("spotify_user_id")
	if err != nil {
		return "", err
	}
	if v, ok := verifyValue(c.Value, app.SignKey); ok {
		return v, nil
	}
	return "", fmt.Errorf("invalid signature")
}

// requireUser is a helper used by handlers to enforce authentication. It
// writes a 401 response on failure and returns the user ID otherwise.
func (app *Application) requireUser(w http.ResponseWriter, r *http.Request) (string, bool) {
	id, err := app.userFromCookie(r)
	if err != nil {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return "", false
	}
	// Enforce CSRF protection on state-changing requests.
	if r.Method != http.MethodGet && r.Method != http.MethodHead && !verifyCSRF(r) {
		http.Error(w, "invalid csrf token", http.StatusForbidden)
		return "", false
	}
	return id, true
}

// userFromGoogleCookie returns the Google account ID from the signed cookie.
// It mirrors userFromCookie but uses the google_user_id cookie.
func (app *Application) userFromGoogleCookie(r *http.Request) (string, error) {
	c, err := r.Cookie("google_user_id")
	if err != nil {
		return "", err
	}
	if v, ok := verifyValue(c.Value, app.SignKey); ok {
		return v, nil
	}
	return "", fmt.Errorf("invalid signature")
}

// requireGoogleUser enforces that the request has a valid google_user_id cookie.
// It writes a 401 response when missing or invalid.
func (app *Application) requireGoogleUser(w http.ResponseWriter, r *http.Request) (string, bool) {
	id, err := app.userFromGoogleCookie(r)
	if err != nil {
		http.Error(w, "authentication required", http.StatusUnauthorized)
		return "", false
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead && !verifyCSRF(r) {
		http.Error(w, "invalid csrf token", http.StatusForbidden)
		return "", false
	}
	return id, true
}

// decodeToken converts the base64 encoded JSON token stored in cookies back
// into an oauth2.Token instance.
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

// encodeToken signs and encodes the OAuth token for storage in a cookie.
func (app *Application) encodeToken(t *oauth2.Token, secure bool) *http.Cookie {
	b, _ := json.Marshal(t)
	return &http.Cookie{
		Name:     "spotify_token",
		Value:    signValue(base64.StdEncoding.EncodeToString(b), app.SignKey),
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	}
}

// refreshIfExpired refreshes the OAuth token if it has expired using the
// configured authenticator. The new token is persisted and written back to the
// cookie.
func (app *Application) refreshIfExpired(w http.ResponseWriter, r *http.Request, userID string, t *oauth2.Token) (*oauth2.Token, error) {
	if t == nil || t.Valid() || t.RefreshToken == "" {
		return t, nil
	}
	client := app.Authenticator.NewClient(t)
	newTok, err := client.Token()
	if err != nil {
		return t, err
	}
	if app.DB != nil && userID != "" {
		app.DB.SaveToken(r.Context(), userID, newTok)
	}
	http.SetCookie(w, app.encodeToken(newTok, r.TLS != nil))
	return newTok, nil
}

// Login begins the Spotify OAuth flow and redirects the user to the
// authorization URL with a signed state value stored in a cookie.
func (app *Application) Login(w http.ResponseWriter, r *http.Request) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		http.Error(w, "failed to generate state", http.StatusInternalServerError)
		return
	}
	state := base64.RawURLEncoding.EncodeToString(b)
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_state",
		Value:    signValue(state, app.SignKey),
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, app.Authenticator.AuthURL(state), http.StatusFound)
}

// OAuthCallback completes the OAuth flow by exchanging the authorization code
// for a token. The resulting token and user ID are stored in signed cookies.
func (app *Application) OAuthCallback(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("oauth_state")
	if err != nil {
		http.Error(w, "state mismatch", http.StatusBadRequest)
		return
	}
	state, ok := verifyValue(c.Value, app.SignKey)
	if !ok || r.URL.Query().Get("state") != state {
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
		app.DB.SaveToken(r.Context(), user.ID, token)
	}
	http.SetCookie(w, app.encodeToken(token, r.TLS != nil))
	if user != nil {
		http.SetCookie(w, &http.Cookie{
			Name:     "spotify_user_id",
			Value:    signValue(user.ID, app.SignKey),
			Path:     "/",
			HttpOnly: true,
			Secure:   r.TLS != nil,
			SameSite: http.SameSiteLaxMode,
		})
	}
	// Issue a CSRF token for the session so clients can include it with
	// state-changing requests.
	if _, err := setCSRFToken(w, r.TLS != nil); err != nil {
		http.Error(w, "csrf token", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/", http.StatusFound)
}

// Logout clears authentication cookies so the user must re-authenticate. It
// simply expires the relevant cookies on the client.
func (app *Application) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "spotify_user_id",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "spotify_token",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})
	http.SetCookie(w, &http.Cookie{Name: "csrf_token", Path: "/", MaxAge: -1})
	http.Redirect(w, r, "/", http.StatusFound)
}

// GoogleLogin starts the Google OAuth flow. The generated state value is signed
// and stored in a cookie mirroring the Spotify login behaviour.
func (app *Application) GoogleLogin(w http.ResponseWriter, r *http.Request) {
	if app.GoogleOAuth == nil {
		http.Error(w, "google auth not configured", http.StatusInternalServerError)
		return
	}
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		http.Error(w, "failed to generate state", http.StatusInternalServerError)
		return
	}
	state := base64.RawURLEncoding.EncodeToString(b)
	http.SetCookie(w, &http.Cookie{
		Name:     "google_state",
		Value:    signValue(state, app.SignKey),
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, app.GoogleOAuth.AuthCodeURL(state), http.StatusFound)
}

// GoogleCallback completes the Google OAuth flow and stores the user ID in a
// signed cookie so subsequent requests can be authenticated.
func (app *Application) GoogleCallback(w http.ResponseWriter, r *http.Request) {
	if app.GoogleOAuth == nil {
		http.Error(w, "google auth not configured", http.StatusInternalServerError)
		return
	}
	c, err := r.Cookie("google_state")
	if err != nil {
		http.Error(w, "state mismatch", http.StatusBadRequest)
		return
	}
	state, ok := verifyValue(c.Value, app.SignKey)
	if !ok || r.URL.Query().Get("state") != state {
		http.Error(w, "state mismatch", http.StatusBadRequest)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "google_state", Path: "/", MaxAge: -1})
	token, err := app.GoogleOAuth.Exchange(r.Context(), r.URL.Query().Get("code"))
	if err != nil {
		http.Error(w, "auth failed", http.StatusInternalServerError)
		return
	}
	client := app.GoogleOAuth.Client(r.Context(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil || resp.StatusCode != http.StatusOK {
		http.Error(w, "failed to fetch user", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()
	var data struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		http.Error(w, "decode user", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "google_user_id",
		Value:    signValue(data.ID, app.SignKey),
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})
	if _, err := setCSRFToken(w, r.TLS != nil); err != nil {
		http.Error(w, "csrf token", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/", http.StatusFound)
}
