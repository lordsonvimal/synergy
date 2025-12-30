# Chess Game

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

### Go live reload
```go
go run github.com/air-verse/air@v1.51.0 \
  --build.cmd "go build -o tmp/bin/main" --build.bin "tmp/bin/main" --build.delay "100" \
  --build.exclude_dir "node_modules" \
  --build.include_ext "go" \
  --build.stop_on_error "false" \
  --misc.clean_on_exit true
```
