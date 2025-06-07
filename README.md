# Smart-Music-Go

"Smart-Music-Go" is a music search tool that showcases the capabilities of Go in web development and provides a starting point for a more feature-rich music application.

The purpose of this project is to create a web application using the Go programming language that interacts with the Spotify Web API. The project serves as a practical example of how to build a web application with Go, demonstrating key aspects such as setting up a web server, handling HTTP requests and responses, structuring the project with packages, and interacting with a third-party API. The project also provides a foundation for further development. Additional features could be added, such as user authentication, saving favorite tracks, creating playlists, and more. The user interface could also be enhanced for a more interactive and user-friendly experience.

# Functionality
The application allows users to search for music tracks. When a user enters a track name, the application communicates with the Spotify API to fetch information about the track. The information retrieved includes the track name, the artist's name, and a link to listen to the track on Spotify.

- cmd/web/: This is where the application is initialized and the server is started. The main.go file will reside here.
- pkg/handlers/: This package will contain the HTTP handlers that respond to web requests.
- pkg/spotify/: This package will contain the code to interact with the Spotify API.
- ui/static/ and ui/templates/: These directories will contain the static files (CSS, JavaScript) and HTML templates for your application.
- go.mod and go.sum: These files are used by Go's module system.

# Getting Started
This section walks through setting up the application for local development.

## Prerequisites
- Go 1.23 or later
- go install github.com/zmb3/spotify@latest


### Environment Variables
The application requires Spotify credentials. Set the following variables before running:

```
SPOTIFY_CLIENT_ID=your-client-id
SPOTIFY_CLIENT_SECRET=your-client-secret
```

Set `DATABASE_PATH` to the SQLite file (defaults to `smartmusic.db`):

```
DATABASE_PATH=smartmusic.db
```
The database schema is created automatically on startup, so no manual migrations are required.

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

### Favorites
After logging in you can mark tracks as favorites from the search results. View them at `/favorites` or from the React "Favorites" tab.

### Frontend Setup

The React frontend lives under `ui/frontend` and is served from `/app/`.
Build the production assets before running the Go server:

```bash
cd ui/frontend
npm install
npm run build
```

After building, start the server as shown above and visit
`http://localhost:4000/app/` to use the React interface.

### Docker
A `Dockerfile` is included for local development. Build and run the container with:

```bash
docker build -t smart-music-go .
docker run --env-file .env -p 4000:4000 smart-music-go
```

### Deployment
Push the Docker image to your registry and run it on your preferred platform:

```bash
docker tag smart-music-go <registry>/smart-music-go
docker push <registry>/smart-music-go
```



# Future Work
- Frontend Development: The user interface is currently very basic. You might want to use a frontend framework like React, Vue, or Angular to create a more interactive and user-friendly UI. This could include things like a more advanced search form, a list of search results with album art and other details, and maybe even an audio player to preview tracks.
- User Authentication: If you want to add features like saving favorite tracks or creating playlists, you'll need to add user authentication. Spotify provides an OAuth 2.0-based authentication and authorization option that you can use.
- Deployment: Once your application is ready, you can deploy it to a server. You could use a cloud service like AWS, Google Cloud, or Heroku. You'll need to set up a domain name, SSL certificate, and possibly a database if you're storing user data.
- Testing: Write more tests to ensure your application works as expected. This could include unit tests for your individual functions, integration tests to make sure they work as expected when used together, and end-to-end tests to simulate user interactions.
- Continuous Integration/Continuous Deployment (CI/CD): Set up a CI/CD pipeline to automatically build, test, and deploy your application whenever you push changes to your code repository. This can help catch bugs early and streamline the deployment process.
- Documentation: Write documentation for the application, including how to use it, how to deploy it, and how to contribute to it.
