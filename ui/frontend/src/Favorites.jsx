// Favorites shows tracks that the user has previously marked as favorites.
import { useEffect, useState } from 'react'

function Favorites() {
  // List of favorite tracks stored for the user.
  const [favs, setFavs] = useState([])
  // Error message shown when the fetch fails.
  const [error, setError] = useState('')
  // Indicates favorites are being loaded.
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    // Retrieve favorites once when the component mounts.
    fetch('/api/favorites')
      .then((res) => {
        if (!res.ok) throw new Error('failed')
        return res.json()
      })
      .then((data) => setFavs(data))
      .catch(() => setError('Failed to load favorites'))
      .finally(() => setLoading(false))
  }, [])

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
  )
}

export default Favorites
