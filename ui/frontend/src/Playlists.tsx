// Playlists fetches and displays the current user's Spotify playlists.
import { useEffect, useState } from "react";
import { api } from "./api";

// Playlist represents the minimal fields returned by the backend API.
interface Playlist {
  ID: string;
  Name: string;
}

function Playlists(): JSX.Element {
  // Array of playlists returned from the API.
  const [playlists, setPlaylists] = useState<Playlist[]>([]);
  // Holds an error message when loading fails.
  const [error, setError] = useState<string>("");
  // Loading indicator while playlists are fetched.
  const [loading, setLoading] = useState<boolean>(true);

  useEffect(() => {
    // Fetch the user's playlists on component mount.
    api<{ Playlists: Playlist[] }>("/api/playlists")
      .then((data) => setPlaylists(data.Playlists || []))
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false));
  }, []);

  return (
    <div>
      <h2>Your Playlists</h2>
      <p>
        <a href="/login">Login with Spotify</a>
      </p>
      {loading && <p>Loading...</p>}
      {error && <p>{error}</p>}
      <ul>
        {/* Render playlist names in a simple list */}
        {playlists.map((p) => (
          <li key={p.ID}>{p.Name}</li>
        ))}
      </ul>
    </div>
  );
}

export default Playlists;
