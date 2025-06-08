// App is the root component for the React UI. It switches between the
// search, playlists and favorites views.
import { useState } from 'react'
import Search from './Search.jsx'
import Playlists from './Playlists.jsx'
import Favorites from './Favorites.jsx'
import './App.css'

function App() {
  const [view, setView] = useState('search')

  return (
    <div className="App">
      <h1>Smart Music Go</h1>
      <nav>
        <button onClick={() => setView('search')}>Search</button>
        <button onClick={() => setView('playlists')}>Playlists</button>
        <button onClick={() => setView('favorites')}>Favorites</button>
      </nav>
      {view === 'search' ? <Search /> : view === 'playlists' ? <Playlists /> : <Favorites />}
    </div>
  )
}

export default App
