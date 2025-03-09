import { defineConfig } from "vite";
import mkcert from "vite-plugin-mkcert";
import solidPlugin from "vite-plugin-solid";

export default defineConfig({
  plugins: [mkcert(), solidPlugin()],
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
