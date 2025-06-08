import { useEffect, useState } from 'react'

function Playlists() {
  const [playlists, setPlaylists] = useState([])
  const [error, setError] = useState('')

  useEffect(() => {
    fetch('/api/playlists')
      .then((res) => {
        if (!res.ok) throw new Error('failed')
        return res.json()
      })
      .then((data) => setPlaylists(data.Playlists || []))
      .catch(() => setError('Failed to load playlists'))
  }, [])

  return (
    <div>
      <h2>Your Playlists</h2>
      <p>
        <a href="/login">Login with Spotify</a>
      </p>
      {error && <p>{error}</p>}
      <ul>
        {playlists.map((p) => (
          <li key={p.ID}>{p.Name}</li>
        ))}
      </ul>
    </div>
  )
}

export default Playlists
