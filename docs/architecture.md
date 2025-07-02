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

## Aggregator Service

When `MUSIC_SERVICE` is set to `aggregate` an `Aggregator` composes multiple
`music.Service` implementations. Searches and recommendations are dispatched to
each provider concurrently (up to `MaxConcurrent` goroutines). Results are
merged with duplicates removed by track ID. If every provider fails the first
error is returned to the caller.

Example configuration using YouTube and SoundCloud:

```
MUSIC_SERVICE=aggregate
YOUTUBE_API_KEY=your-key
SOUNDCLOUD_CLIENT_ID=client
```

Requests now include tracks gathered from all providers.
