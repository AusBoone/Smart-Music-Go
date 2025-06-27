// Collections lets the user create a shared playlist and view its tracks.
import { useState } from "react";

function Collections() {
  const [id, setId] = useState("");
  const [tracks, setTracks] = useState([]);

  const create = async () => {
    const res = await fetch("/api/collections", { method: "POST" });
    const data = await res.json();
    setId(data.id);
  };

  const load = async () => {
    const res = await fetch(`/api/collections/${id}/tracks`);
    setTracks(await res.json());
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
