import React from "react";
import { createRoot } from "react-dom/client";
import "./style.css";

function App() {
  return (
    <div className="wailsplugs-react-card">
      <strong>React plugin</strong>
      <span>This UI inherits the host document CSS cascade.</span>
    </div>
  );
}

const mount = document.getElementById("wailsplugs-react-root");
if (mount) createRoot(mount).render(<App />);

void import("./lazy").then(({ chunkLoaded }) => {
  if (chunkLoaded) console.debug("React plugin lazy chunk loaded");
});
