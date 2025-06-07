import { useEffect, useState } from 'react'

function Playlists() {
  const [html, setHtml] = useState('')

  useEffect(() => {
    fetch('/playlists')
      .then((res) => res.text())
      .then((text) => setHtml(text))
      .catch(() => setHtml('Failed to load playlists'))
  }, [])

  return (
    <div>
      <h2>Your Playlists</h2>
      <p>
        <a href="/login">Login with Spotify</a>
      </p>
      <div dangerouslySetInnerHTML={{ __html: html }} />
    </div>
  )
}

export default Playlists
