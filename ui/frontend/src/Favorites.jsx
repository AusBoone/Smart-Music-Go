// Favorites shows tracks that the user has previously marked as favorites.
import { useEffect, useState } from 'react'

function Favorites() {
  const [favs, setFavs] = useState([])
  const [error, setError] = useState('')

  useEffect(() => {
    fetch('/api/favorites')
      .then((res) => {
        if (!res.ok) throw new Error('failed')
        return res.json()
      })
      .then((data) => setFavs(data))
      .catch(() => setError('Failed to load favorites'))
  }, [])

  return (
    <div>
      <h2>Your Favorites</h2>
      {error && <p>{error}</p>}
      <ul>
        {favs.map((f) => (
          <li key={f.TrackID}>
            {f.TrackName} - {f.ArtistName}
          </li>
        ))}
      </ul>
    </div>
  )
}

export default Favorites
