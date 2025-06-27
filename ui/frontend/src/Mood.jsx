// Mood allows users to request recommendations based on tempo.
// It sends the track ID and optional tempo range to the server
// which uses Spotify audio features to build the playlist.
import { useState } from "react";

function Mood() {
  const [trackID, setTrackID] = useState("");
  const [results, setResults] = useState([]);
  const [error, setError] = useState("");

  const fetchMood = async () => {
    if (!trackID) return;
    try {
      const res = await fetch(
        `/api/recommendations/mood?track_id=${encodeURIComponent(trackID)}`,
      );
      if (!res.ok) {
        setError("Failed to load recommendations");
        setResults([]);
        return;
      }
      const data = await res.json();
      setResults(data);
      setError("");
    } catch {
      setError("Failed to load recommendations");
      setResults([]);
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
      {error && <p>{error}</p>}
      <div className="results">
        {results.map((t) => (
          <div className="track" key={t.ID}>
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
