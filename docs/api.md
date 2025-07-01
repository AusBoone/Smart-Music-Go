# API Reference

This document details the HTTP endpoints exposed by **Smart-Music-Go**. All JSON responses use `application/json` and require Spotify authentication unless otherwise noted. Endpoints under `/api` are primarily consumed by the React frontend but can be called directly.

## Authentication

Authenticate with Spotify by visiting `/login`. After authorizing, the callback at `/callback` stores a signed cookie containing the user ID and token. Most endpoints that access user data expect this cookie.

## Endpoints

Each endpoint description notes the HTTP method, required parameters, and
authentication needs. Unless otherwise specified responses are JSON.
Errors are returned in the form `{ "error": "message" }` with an appropriate
HTTP status code.

### `GET /`
Render the landing page with a search form.

### `GET /search?track=NAME`
Render search results for the provided track name.

### `GET /api/search?track=NAME`
Search for tracks across configured services. Parameters:
* `track` - required string query. URL-encoded.

Returns an array of track objects on success. Example response:
```json
[
  { "ID": "123", "Name": "Song", "Artists": [{ "Name": "Artist" }] }
]
```
If no tracks match, the server responds with `404` and a JSON error message.

### `GET /recommendations?track_id=ID`
Render HTML recommendations based on a seed track.

### `GET /api/recommendations?track_id=ID`
Return recommendations as JSON. Requires a valid `track_id`.

### `GET /api/recommendations/mood?track_id=ID[&min_tempo=x&max_tempo=y&min_energy=a&max_energy=b&min_dance=c&max_dance=d]`
Generate recommendations using audio features around the seed track. A Spotify client must be configured.

### `GET /api/recommendations/advanced?track_id=ID[&min_energy=x&min_valence=y]`
Advanced filtering of recommendations using explicit audio feature bounds.

### `GET /playlists`
Display the user's Spotify playlists. Requires authentication.

### `GET /api/playlists`
Return the user's playlists as provided by Spotify. Authentication required.
Response matches Spotify's `SimplePlaylistPage` JSON schema.

### `POST /favorites`
Add a track to the authenticated user's favorites list. Body fields:
* `track_id` - Spotify track ID
* `track_name` - display name
* `artist_name` - main artist name

Example:
```json
{ "track_id": "id", "track_name": "name", "artist_name": "artist" }
```
Returns `201 Created` on success. Missing fields result in `400`.

### `GET /favorites`
Render the user's favorite tracks.

### `GET /api/favorites`
Return favorites as JSON. Authentication required.

### `POST /api/history`
Record that a track was played for the current user. Body fields:
* `track_id` - Spotify track ID
* `artist_name` - artist string

Example:
```json
{ "track_id": "id", "artist_name": "artist" }
```
Returns `201 Created` when saved or `400` on validation errors.

### `POST /api/collections`
Create a collaborative playlist owned by the current user. Returns the new
collection ID in the form:
```json
{ "id": "collection-id" }
```

### `POST /api/collections/{id}/tracks`
Add a track to the specified collection. Body uses the same fields as the
`/favorites` endpoint. Responds with `201 Created`.

### `POST /api/collections/{id}/users`
Add a user to an existing collection. Body:
```json
{ "user_id": "spotify-user" }
```
Returns `201 Created` when the user is associated.

### `GET /api/collections/{id}/tracks`
List tracks stored in the collection. Response example:
```json
[
  { "TrackID": "1", "TrackName": "Song", "ArtistName": "Artist" }
]
```

### `GET /api/insights`
Return top artists played in the last week. Response is an array of
`{ "Artist": "name", "Count": 10 }` objects.

### `GET /api/insights/tracks?days=N`
Return top tracks for the last `N` days (defaults to 7). Each element contains
the track ID and play count.

### `GET /api/insights/monthly?since=YYYY-MM-DD`
Return monthly play counts since the given date (defaults to one year ago). The
result is an array of `{ "Month": "YYYY-MM", "Count": N }` entries sorted
chronologically.

