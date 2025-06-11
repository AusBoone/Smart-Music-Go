// App is the root component for the React UI. It switches between the
// search, playlists and favorites views.
import { useState } from "react";
import Search from "./Search.jsx";
import Playlists from "./Playlists.jsx";
import Favorites from "./Favorites.jsx";
import Recommendations from "./Recommendations.jsx";
import "./App.css";

function App() {
  // Track which section of the app is currently visible.
  const [view, setView] = useState("search");

  return (
    <div className="App">
      <h1>Smart Music Go</h1>
      <nav>
        <button onClick={() => setView("search")}>Search</button>
        <button onClick={() => setView("recommendations")}>
          Recommendations
        </button>
        <button onClick={() => setView("playlists")}>Playlists</button>
        <button onClick={() => setView("favorites")}>Favorites</button>
      </nav>
      {/* Conditionally render the selected view component */}
      {view === "search" ? (
        <Search />
      ) : view === "recommendations" ? (
        <Recommendations />
      ) : view === "playlists" ? (
        <Playlists />
      ) : (
        <Favorites />
      )}
    </div>
  );
}

export default App;
