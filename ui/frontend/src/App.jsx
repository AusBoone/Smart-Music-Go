// App is the root component for the React UI. It switches between the
// various views (search, playlists, favorites, etc.) and controls the
// light/dark theme. This file also applies a small fade animation when
// navigating between sections to make the interface feel more responsive.
import { useState, useEffect } from "react";
import Search from "./Search.jsx";
import Playlists from "./Playlists.jsx";
import Favorites from "./Favorites.jsx";
import Recommendations from "./Recommendations.jsx";
import Mood from "./Mood.jsx";
import Insights from "./Insights.jsx";
import Collections from "./Collections.jsx";
import "./App.css";

function App() {
  // Track which section of the app is currently visible.
  const [view, setView] = useState("search");
  // Light or dark theme for styling. The initial value is read from
  // localStorage so returning users keep their preference.
  const [theme, setTheme] = useState(
    () => localStorage.getItem("theme") || "light",
  );
  // When true the container receives the fade-in class for a short
  // animation whenever the view changes.
  const [animate, setAnimate] = useState(false);

  // Trigger animation on view changes and clear the flag after the
  // duration finishes so the class can be re-applied later.
  useEffect(() => {
    setAnimate(true);
    const timer = setTimeout(() => setAnimate(false), 400);
    return () => clearTimeout(timer);
  }, [view]);

  const toggleTheme = () =>
    setTheme((prev) => (prev === "light" ? "dark" : "light"));

  // Persist the current theme in localStorage whenever it changes.
  useEffect(() => {
    localStorage.setItem("theme", theme);
  }, [theme]);

  return (
    <div className={`App ${theme} ${animate ? "fade-in" : ""}`}>
      <h1>Smart Music Go</h1>
      <nav>
        <button onClick={() => setView("search")}>Search</button>
        <button onClick={() => setView("recommendations")}>
          Recommendations
        </button>
        <button onClick={() => setView("mood")}>Mood</button>
        <button onClick={() => setView("playlists")}>Playlists</button>
        <button onClick={() => setView("favorites")}>Favorites</button>
        <button onClick={() => setView("insights")}>Insights</button>
        <button onClick={() => setView("collections")}>Collections</button>
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
      ) : view === "favorites" ? (
        <Favorites />
      ) : view === "insights" ? (
        <Insights />
      ) : (
        <Collections />
      )}
    </div>
  );
}

export default App;
