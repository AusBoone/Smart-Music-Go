import { useState } from 'react'

function Search() {
  const [query, setQuery] = useState('')
  const [resultsHtml, setResultsHtml] = useState('')

  const handleSearch = async () => {
    if (!query) return
    const res = await fetch(`/search?track=${encodeURIComponent(query)}`)
    const text = await res.text()
    setResultsHtml(text)
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
      <div dangerouslySetInnerHTML={{ __html: resultsHtml }} />
    </div>
  )
}

export default Search
