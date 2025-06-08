# Contributing

Thank you for considering contributing to Smart-Music-Go! The following guidelines help keep the project consistent and easy to work with.

## Development Setup
1. Install Go 1.22 or later and clone the repository.
2. Copy `.env.example` to `.env` and provide your Spotify credentials as described in [docs/usage.md](docs/usage.md).
3. Build the frontend assets under `ui/frontend` with `npm install && npm run build`.

## Coding Standards
- Run `gofmt -w` on all Go files before committing.
- Ensure tests pass with:
  ```bash
  go test ./...
  ```
- For frontend changes run `npm run build` so that `ui/frontend/dist` is updated.

## Pull Requests
1. Create a feature branch for your change.
2. Commit your work with clear messages.
3. Open a pull request describing the motivation and any relevant issue numbers.

We welcome bug fixes, new features and documentation improvements!
