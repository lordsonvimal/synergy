# ChessLab — TODO

## 1. Nx Integration

- [x] Add `project.json` with `dev`, `serve`, `build`, and `lint` targets (model after `apps/core/project.json`)
- [x] `dev` target: run `make live` (parallel templ + air + tailwind)
- [x] `serve` target: run compiled binary (`./dist/main`)
- [x] `build` target: `templ generate` + `go build` + Tailwind build
- [x] `lint` target: `golangci-lint run`
- [x] Add `"dev"` script to `package.json` so Nx can discover it
- [x] Fix `build` script output path — now uses `./dist/style.css`
- [x] Add `"private": true` to `package.json`
- [x] Remove `"main": "index.js"` from `package.json` (not applicable to Go app)
- [x] Remove `package-lock.json` — repo uses Yarn

## 2. Design System (Everwise Crest Tokens)

- [x] Import `@synergy/theme/tokens.css` and `@synergy/theme/dark.css` in `ui/styles/style.css`
- [x] Configure Tailwind to resolve the theme package path (Tailwind v4 uses CSS imports; removed dead `tailwind.config.js`)
- [x] Replace all default Tailwind palette classes with brand tokens:

| Location | Current | Replace With |
|----------|---------|--------------|
| `layout.templ` body | `bg-gray-100` | `bg-canvas` |
| `gamemodes.templ` body | `bg-gray-100` | `bg-canvas` |
| `gamemodes.templ` card | `bg-white` | `bg-surface` |
| `gamemodes.templ` card hover | `hover:bg-blue-50` | `hover:bg-primary-subtle` |
| `gamemodes.templ` variant text | `text-gray-600` | `text-ink-secondary` |
| `gamemodes.templ` time text | `text-gray-500` | `text-ink-dim` |
| `gamemodes.templ` card radius | `rounded-xl` | `rounded-md` (button-level) |
| `newgame.templ` body | `bg-gray-100` | `bg-canvas` |
| `chesssquare.templ` light square | `bg-white` | `bg-surface` |
| `chesssquare.templ` dark square | `bg-gray-300` | `bg-muted` |
| `chesssquare.templ` target ring | `ring-blue-600 bg-blue-200/30` | `ring-primary bg-primary-subtle` |
| `chesssquare.templ` selected ring | `ring-yellow-400 bg-yellow-200/30` | `ring-accent bg-accent-subtle` |
| `gameinfopanel.templ` panel | `bg-white shadow-lg rounded-xl` | `bg-surface border border-edge rounded-lg shadow-md` |
| `gameinfopanel.templ` heading | `text-gray-900` | `text-ink` |
| `gameinfopanel.templ` labels | `text-gray-700` | `text-ink-secondary` |
| `gameinfopanel.templ` white turn | `bg-blue-600` | `bg-primary` |
| `gameinfopanel.templ` black turn | `bg-gray-800` | Use `bg-ink` or contextual token |
| `gameinfopanel.templ` check badge | `bg-red-600` | `bg-error` |
| `gameinfopanel.templ` game state | `bg-yellow-100 text-yellow-800` | `bg-warning-subtle text-warning` |
| `gameinfopanel.templ` winner badge | `bg-green-600` | `bg-success` |
| `promotionoverlay.templ` dialog | `bg-white rounded shadow-lg` | `bg-surface-raised rounded-xl shadow-xl border border-edge` |
| `promotionbutton.templ` hover | `hover:bg-gray-200` | `hover:bg-muted` |

## 3. Dark Mode

- [x] Add `data-theme` attribute support on `<html>`
- [x] Add inline script in `<head>` to apply theme before first paint (prevent FOUC)
- [x] Respect `prefers-color-scheme: dark` as system default
- [x] Store user preference in `localStorage` key `theme`
- [x] Add theme toggle button in game UI
- [ ] Test all components in both light and dark themes

## 4. Semantic HTML & Accessibility

- [ ] Add `lang="en"` to `<html>` in `layout.templ` and `newgame.templ`
- [ ] Add `<meta charset="UTF-8">` to `layout.templ` and `newgame.templ`
- [ ] Add `<meta name="viewport" content="width=device-width, initial-scale=1">` to all pages
- [ ] Add `aria-label` to chess squares (e.g., "e4, White Pawn")
- [ ] Add `role="button"` and `tabindex="0"` to clickable `<td>` squares
- [ ] Add keyboard event handling (`Enter`/`Space`) for square selection
- [ ] Add `aria-label` to promotion buttons (e.g., "Promote to Queen")
- [ ] Add visible focus indicators on all interactive elements
- [ ] Use ARIA live region (`aria-live="polite"`) for game state changes (check, checkmate, turn)
- [ ] Wrap chess piece Unicode in `<span>` with `aria-label` for screen reader context

## 5. Responsive Design

- [ ] Add viewport meta tag to all pages
- [ ] Implement mobile layout: stack board above game info panel vertically on small screens
- [ ] Make game info panel (`aside w-72`) responsive — full-width below `lg` breakpoint
- [ ] Add app shell: top bar with game title, back navigation, theme toggle
- [ ] Constrain game modes page to `max-w-2xl mx-auto` (already partially done) with proper padding
- [ ] Test at 375px, 768px, 1024px breakpoints

## 6. Layout Consolidation

- [x] Use `layout.templ` as the shared wrapper for all pages (currently unused)
- [x] Remove duplicate `<!DOCTYPE html>` / `<head>` blocks from `gamemodes.templ` and `newgame.templ`
- [x] Standardize static asset path: `/static/style.css` everywhere
- [x] Move Datastar script and shared head elements into the layout

## 7. Security

- [ ] Validate WebSocket `CheckOrigin` — currently accepts all origins
- [ ] Add CSRF protection on POST forms (`/game`, `/game/:gameID/select/:square`)
- [ ] Bundle Datastar locally instead of loading from CDN (local-first principle)

## 8. Error Handling & UX

- [ ] Add loading state on game mode form submission
- [ ] Add user-friendly error UI for failed game creation (not bare `c.String`)
- [ ] Add back navigation from game page to game modes
- [ ] Add empty state for game modes page (when no modes available)
- [ ] Add error recovery UI (retry buttons)
- [ ] Surface silent failures (SSE disconnect, WebSocket drop) to user via toast/banner

## 9. Performance & Dead Code

- [ ] Gate `requestAnimationFrame` loop in `initClock.js` — only run when clock elements exist
- [ ] Fix `sync.js` — `document.getElementById` at module top will fail if DOM not ready
- [ ] Wire clock UI elements into templates (referenced in JS but never rendered)
- [ ] Remove or wire `GameWS` handler into routes (defined but unused)
- [ ] Remove stale `esbuild` Makefile target (references non-existent `js/index.ts`)
- [x] Fix `tailwind.config.js` — removed (Tailwind v4 uses CSS-based config, file was dead)
- [ ] Move WAL file output to a configurable data directory (not app root)

## 10. Testing

- [ ] Add `data-testid` attributes to all interactive elements
- [ ] Add `.env.example` documenting expected environment variables
- [ ] Add basic Go tests for game logic and handlers

---

## Priority

1. Nx integration — app is invisible to monorepo toolchain without it
2. Theme tokens — import shared theme, replace default Tailwind classes
3. Viewport meta + responsive layout — currently broken on mobile
4. Dark mode — required for all apps
5. Accessibility — keyboard support, ARIA, focus indicators
6. Layout consolidation — remove duplication
7. Security — origin validation, CSRF, bundle CDN deps
8. Remaining — dead code, error handling, testing
