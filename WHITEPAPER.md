# Smart-Music-Go Whitepaper

## Abstract
Smart-Music-Go is an open source web application that demonstrates how a concise Go backend and React frontend can work together to explore music across many streaming platforms. The project began as a small example of OAuth with Spotify but now offers a generic search and recommendation interface, playlist browsing, and a favorites database. It aims to be an educational reference for building modular Go services while also providing a practical tool for discovering music.

## Background
Music discovery is often locked inside proprietary ecosystems. Users may build playlists on one platform but have friends using another. Smart-Music-Go shows how a single interface can search multiple providers, letting listeners share recommendations regardless of where a track originates. The project also highlights best practices for web security and containerized deployment.

## Goals and Motivation
- Simplify authentication with Spotify and other services.
- Provide a common interface for searching tracks and fetching recommendations from different providers.
- Store favorites and listening history in SQLite for personalized insights.
- Document the architecture so developers can extend it with new features like mood-based recommendations or collaborative playlists.
## Design Decisions
Smart-Music-Go favors explicit dependencies and small packages so that new developers can grasp the code quickly. SQLite was chosen for simplicity and because it requires no external service. The application avoids global state by passing an `Application` struct to handlers and uses interfaces to allow mock implementations in tests. Concurrency is used sparingly: the aggregator launches goroutines to query multiple services in parallel while the rest of the server runs single threaded handlers to keep state management simple.

When failures occur the aggregator preserves the first error returned so callers know why the overall request failed.  Search results are de-duplicated by track ID and ordered by insertion to keep responses predictable across providers.  This deterministic behaviour simplifies unit testing and avoids confusing changes in the UI.  Go modules are used to pin dependency versions which ensures builds are reproducible.


## Architecture Overview
The code is organized into clear packages so that functionality is easy to locate and replace:
- `cmd/web` – entry point configuring dependencies and starting the HTTP server.
- `pkg/handlers` – HTTP handlers working with an `Application` struct that contains the database layer and service clients.
- `pkg/spotify`, `pkg/youtube` and similar packages – wrappers around specific music platforms implementing the `music.Service` interface.
- `pkg/db` – SQLite layer responsible for persisting OAuth tokens, favorites, and play history.
- `ui/templates` – server rendered HTML templates.
- `ui/frontend` – React application compiled into static assets.

Requests first hit a handler which interacts with a service implementation. The handler may need to refresh OAuth tokens stored in SQLite, then issue calls to the provider (for example Spotify). Responses are either rendered via templates or returned as JSON for the React app. Cookies are signed using a secret to detect tampering. When `MUSIC_SERVICE=aggregate` the `Aggregator` dispatches queries concurrently to each configured provider and merges the results.

### Data Model
The SQLite database contains several tables:
- `favorites` – track ID, name, artist and associated user ID.
- `tokens` – OAuth tokens per service and user.
- `history` – optional table storing when tracks were played.
- `collections` – collaborative playlists with associated track and user tables.
No migrations are required because the schema is created automatically on startup.

## Music Service Interface
All providers conform to the following interface so new services can be added easily:
```go
type Service interface {
    SearchTrack(ctx context.Context, q string) ([]music.Track, error)
    GetRecommendations(ctx context.Context, seeds []string) ([]music.Track, error)
}
```
Implementations exist for Spotify, YouTube, SoundCloud, Apple Music, Tidal and an Amazon Music placeholder. When aggregation is enabled searches run in parallel across each service using a pool of goroutines controlled by a `MaxConcurrent` limit. Results are deduplicated by track ID and then sorted.

## Authentication and Security
Environment variables supply client credentials for Spotify and other services along with a cookie signing key. After the OAuth dance a session cookie stores the user ID and token. Cookies use the `HttpOnly` and `Secure` flags when HTTPS is active and are regenerated if tokens are refreshed. Each state changing endpoint requires a CSRF token found in the `csrf_token` cookie to protect against cross-site request forgery. The server sets strict headers such as `Content-Security-Policy`, `Referrer-Policy` and `X-Frame-Options` to mitigate common web attacks. Database queries use prepared statements to avoid injection vulnerabilities.

## API Summary
The HTTP API documented in `docs/api.md` and `docs/openapi/openapi.yaml` exposes endpoints for searching, recommendations, playlist retrieval, favorites management, share links and listening insights. All JSON responses use `application/json` and errors are formatted as `{ "error": "message" }`. The OpenAPI definition can be used to generate client libraries or interactive docs.
## User Flow
1. The user visits `/login` to authenticate with Spotify. After the callback sets a signed cookie the browser is redirected to the home page.
2. Searches are performed from `/search` or via the React interface under `/app/`. The handler delegates to the active `music.Service` implementation.
3. Tracks can be marked as favorites. They are saved to the `favorites` table and visible from `/favorites` or `/api/favorites`.
4. Signing in with Google via `/login/google` enables share links for tracks and playlists.
5. Optional endpoints record listening history which can later be summarized through the insights APIs.


## Deployment Model
The repository includes several deployment options:
- **Dockerfile** – builds a self-contained image that also compiles the React frontend.
- **docker-compose.yml** – suitable for local development and small servers, persisting the SQLite database to a host volume.
- **deploy/aws** – example Terraform code for running on AWS Fargate behind an Application Load Balancer. After pushing the Docker image, update `terraform.tfvars` with credentials and run `terraform apply` to create the stack.
- **Custom hosting** – any platform capable of running a container with environment variables will work. Configure TLS termination at the reverse proxy or load balancer and ensure `SPOTIFY_REDIRECT_URL` matches the public callback address.

## Configuration Reference
The server is configured entirely through environment variables. The most commonly used settings are:
- `SPOTIFY_CLIENT_ID` and `SPOTIFY_CLIENT_SECRET` – credentials for the Spotify API.
- `SPOTIFY_REDIRECT_URL` – OAuth callback URL (defaults to `http://localhost:4000/callback`).
- `SIGNING_KEY` – random string used to sign cookies and detect tampering.
- `DATABASE_PATH` – SQLite file path (defaults to `smartmusic.db`).
- `MUSIC_SERVICE` – selects the backend provider (`spotify`, `youtube`, `soundcloud`, `applemusic`, `tidal`, `amazon` or `aggregate`).
- `YOUTUBE_API_KEY`, `SOUNDCLOUD_CLIENT_ID` and `TIDAL_TOKEN` – optional tokens when using those providers.
- `GOOGLE_CLIENT_ID` and `GOOGLE_CLIENT_SECRET` – enable share links via Google OAuth.
Additional variables control the listening port (`PORT`), Google redirect URL and aggregator concurrency limits.

## Frontend Architecture
The React application under `ui/frontend` is a small single-page app built with Vite. Pages are organized into components for search, playlists, favorites and insights. State is managed with React context so that authentication and favorites are available throughout the component tree. Playwright tests exercise typical user interactions such as adding a favorite or generating a share link.

## Request Lifecycle Example
1. The browser sends a search request to `/api/search?track=name` including the CSRF token.
2. The handler verifies the token, refreshes OAuth credentials if needed and dispatches the query to the `music.Service` implementation.
3. Results are aggregated, deduplicated and returned as JSON.
4. The React frontend displays the list and allows the user to mark favorites which trigger `POST /api/favorites` calls.

## Testing Strategy
Backend packages include unit tests for handler logic and database operations. `cmd/web/integration_test.go` spins up an in-memory server to verify routing and error handling. The React frontend uses Playwright for end-to-end tests driven from the compiled build. Running `go test ./...` and `npx playwright test` ensures new changes do not break existing functionality.

## Vision and Roadmap
Future releases aim to evolve the project beyond a simple Spotify client:
- **Smart recommendations** that leverage Spotify audio analysis APIs for mood or energy based playlists.
- **Cross-platform search** via the aggregator so results from YouTube, SoundCloud and other services appear in one list.
- **Listening insights** over time using the history table to highlight top artists and tracks each month.
- **Collaborative playlists** stored in the collections tables so multiple users can add tracks.
- **UI enhancements** including theme switching, audio previews and improved mobile support.
## Extending the Project
Developers are encouraged to add new streaming providers by implementing the `music.Service` interface in a new package. Additional tables can be added to the SQLite schema if migrations are introduced. The React frontend may be expanded with more components or replaced entirely. Continuous integration and deployment scripts can be tailored to your infrastructure.
## Community and Contributions
The project is licensed under the MIT License and welcomes pull requests. Issues and discussions are tracked on GitHub. Contributors should follow the style guidelines in `CONTRIBUTING.md` and include unit tests for new features.



## Conclusion
Smart-Music-Go demonstrates how a small Go codebase can power a flexible music discovery tool. Its modular architecture and documented API make it a useful starting point for developers experimenting with multiple streaming services or advanced recommendation features. Contributions are welcome to expand integrations, improve the user experience and harden security even further.
