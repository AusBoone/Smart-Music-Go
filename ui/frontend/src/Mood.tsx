// Mood allows users to request recommendations based on tempo.
// It sends the track ID and optional tempo range to the server
// which uses Spotify audio features to build the playlist. Track
// cards reuse the same animated styling as the main search view.
import { useState } from "react";
import { api } from "./api";

// Track mirrors the subset of fields needed for display.
interface Track {
  ID: string;
  Name: string;
  Artists: { Name: string }[];
  Album?: { Images?: { URL: string }[] };
}

function Mood(): JSX.Element {
  const [trackID, setTrackID] = useState<string>("");
  const [results, setResults] = useState<Track[]>([]);
  const [error, setError] = useState<string>("");
  // Indicates a request is in flight.
  const [loading, setLoading] = useState<boolean>(false);

  const fetchMood = async () => {
    if (!trackID) return;
    try {
      setLoading(true);
      const data = await api<Track[]>(
        `/api/recommendations/mood?track_id=${encodeURIComponent(trackID)}`,
      );
      setResults(data);
      setError("");
      setLoading(false);
    } catch (e: any) {
      setError(e.message);
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
