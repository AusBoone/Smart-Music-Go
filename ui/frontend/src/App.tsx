// App is the root component for the React UI. It handles theme state and
// client-side routing using React Router so views can be directly linked.
import { useState, useEffect } from "react";
import { BrowserRouter as Router, Routes, Route, Link } from "react-router-dom";
import Search from "./Search";
import Playlists from "./Playlists";
import Favorites from "./Favorites";
import Recommendations from "./Recommendations";
import Mood from "./Mood";
import Insights from "./Insights";
import Collections from "./Collections";
import "./App.css";

function App() {
  // Light or dark theme preference stored in localStorage between sessions.
  const [theme, setTheme] = useState(
    () => localStorage.getItem("theme") || "light",
  );
  // Triggers a short fade animation whenever navigation occurs.
  const [animate, setAnimate] = useState(false);

  useEffect(() => {
    setAnimate(true);
    const timer = setTimeout(() => setAnimate(false), 400);
    return () => clearTimeout(timer);
  }, []);

  const toggleTheme = () =>
    setTheme((prev) => (prev === "light" ? "dark" : "light"));

  useEffect(() => {
    localStorage.setItem("theme", theme);
  }, [theme]);

  return (
    <Router>
      <div className={`App ${theme} ${animate ? "fade-in" : ""}`}>
        <h1>Smart Music Go</h1>
        <nav>
          <Link to="/search">Search</Link>
          <Link to="/recommendations">Recommendations</Link>
          <Link to="/mood">Mood</Link>
          <Link to="/playlists">Playlists</Link>
          <Link to="/favorites">Favorites</Link>
          <Link to="/insights">Insights</Link>
          <Link to="/collections">Collections</Link>
          <a href="/logout">Logout</a>
          <button onClick={toggleTheme}>Toggle Theme</button>
        </nav>
        <Routes>
          <Route path="/search" element={<Search theme={theme} />} />
          <Route path="/recommendations" element={<Recommendations />} />
          <Route path="/mood" element={<Mood />} />
          <Route path="/playlists" element={<Playlists />} />
          <Route path="/favorites" element={<Favorites />} />
          <Route path="/insights" element={<Insights />} />
          <Route path="/collections" element={<Collections />} />
          <Route path="*" element={<Search theme={theme} />} />
        </Routes>
      </div>
    </Router>
  );
}

export default App;
