// Recommendations retrieves track suggestions based on a seed track ID.
// Results are presented with the animated card component for a modern look.
import { useState } from "react";

function Recommendations() {
  const [trackID, setTrackID] = useState("");
  const [results, setResults] = useState([]);
  const [error, setError] = useState("");
  // Tracks whether the recommendation request is in progress.
  const [loading, setLoading] = useState(false);

  const fetchRecs = async () => {
    if (!trackID) return;
    try {
      setLoading(true);
      const res = await fetch(
        `/api/recommendations?track_id=${encodeURIComponent(trackID)}`,
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
      <h2>Recommendations</h2>
      <input
        value={trackID}
        onChange={(e) => setTrackID(e.target.value)}
        placeholder="Enter a track ID"
      />
      <button onClick={fetchRecs}>Get Recommendations</button>
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

export default Recommendations;
