# Smart-Music-Go

Smart-Music-Go is a lightweight web app written in Go. It illustrates how to authenticate with Spotify, search the catalog, browse your playlists and store favorite tracks in a SQLite database. The server exposes JSON APIs consumed by a small React frontend while also rendering basic HTML templates.

Originally created as an educational example of Go web development, the project now provides a simple but functional Spotify client that can be extended into a richer music discovery tool.

# Functionality
The application lets you log in with Spotify, search the catalog and browse your existing playlists. Search results display the track name, artist and a link to Spotify, and you can mark any item as a favorite for later viewing.

- cmd/web/: This is where the application is initialized and the server is started. The main.go file will reside here.
- pkg/handlers/: This package will contain the HTTP handlers that respond to web requests.
- pkg/spotify/: This package will contain the code to interact with the Spotify API.
- ui/static/ and ui/templates/: These directories will contain the static files (CSS, JavaScript) and HTML templates for your application.
- go.mod and go.sum: These files are used by Go's module system.

# Getting Started
This section walks through setting up the application for local development.

## Prerequisites
- Go 1.22 or later
- go install github.com/zmb3/spotify@latest


### Environment Variables
The application requires Spotify credentials. Set the following variables before running:

```
SPOTIFY_CLIENT_ID=your-client-id
SPOTIFY_CLIENT_SECRET=your-client-secret
SIGNING_KEY=some-random-string
```

`SIGNING_KEY` is used to sign cookies so tampering attempts are detected.

Set `DATABASE_PATH` to the SQLite file (defaults to `smartmusic.db`):

```
DATABASE_PATH=smartmusic.db
```
The database schema is created automatically on startup, so no manual migrations are required.

`MUSIC_SERVICE` selects the backend provider. Options are `spotify`, `youtube`,
`soundcloud` or `aggregate`. When using the YouTube or SoundCloud providers you
must set `YOUTUBE_API_KEY` or `SOUNDCLOUD_CLIENT_ID` respectively. The
`aggregate` mode queries all available services.

You can copy the provided `.env.example` to `.env` and populate your values:

```
cp .env.example .env
```

### Redirect URI
Add `http://localhost:4000/callback` as an allowed redirect URI in your Spotify
developer dashboard and ensure `SPOTIFY_REDIRECT_URL` matches this value.

### Running the Server

```bash
go run cmd/web/main.go
```

### Viewing Results
Visit `http://localhost:4000/login` to authorize with Spotify. After authorization, open `http://localhost:4000/playlists` or perform a search.
The API endpoints `/api/search`, `/api/playlists` and `/api/favorites` return JSON used by the React interface.
Additional endpoints provide listening insights and collaborative playlist features:

```
curl http://localhost:4000/api/insights/monthly
curl -X POST http://localhost:4000/api/collections
curl http://localhost:4000/api/recommendations/mood?track_id=123
curl "http://localhost:4000/api/recommendations/advanced?track_id=123&min_energy=0.5"
```


### Favorites
After logging in you can mark tracks as favorites from the search results. View them at `/favorites` or from the React "Favorites" tab.

### Frontend Setup

The React frontend lives under `ui/frontend` and is served from `/app/`.
Build the production assets before running the Go server (the Docker image
builds these automatically):

```bash
cd ui/frontend
npm install
npm run build
```

After building, start the server as shown above and visit
`http://localhost:4000/app/` to use the React interface.

### Docker
A `Dockerfile` is included for local development. Building the image also compiles
the React frontend, so no manual npm build is required. Build and run the container with:

```bash
docker build -t smart-music-go .
docker run --env-file .env -p 4000:4000 smart-music-go
```

You can also run the project via Docker Compose which persists the SQLite
database to a host volume:

```bash
docker compose up --build
```

### Deployment
Push the Docker image to your registry and run it on your preferred platform:

```bash
docker tag smart-music-go <registry>/smart-music-go
docker push <registry>/smart-music-go
```

Example Terraform configuration for deploying to AWS Fargate is provided under
`deploy/aws`. It creates an ECS service behind an Application Load Balancer with
HTTPS termination. After pushing your image, update the variables in
`terraform.tfvars` and run:

```bash
cd deploy/aws
terraform init
terraform apply
```

The load balancer DNS output contains the HTTPS endpoint for the application.

### Production Configuration
Set environment variables using your platform's secret storage (for example
AWS Secrets Manager or Heroku config vars). When running via Docker Compose or
Terraform, the variables from `.env` can be supplied via `env_file` or the
Terraform variables.


For SSL, terminate TLS at your load balancer or reverse proxy using a certificate
from Let's Encrypt or your cloud provider's certificate manager.

## Documentation
Detailed guides for running the application, deploying to production and
contributing can be found in the [docs](docs) directory:

- [Usage](docs/usage.md) – running locally and configuring authentication
 - [Deployment](docs/deployment.md) – production deployment options
 - [Architecture](docs/architecture.md) – overview of packages and request flow
 - [Contributing](CONTRIBUTING.md)
 - [OpenAPI Spec](docs/openapi/openapi.yaml) – machine readable API definition

The OpenAPI file can be used with tools like `swagger-codegen` to generate client
libraries or interactive documentation:

```bash
swagger-codegen generate -i docs/openapi/openapi.yaml -l html2 -o api-docs
```



## Vision
Smart-Music-Go aims to evolve beyond a basic Spotify interface. Planned improvements include:

- **Smart recommendations** that leverage Spotify's audio analysis API to suggest tracks or build playlists by mood, tempo, energy and danceability.
- **Multi-service search** across platforms like YouTube or SoundCloud for broader discovery. A new `aggregate` mode queries all providers at once.
- **Listening insights** now provide top artists and top tracks over a chosen period.
- **Monthly summaries** group your listening history by month for trend analysis.
- **Collaborative playlists** APIs allow creating collections that multiple users can populate.
- **Personal listening insights** stored in SQLite to highlight trends and weekly discoveries.
- **Enhanced UI/UX** with theme switching and audio previews in the React frontend.

