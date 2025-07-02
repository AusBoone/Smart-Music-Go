// Collections lets the user create a shared playlist and view its tracks.
import { useState } from "react";

interface CollectionTrack {
  TrackID: string;
  TrackName: string;
  ArtistName: string;
}

function Collections(): JSX.Element {
  const [id, setId] = useState<string>("");
  const [tracks, setTracks] = useState<CollectionTrack[]>([]);
  // Loading state for track retrieval.
  const [loading, setLoading] = useState<boolean>(false);

  const create = async () => {
    const res = await fetch("/api/collections", { method: "POST" });
    const data = await res.json();
    setId(data.id);
  };

  const load = async () => {
    setLoading(true);
    const res = await fetch(`/api/collections/${id}/tracks`);
    setTracks(await res.json());
    setLoading(false);
  };

  return (
    <div>
      <button onClick={create}>New Collection</button>
      {id && (
        <div>
          <p>Collection ID: {id}</p>
          <button onClick={load}>Load Tracks</button>
        </div>
      )}
      {loading && <p>Loading...</p>}
      <ul>
        {tracks.map((t) => (
          <li key={t.TrackID}>
            {t.TrackName} - {t.ArtistName}
          </li>
        ))}
      </ul>
    </div>
  );
}

export default Collections;
