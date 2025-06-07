import { useState } from 'react'
import Search from './Search.jsx'
import Playlists from './Playlists.jsx'
import './App.css'

function App() {
  const [view, setView] = useState('search')

  return (
    <div className="App">
      <h1>Smart Music Go</h1>
      <nav>
        <button onClick={() => setView('search')}>Search</button>
        <button onClick={() => setView('playlists')}>Playlists</button>
      </nav>
      {view === 'search' ? <Search /> : <Playlists />}
    </div>
  )
}

export default App
