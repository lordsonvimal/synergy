import { render } from "solid-js/web";
import { App } from "./App.js";

if ("serviceWorker" in navigator) {
  navigator.serviceWorker.register("/sw.js");
}

const root = document.getElementById("app");

if (root) {
  render(() => <App />, root);
}
