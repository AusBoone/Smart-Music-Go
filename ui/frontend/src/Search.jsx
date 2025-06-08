import { useState } from 'react'

function Search() {
  const [query, setQuery] = useState('')
  const [results, setResults] = useState([])
  const [error, setError] = useState('')

  const handleSearch = async () => {
    if (!query) return
    try {
      const res = await fetch(`/api/search?track=${encodeURIComponent(query)}`)
      if (!res.ok) {
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
    <div>
      <h2>Search Tracks</h2>
      <input
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        placeholder="Enter a track name"
      />
      <button onClick={handleSearch}>Search</button>
      {error && <p>{error}</p>}
      <div className="results">
        {results.map((t) => (
          <div className="track" key={t.ID}>
            {t.Album?.Images?.length > 0 && (
              <img src={t.Album.Images[0].URL} alt="Album art" />
            )}
            <p className="name">{t.Name}</p>
            <p className="artist">{t.Artists[0]?.Name}</p>
            <button onClick={() => addFav(t)}>Favorite</button>
          </div>
        ))}
      </div>
    </div>
  )
}

export default Search
