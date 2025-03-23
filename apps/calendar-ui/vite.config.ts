import { defineConfig } from "vite";
import solidPlugin from "vite-plugin-solid";

export default defineConfig({
  plugins: [solidPlugin()],
  server: {
    https: {
      cert: "../core/certs/server.crt",
      key: "../core/certs/server.key"
    },
    port: 3001
  },
  build: {
    target: "esnext"
  }
});
