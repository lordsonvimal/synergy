# ChessLeap

#### Run tailwind in dev
```
npx @tailwindcss/cli -i ./ui/styles/style.css -o ./dist/style.css --watch
```

### Generate templ
```
templ generate
```

#### Live reload templ
```
templ generate --watch --proxy="http://localhost:4000" --cmd="go run ."
```

### Stockfish (required for game analysis)

The analysis worker shells out to a Stockfish subprocess. Install it once:

```bash
# macOS
brew install stockfish

# Ubuntu / Debian
sudo apt-get install stockfish
```

Resolution order at runtime (see `engine/uci/resolve.go`):
1. `CHESSLEAP_STOCKFISH_PATH` env var
2. `engines/stockfish-<os>-<arch>` next to the running binary
3. `stockfish` on `$PATH`

Verify the UCI integration:

```bash
go test ./engine/uci/ -v
```

`TestSpawnAnalyzeMultiPV` and `TestPoolAcquireRelease` skip cleanly if no binary is found.

### Go live reload
```go
go run github.com/air-verse/air@v1.51.0 \
  --build.cmd "go build -o tmp/bin/main" --build.bin "tmp/bin/main" --build.delay "100" \
  --build.exclude_dir "node_modules" \
  --build.include_ext "go" \
  --build.stop_on_error "false" \
  --misc.clean_on_exit true
```
