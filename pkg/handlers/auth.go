// Package handlers contains HTTP handlers for Smart-Music-Go. This file groups
// authentication related helpers and endpoints such as the OAuth login and
// callback handlers. Keeping these routines separate from the main handlers
// file improves readability and maintainability.

package handlers

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
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
			SameSite: http.SameSiteLaxMode,
		})
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
		SameSite: http.SameSiteLaxMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "spotify_token",
		Path:     "/",
		MaxAge:   -1,
		SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, "/", http.StatusFound)
}
