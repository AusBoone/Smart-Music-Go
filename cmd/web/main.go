// This file will initialize our application and start the server.

package main

import (
	"net/http"
	"Smart-Music-Go/pkg/handlers"
)

func main() {
	// Initialize a new http.ServeMux, which is basically a HTTP request router (or multiplexer)
	mux := http.NewServeMux()

	// Initialize a new instance of application which contains pointers to our two handler methods
	app := &handlers.Application{}

	// Register the two URL patterns and their corresponding handler functions to the router
	mux.HandleFunc("/", app.Home)
	mux.HandleFunc("/search", app.Search)

	// Start the HTTP server
	http.ListenAndServe(":4000", mux)
}
