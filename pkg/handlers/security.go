// This file defines middleware used to attach common security headers to every
// HTTP response. Adding these headers helps mitigate common attacks such as
// clickjacking and MIME sniffing.
package handlers

import "net/http"

// SecurityHeaders wraps another http.Handler and sets several defensive HTTP
// headers before delegating to it. The headers enforce a simple Content Security
// Policy, disable MIME sniffing and prevent the page from being embedded in a
// frame. When served over HTTPS the function also enables Strict Transport
// Security to instruct browsers to prefer secure connections on future requests.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", "default-src 'self'")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "same-origin")
		if r.TLS != nil {
			w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		}
		next.ServeHTTP(w, r)
	})
}
