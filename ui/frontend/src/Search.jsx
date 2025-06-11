// Search lets the user query tracks via the server API and mark them as
// favorites.
import { useState } from 'react'

function Search({ theme }) {
  // Query string entered by the user.
  const [query, setQuery] = useState('')
  // Array of tracks returned from the API.
  const [results, setResults] = useState([])
  // Error message to display if the search fails.
  const [error, setError] = useState('')

  const handleSearch = async () => {
    // Skip searching if the input is empty.
    if (!query) return
    try {
      const res = await fetch(`/api/search?track=${encodeURIComponent(query)}`)
      if (!res.ok) {
        // Attempt to read an error message from the response.
        const data = await res.json().catch(() => ({}))
        setError(data.error || 'Search failed')
        setResults([])
        return
      }
      const data = await res.json()
      setResults(data)
      setError('')
    } catch {
      setError('Search failed')
      setResults([])
    }
  }

  const addFav = async (t) => {
    // Send the selected track to the server to be stored as a favorite.
    await fetch('/favorites', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        track_id: t.ID,
        track_name: t.Name,
        artist_name: t.Artists[0]?.Name,
      }),
    })
  }

  return (
    <div className={theme}>
      <h2>Search Tracks</h2>
      <input
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        placeholder="Enter a track name"
      />
      <button onClick={handleSearch}>Search</button>
      {error && <p>{error}</p>}
      <div className="results">
        {/* Render each returned track with basic info and a Favorite button */}
        {results.map((t) => (
          <div className="track" key={t.ID}>
            {t.Album?.Images?.length > 0 && (
              <img src={t.Album.Images[0].URL} alt="Album art" />
            )}
            <p className="name">{t.Name}</p>
            <p className="artist">{t.Artists[0]?.Name}</p>
            {t.PreviewURL && (
              <audio
                controls
                src={t.PreviewURL}
                className={`preview ${theme}`}
              />
            )}
            <button onClick={() => addFav(t)}>Favorite</button>
          </div>
        ))}
      </div>
    </div>
  )
}

export default Search
