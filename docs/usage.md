# Usage

This guide covers running Smart-Music-Go locally and configuring Spotify authentication.

## Prerequisites
- Go 1.22 or later
- `go install github.com/zmb3/spotify@latest`
- Node.js and npm for building the React frontend

## Configuration
1. Copy `.env.example` to `.env` and fill in your Spotify credentials:
   ```bash
   cp .env.example .env
   ```
2. Edit `.env` and set:
   ```
   SPOTIFY_CLIENT_ID=<your-client-id>
   SPOTIFY_CLIENT_SECRET=<your-client-secret>
   SPOTIFY_REDIRECT_URL=http://localhost:4000/callback
   ```
   Optionally set `DATABASE_PATH` to control where the SQLite database is stored (defaults to `smartmusic.db`).
3. In your Spotify developer dashboard add `http://localhost:4000/callback` as an allowed redirect URI.

## Building the Frontend
```
cd ui/frontend
npm install
npm run build
```
The built assets will be served from `/app/` when the Go server runs. When using
Docker or Docker Compose this build step is handled automatically during the
image build.

## Running the Server
Start the application with:
```bash
go run cmd/web/main.go
```
Visit `http://localhost:4000/login` to authenticate with Spotify then browse playlists or search for tracks. Favorites can be managed at `/favorites`.
JSON responses are served from `/api/search`, `/api/playlists` and `/api/favorites`. Favorites may be removed via `DELETE /api/favorites` and exported as CSV from `/api/favorites.csv`.
Sign in with Google at `/login/google` to enable generating share links with the "Share" button on search results.
Share links use random IDs for security. Playlists can also be shared from the playlists page using the generated link.
All write requests must include the CSRF token from the `csrf_token` cookie in the `X-CSRF-Token` header.
For monthly summaries of your listening history call `/api/insights/monthly`. Collaborative playlists can be created via `/api/collections` and managed using the `/api/collections/{id}/tracks` and `/api/collections/{id}/users` endpoints.
Mood based recommendations are available via `/api/recommendations/mood?track_id=<id>` when a Spotify client is configured. Advanced filtering of energy and valence can be accessed through `/api/recommendations/advanced` with query parameters like `min_energy`.
To search across all enabled services (Spotify, YouTube, SoundCloud, Apple Music, Tidal and Amazon) set `MUSIC_SERVICE=aggregate`. Provide `YOUTUBE_API_KEY`, `SOUNDCLOUD_CLIENT_ID` and `TIDAL_TOKEN` if you want those sources included.

## Docker
A `Dockerfile` and `docker-compose.yml` are provided for local development. The
image build automatically compiles the React frontend so no manual npm build is
needed.
To build and run directly with Docker:
```bash
docker build -t smart-music-go .
docker run --env-file .env -p 4000:4000 smart-music-go
```
To start with Docker Compose (persists the SQLite database to `./data`):
```bash
docker compose up --build
```

## Frontend Tests
The React application is tested using Playwright. From `ui/frontend` run:
```bash
npm install
npx playwright test
```
The tests assume the development server is running on port 4173 via `npm run preview` as configured in `playwright.config.cjs`.
