// Search lets the user query tracks via the server API and mark them as
// favorites. Returned tracks are displayed using the "card" style with a
// subtle fade-in animation defined in the global CSS.
import { useState } from "react";

interface Track {
  ID: string;
  Name: string;
  Artists: { Name: string }[];
  Album?: { Images?: { URL: string }[] };
  PreviewURL?: string;
}

interface Props {
  theme: string;
}

function Search({ theme }: Props): JSX.Element {
  // Query string entered by the user.
  const [query, setQuery] = useState<string>("");
  // Array of tracks returned from the API.
  const [results, setResults] = useState<Track[]>([]);
  // Error message to display if the search fails.
  const [error, setError] = useState<string>("");
  // Indicates the search request is in progress.
  const [loading, setLoading] = useState<boolean>(false);

  const handleSearch = async () => {
    // Skip searching if the input is empty.
    if (!query) return;
    try {
      setLoading(true);
      const res = await fetch(`/api/search?track=${encodeURIComponent(query)}`);
      if (!res.ok) {
        // Attempt to read an error message from the response.
        const data = await res.json().catch(() => ({}));
        setError(data.error || "Search failed");
        setResults([]);
        setLoading(false);
        return;
      }
      const data = await res.json();
      setResults(data);
      setError("");
      setLoading(false);
    } catch {
      setError("Search failed");
      setResults([]);
      setLoading(false);
    }
  };

  const addFav = async (t: Track) => {
    // Send the selected track to the server to be stored as a favorite.
    await fetch("/favorites", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        track_id: t.ID,
        track_name: t.Name,
        artist_name: t.Artists[0]?.Name,
      }),
    });
  };

  return (
    <div className={theme}>
      <h2>Search Tracks</h2>
      <input
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        placeholder="Enter a track name"
      />
      <button onClick={handleSearch}>Search</button>
      {loading && <p>Loading...</p>}
      {error && <p>{error}</p>}
      <div className="results">
        {/* Render each returned track with basic info and a Favorite button */}
        {results.map((t) => (
          <div className="track card fade-in" key={t.ID}>
            {t.Album?.Images?.length > 0 && (
              <img src={t.Album.Images[0].URL} alt="Album art" />
            )}
            <p className="name">{t.Name}</p>
            <p className="artist">{t.Artists[0]?.Name}</p>
            {t.PreviewURL && (
              <audio
                controls
                src={t.PreviewURL}
                className={`preview ${theme}`}
              />
            )}
            <button onClick={() => addFav(t)}>Favorite</button>
          </div>
        ))}
      </div>
    </div>
  );
}

export default Search;
