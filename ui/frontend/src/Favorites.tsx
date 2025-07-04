// Favorites shows tracks that the user has previously marked as favorites.
import { useEffect, useState } from "react";
import { api } from "./api";

// Favorite corresponds to a favorite track entry returned by the API.
interface Favorite {
  TrackID: string;
  TrackName: string;
  ArtistName: string;
}

function Favorites(): JSX.Element {
  // List of favorite tracks stored for the user.
  const [favs, setFavs] = useState<Favorite[]>([]);
  // Error message shown when the fetch fails.
  const [error, setError] = useState<string>("");
  // Indicates favorites are being loaded.
  const [loading, setLoading] = useState<boolean>(true);

  useEffect(() => {
    // Retrieve favorites once when the component mounts.
    api<Favorite[]>("/api/favorites")
      .then(setFavs)
      .catch((e) => setError(e.message))
      .finally(() => setLoading(false));
  }, []);

  return (
    <div>
      <h2>Your Favorites</h2>
      {loading && <p>Loading...</p>}
      {error && <p>{error}</p>}
      <ul>
        {/* Display each favorite track */}
        {favs.map((f) => (
          <li key={f.TrackID}>
            {f.TrackName} - {f.ArtistName}
          </li>
        ))}
      </ul>
    </div>
  );
}

export default Favorites;
