# run templ generation in watch mode to detect all .templ files and 
# re-create _templ.txt files on change, then send reload event to browser. 
# Default url: http://localhost:7331
live/templ:
	templ generate --watch --proxy="http://localhost:7331" --open-browser=false -v

# run air to detect any go file changes to re-build and re-run the server.
live/server:
	air -c .air.toml

# run tailwindcss to generate the styles.css bundle in watch mode.
live/tailwind:
	npx @tailwindcss/cli -i ./ui/styles/style.css -o ./dist/style.css --watch

# watch for any js or css change in the assets/ folder, then reload the browser via templ proxy.
live/sync_assets:
	air -c .air.assets.toml

# start all 4 watch processes in parallel.
# Seed dist/ first so the Go server's manifest loader has something to read on
# startup. Dev mode skips hashing/gzip so subsequent rebuilds reuse stable
# filenames and no server restart is needed.
live:
	@mkdir -p tmp/assets
	@CHESS_BUILD_DEV=1 yarn build --silent
	make -j4 live/templ live/server live/tailwind live/sync_assets
