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
JSON responses are served from `/api/search`, `/api/playlists` and `/api/favorites` for use by the React frontend.

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
