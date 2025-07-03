# Music Services Integration

Smart-Music-Go abstracts third-party music platforms behind a small interface so features work consistently regardless of provider. Each service implements the following methods:

```
type Service interface {
    SearchTrack(ctx context.Context, q string) ([]music.Track, error)
    GetRecommendations(ctx context.Context, seeds []string) ([]music.Track, error)
}
```

This allows the rest of the application to call `SearchTrack` or `GetRecommendations` without needing to know which service provides the results. The built-in implementations are:

- **Spotify** – Full API support for search, playlists and recommendations. Requires client credentials and handles OAuth for user specific features.
- **YouTube** – Searches the YouTube Music catalog via the Data API when `YOUTUBE_API_KEY` is set.
- **SoundCloud** – Uses the SoundCloud public API. A `SOUNDCLOUD_CLIENT_ID` is required.
- **Apple Music** – Queries the iTunes Search API; no authentication needed.
- **Tidal** – Performs basic track search using the Tidal API with a `TIDAL_TOKEN` extracted from the web player.
- **Amazon Music** – Placeholder client that currently returns a “not implemented” error.

When `MUSIC_SERVICE=aggregate` the `Aggregator` from `pkg/music` composes several services at once. Each search or recommendation request is executed concurrently across all enabled providers. Results are merged by track ID so duplicates are removed. If every provider fails, the first error is returned.

Configuration variables select the active provider. For example:

```
MUSIC_SERVICE=aggregate
YOUTUBE_API_KEY=your-youtube-key
SOUNDCLOUD_CLIENT_ID=client
TIDAL_TOKEN=tidal-token
```

Apple Music and Amazon Music do not require additional tokens. With aggregation enabled the user can search Spotify, YouTube, SoundCloud, Apple Music, Tidal and Amazon (once implemented) through a single interface.

