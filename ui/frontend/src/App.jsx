// App is the root component for the React UI. It switches between the
// search, playlists and favorites views.
import { useState } from "react";
import Search from "./Search.jsx";
import Playlists from "./Playlists.jsx";
import Favorites from "./Favorites.jsx";
import Recommendations from "./Recommendations.jsx";
import Mood from "./Mood.jsx";
import "./App.css";

function App() {
  // Track which section of the app is currently visible.
  const [view, setView] = useState("search");
  // Light or dark theme for styling.
  const [theme, setTheme] = useState("light");

  const toggleTheme = () =>
    setTheme((prev) => (prev === "light" ? "dark" : "light"));

  return (
    <div className={`App ${theme}`}>
      <h1>Smart Music Go</h1>
      <nav>
        <button onClick={() => setView("search")}>Search</button>
        <button onClick={() => setView("recommendations")}>
          Recommendations
        </button>
        <button onClick={() => setView("mood")}>Mood</button>
        <button onClick={() => setView("playlists")}>Playlists</button>
        <button onClick={() => setView("favorites")}>Favorites</button>
        <button onClick={toggleTheme}>Toggle Theme</button>
      </nav>
      {/* Conditionally render the selected view component */}
      {view === "search" ? (
        <Search theme={theme} />
      ) : view === "recommendations" ? (
        <Recommendations />
      ) : view === "mood" ? (
        <Mood />
      ) : view === "playlists" ? (
        <Playlists />
      ) : (
        <Favorites />
      )}
    </div>
  );
}

export default App;
