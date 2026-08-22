import { createApp } from "vue";
import App from "./App.vue";
import "./style.css";

const mount = document.getElementById("wailsplugs-vue-root");
if (mount) createApp(App).mount(mount);

void import("./lazy").then(({ chunkLoaded }) => {
  if (chunkLoaded) console.debug("Vue plugin lazy chunk loaded");
});
