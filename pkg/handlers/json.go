// Package handlers contains HTTP handler implementations for Smart-Music-Go.
// This file adds a small helper for decoding JSON requests with validation.
//
// decodeJSON reads the request body into v, enforcing a reasonable limit and
// rejecting unknown fields. It returns an error suitable for use with
// respondJSONError. Callers should check for errors and send an appropriate
// response.
package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

// decodeJSON attempts to decode the request body into the provided destination.
// The body is limited to 1MB to guard against malicious requests. Unknown
// fields cause an error so clients cannot send unexpected data.
func decodeJSON(r *http.Request, v any) error {
	if r.Body == nil {
		return errors.New("empty body")
	}
	defer r.Body.Close()
	r.Body = http.MaxBytesReader(nil, r.Body, 1<<20) // 1MB
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		if err == io.EOF {
			return errors.New("empty body")
		}
		return err
	}
	if dec.More() {
		return errors.New("extra data in request body")
	}
	return nil
}
