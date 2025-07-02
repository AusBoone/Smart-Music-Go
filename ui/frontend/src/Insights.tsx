// Insights displays listening analytics for the logged in user.
import { useEffect, useState } from "react";

interface ArtistCount {
  Artist: string;
  Count: number;
}

interface TrackCount {
  TrackID: string;
  Count: number;
}

function Insights(): JSX.Element {
  const [artists, setArtists] = useState<ArtistCount[]>([]);
  const [tracks, setTracks] = useState<TrackCount[]>([]);

  useEffect(() => {
    fetch("/api/insights")
      .then((r) => r.json())
      .then(setArtists)
      .catch(() => {});
    fetch("/api/insights/tracks")
      .then((r) => r.json())
      .then(setTracks)
      .catch(() => {});
  }, []);

  return (
    <div>
      <h2>Top Artists This Week</h2>
      <ul>
        {artists.map((a) => (
          <li key={a.Artist}>
            {a.Artist} ({a.Count})
          </li>
        ))}
      </ul>
      <h2>Top Tracks This Week</h2>
      <ul>
        {tracks.map((t) => (
          <li key={t.TrackID}>
            {t.TrackID} ({t.Count})
          </li>
        ))}
      </ul>
    </div>
  );
}

export default Insights;
