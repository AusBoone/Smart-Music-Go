# Architecture Overview

This document outlines the major components of **Smart-Music-Go** and how they interact.

## Packages

- `cmd/web`: entry point of the application. `main.go` wires the dependencies and starts the HTTP server.
- `pkg/handlers`: contains all HTTP handlers. Each handler receives an `Application` struct which holds the dependencies required to service the request such as the database and Spotify client.
- `pkg/spotify`: wraps the official Spotify client to provide simple helper methods used by handlers and tests.
- `pkg/db`: responsible for persisting OAuth tokens and user favourites in a SQLite database.

## Request flow

1. A request reaches the server and is routed to the appropriate handler in `pkg/handlers`.
2. Handlers call methods on the `SpotifyClient` to fetch data from the Spotify API. When authenticated requests are required the handler refreshes the OAuth token if necessary and stores the updated value in the database.
3. Templates located under `ui/templates` are rendered to build HTML pages. JSON APIs simply marshal and return the data structures used by the templates.

Cookies storing user IDs and tokens are signed to prevent tampering. Tokens are refreshed transparently when they expire and are updated both in the cookie and the database.
