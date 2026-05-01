import { defineConfig } from "vite";
import { resolve } from "path";
import solidPlugin from "vite-plugin-solid";
import tailwindcss from "@tailwindcss/vite";

export default defineConfig({
  root: "pwa",
  plugins: [solidPlugin(), tailwindcss()],
  server: {
    port: 5101,
    https: {
      cert: resolve(__dirname, "certs/cert.pem"),
      key: resolve(__dirname, "certs/key.pem")
    }
  },
  build: {
    outDir: resolve(__dirname, "dist/pwa"),
    emptyOutDir: true,
    target: "esnext"
  }
});
