# run air to detect any go file changes to re-build and re-run the server.
live/server:
	air -c .air.toml

# run tailwindcss to generate the styles.css bundle in watch mode.
live/tailwind:
	npx @tailwindcss/cli -i ./ui/styles/style.css -o ./dist/style.css --config ./tailwind.config.js --watch

# run esbuild to generate the index.js bundle in watch mode.
live/esbuild:
	npx --yes esbuild js/index.ts --bundle --outdir=assets/ --watch

# run templ generation in watch mode to detect all .templ files and 
# re-create _templ.txt files on change, then send reload event to browser. 
# Default url: http://localhost:3002
live/templ:
	TEMPL_DEV_PORT=7331 templ generate --watch --proxy="http://localhost:3002" --open-browser=false -v

# watch for any js or css change in the assets/ folder, then reload the browser via templ proxy.
live/sync_assets:
	air -c .air.assets.toml

# start all 4 watch processes in parallel.
live:
	make -j4 live/templ live/server live/tailwind live/sync_assets
