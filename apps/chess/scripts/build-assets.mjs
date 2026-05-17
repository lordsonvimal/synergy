// Hash CSS / fonts / favicon, write the unified dist/manifest.json, and
// emit .gz variants for everything text-compressible so the Go static
// middleware can serve precompressed bytes. Dev mode (CHESS_BUILD_DEV=1)
// skips hashing and skips gzip (no value to gzip on localhost).
import { createHash } from "node:crypto";
import {
  copyFile, mkdir, readFile, readdir, rm, stat, writeFile,
} from "node:fs/promises";
import { existsSync } from "node:fs";
import { gzipSync, constants as zlibConstants } from "node:zlib";
import { dirname, extname, join, relative } from "node:path";

const DEV = process.env.CHESS_BUILD_DEV === "1";

const hash = (buf) => createHash("sha256").update(buf).digest("hex").slice(0, 8).toUpperCase();

const hashedName = (logical, buf) => {
  if (DEV) return logical;
  const ext = extname(logical);
  const stem = logical.slice(0, -ext.length);
  return `${stem}.${hash(buf)}${ext}`;
};

// Files we want gzipped at build time. Skip already-compressed formats.
const GZIP_EXT = new Set([".js", ".css", ".svg", ".json", ".map", ".html", ".txt"]);
const shouldGzip = (path) => !DEV && GZIP_EXT.has(extname(path));

async function emitFile(logicalKey, srcPath) {
  const buf = await readFile(srcPath);
  const outName = hashedName(logicalKey, buf);
  const outPath = join("dist", outName);
  await mkdir(dirname(outPath), { recursive: true });
  await writeFile(outPath, buf);
  if (shouldGzip(outName)) {
    const gz = gzipSync(buf, { level: zlibConstants.Z_BEST_COMPRESSION });
    await writeFile(outPath + ".gz", gz);
  }
  return outName;
}

// Walk a directory recursively yielding {logical, abs} pairs for every file.
async function* walk(root, prefix) {
  for (const entry of await readdir(root, { withFileTypes: true })) {
    const abs = join(root, entry.name);
    const logical = prefix ? `${prefix}/${entry.name}` : entry.name;
    if (entry.isDirectory()) yield* walk(abs, logical);
    else yield { logical, abs };
  }
}

// Start from the existing manifest so partial rebuilds (e.g. running just
// `yarn build:js` after iterating on a JS file) preserve entries from the
// last full build instead of dropping them — without this, the Go server's
// manifest loader would 404 on whichever subsystem we didn't rebuild this
// pass (CSS, fonts, favicon, etc.).
let manifest = {};
if (existsSync("dist/manifest.json")) {
  try {
    manifest = JSON.parse(await readFile("dist/manifest.json", "utf8"));
  } catch {
    manifest = {}; // corrupted manifest — start fresh rather than crash
  }
}

// Drop any entries pointing at files that no longer exist on disk so a stale
// hash from a previous build doesn't keep showing up after the source went
// away (this is the cleanup other passes already implicitly did by wiping
// the whole map).
for (const [logical, file] of Object.entries(manifest)) {
  if (!existsSync(join("dist", file))) delete manifest[logical];
}

// 1. CSS (Tailwind's output already exists at dist/style.css thanks to build:css).
if (existsSync("dist/style.css")) {
  const css = await readFile("dist/style.css");
  const outName = hashedName("style.css", css);
  if (outName !== "style.css") {
    await writeFile(join("dist", outName), css);
    if (shouldGzip(outName)) {
      await writeFile(join("dist", outName + ".gz"), gzipSync(css, { level: zlibConstants.Z_BEST_COMPRESSION }));
    }
    await rm("dist/style.css", { force: true });
  } else if (shouldGzip(outName)) {
    await writeFile(join("dist", outName + ".gz"), gzipSync(css, { level: zlibConstants.Z_BEST_COMPRESSION }));
  }
  manifest["style.css"] = outName;
}

// 2. Favicon.
if (existsSync("assets/favicon.svg")) {
  manifest["favicon.svg"] = await emitFile("favicon.svg", "assets/favicon.svg");
}

// 3. Fonts (any file in assets/fonts/ is shipped as-is, hashed).
if (existsSync("assets/fonts")) {
  for await (const { logical, abs } of walk("assets/fonts", "fonts")) {
    manifest[logical] = await emitFile(logical, abs);
  }
}

// 3b. Sounds — copied verbatim to dist/sounds/ (no hashing). sound.js loads
// them by stable URL (/static/sounds/<name>.ogg); browser cache headers
// from the static handler are good enough for files that change rarely.
if (existsSync("assets/sounds")) {
  for await (const { logical, abs } of walk("assets/sounds", "sounds")) {
    const buf = await readFile(abs);
    const out = join("dist", logical);
    await mkdir(dirname(out), { recursive: true });
    await writeFile(out, buf);
  }
}

// 4. JS manifest produced by build-js.mjs.
if (existsSync("dist/js-manifest.json")) {
  const jsManifest = JSON.parse(await readFile("dist/js-manifest.json", "utf8"));
  for (const [logical, file] of Object.entries(jsManifest)) {
    manifest[logical] = file;
    // Also produce .gz for the bundled JS output that build-js wrote.
    const out = join("dist", file);
    if (shouldGzip(file) && existsSync(out)) {
      const buf = await readFile(out);
      await writeFile(out + ".gz", gzipSync(buf, { level: zlibConstants.Z_BEST_COMPRESSION }));
    }
  }
  await rm("dist/js-manifest.json", { force: true });
}

await writeFile("dist/manifest.json", JSON.stringify(manifest, null, 2));
console.log("[build-assets] manifest entries:", Object.keys(manifest).length);
