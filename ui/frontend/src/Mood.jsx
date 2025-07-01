// Mood allows users to request recommendations based on tempo.
// It sends the track ID and optional tempo range to the server
// which uses Spotify audio features to build the playlist. Track
// cards reuse the same animated styling as the main search view.
import { useState } from "react";

function Mood() {
  const [trackID, setTrackID] = useState("");
  const [results, setResults] = useState([]);
  const [error, setError] = useState("");
  // Indicates a request is in flight.
  const [loading, setLoading] = useState(false);

  const fetchMood = async () => {
    if (!trackID) return;
    try {
      setLoading(true);
      const res = await fetch(
        `/api/recommendations/mood?track_id=${encodeURIComponent(trackID)}`,
      );
      if (!res.ok) {
        setError("Failed to load recommendations");
        setResults([]);
        setLoading(false);
        return;
      }
      const data = await res.json();
      setResults(data);
      setError("");
      setLoading(false);
    } catch {
      setError("Failed to load recommendations");
      setResults([]);
      setLoading(false);
    }
  };

  return (
    <div>
      <h2>Mood Playlist</h2>
      <input
        value={trackID}
        onChange={(e) => setTrackID(e.target.value)}
        placeholder="Track ID"
      />
      <button onClick={fetchMood}>Generate</button>
      {loading && <p>Loading...</p>}
      {error && <p>{error}</p>}
      <div className="results">
        {results.map((t) => (
          <div className="track card fade-in" key={t.ID}>
            {t.Album?.Images?.length > 0 && (
              <img src={t.Album.Images[0].URL} alt="Album art" />
            )}
            <p className="name">{t.Name}</p>
            <p className="artist">{t.Artists[0]?.Name}</p>
          </div>
        ))}
      </div>
    </div>
  );
}

export default Mood;
