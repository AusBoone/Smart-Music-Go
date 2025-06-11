// Recommendations retrieves track suggestions based on a seed track ID.
import { useState } from "react";

function Recommendations() {
  const [trackID, setTrackID] = useState("");
  const [results, setResults] = useState([]);
  const [error, setError] = useState("");

  const fetchRecs = async () => {
    if (!trackID) return;
    try {
      const res = await fetch(
        `/api/recommendations?track_id=${encodeURIComponent(trackID)}`,
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
      <h2>Recommendations</h2>
      <input
        value={trackID}
        onChange={(e) => setTrackID(e.target.value)}
        placeholder="Enter a track ID"
      />
      <button onClick={fetchRecs}>Get Recommendations</button>
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

export default Recommendations;
