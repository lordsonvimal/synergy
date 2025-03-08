import { defineConfig } from "vite";
import basicSsl from "@vitejs/plugin-basic-ssl";
import solidPlugin from "vite-plugin-solid";

export default defineConfig({
  plugins: [solidPlugin()],
  server: {
    port: 3001
  },
  build: {
    target: "esnext"
  }
});
