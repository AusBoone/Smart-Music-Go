import { useEffect, useState } from 'react'

function Favorites() {
  const [html, setHtml] = useState('')

  useEffect(() => {
    fetch('/favorites')
      .then((res) => res.text())
      .then((text) => setHtml(text))
      .catch(() => setHtml('Failed to load favorites'))
  }, [])

  return (
    <div>
      <h2>Your Favorites</h2>
      <div dangerouslySetInnerHTML={{ __html: html }} />
    </div>
  )
}

export default Favorites
