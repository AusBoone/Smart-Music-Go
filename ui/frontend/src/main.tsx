// Entry point that mounts the React application into the page.
import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import "./index.css";
import App from "./App";

const container = document.getElementById("root") as HTMLElement;

createRoot(container).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
