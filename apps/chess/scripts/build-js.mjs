// Bundle each JS entrypoint with esbuild and emit content-hashed filenames.
// Writes nothing else — the asset script picks up the produced files and
// merges them into dist/manifest.json. Dev mode (CHESS_BUILD_DEV=1) skips
// hashing so air's filesystem watcher doesn't loop on every rebuild.
import { build } from "esbuild";
import { mkdir, readdir, rm, writeFile } from "node:fs/promises";
import { existsSync } from "node:fs";
import { join } from "node:path";

const DEV = process.env.CHESS_BUILD_DEV === "1";
const OUTDIR = "dist/js";

// Entrypoints: logical-name -> source path.
//   datastar.js is the page-wide reactive runtime. It is built standalone and
//   exposed to the browser via an import map (see layout.templ). Application
//   bundles import it via the bare specifier "datastar", which esbuild marks
//   external so each bundle stays small and a single runtime instance is
//   shared at runtime.
const ENTRIES = {
  "datastar":  "assets/datastar.js",
  "initClock": "assets/js/initClock.js",
  "board":     "assets/js/board.js",
};

await rm(OUTDIR, { recursive: true, force: true });
await mkdir(OUTDIR, { recursive: true });

const result = await build({
  entryPoints: Object.entries(ENTRIES)
    .filter(([, src]) => existsSync(src))
    .map(([name, src]) => ({ in: src, out: name })),
  bundle: true,
  format: "esm",
  splitting: false,
  // datastar is loaded once via an import map; every other bundle imports it
  // via the bare specifier "datastar" so the browser resolves it from the map.
  external: ["datastar"],
  minify: !DEV,
  sourcemap: DEV ? "inline" : false,
  target: ["es2022"],
  outdir: OUTDIR,
  entryNames: DEV ? "[name]" : "[name].[hash]",
  metafile: true,
  logLevel: "info",
});

// Build js portion of the manifest: "js/<logical>.js" -> "js/<actual>".
const jsManifest = {};
for (const [outPath] of Object.entries(result.metafile.outputs)) {
  // outPath looks like "dist/js/board.A1B2C3D4.js"
  const file = outPath.replace(/^dist\//, "");
  const base = file.replace(/^js\//, "");
  // logical name is everything up to the optional ".HASH" suffix.
  const logical = base.replace(/\.[A-Z0-9]{8}(?=\.js$)/, "");
  jsManifest[`js/${logical}`] = file;
}

await writeFile("dist/js-manifest.json", JSON.stringify(jsManifest, null, 2));
console.log("[build-js] emitted", Object.keys(jsManifest).length, "bundles");
