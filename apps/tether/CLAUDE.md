# Tether

Remote terminal PWA — phone tethered to your Mac. Renders a full PTY session via xterm.js with voice and keyboard input over WebSocket.

## Architecture

Two components sharing a single `package.json`:

- **Relay Server** (`server/src/`) — Node.js + TypeScript. Express serves the PWA over HTTPS (port 5100), WebSocket (`ws`) bridges the phone to tmux sessions via `node-pty`. Each tab maps to a tmux session (`tether-{tabId}`). A singleton `TerminalManager` persists across WebSocket reconnections.
- **PWA** (`pwa/src/`) — SolidJS + TypeScript, built with Vite (port 5101). Tailwind CSS v4 for styling. Uses browser-native Web Speech API for STT/TTS. No audio leaves the phone — only transcribed text is sent over WebSocket.

```
Phone (PWA) ◄──WSS (same WiFi)──► Relay Server :5100 ◄──node-pty──► tmux sessions ◄──► Shell/Claude
```

## Tech Stack

| Layer | Tech |
|-------|------|
| Server | Node.js, TypeScript, Express, ws, node-pty, tmux, strip-ansi |
| Server dev | tsx (runtime), tsconfig strict |
| PWA | SolidJS, TypeScript, Vite |
| PWA styling | Tailwind CSS v4 (no custom CSS, extend @theme for colors) |
| PWA libs | marked (markdown), highlight.js (syntax) |
| Speech | Web Speech API — browser-native, zero server-side audio processing |
| HTTPS | mkcert self-signed certs (app-local, not shared with other apps) |
| PWA install | manifest.json + service worker |
| Lint | ESLint |
| Test | Vitest |

## Project Structure

```
tether/
├── package.json              # Single package.json for both server and PWA
├── project.json              # Nx targets
├── tsconfig.json             # Base TypeScript config
├── vite.config.ts            # Vite config (SolidJS + Tailwind plugins)
├── vitest.config.ts          # Test config
├── eslint.config.js          # ESLint flat config
├── server/
│   ├── tsconfig.json         # Server-specific TS overrides
│   └── src/
│       ├── index.ts          # Express + WSS entry point (port 5100)
│       ├── handler.ts        # WebSocket message handler
│       ├── tmux.ts            # tmux session lifecycle (create, attach, capture, kill)
│       ├── terminal.ts       # Singleton terminal manager: tmux-backed sessions, reattach + scrollback replay
│       ├── permissions.ts    # Detect permission patterns in stdout
│       ├── ansi.ts           # ANSI escape code stripping
│       └── auth.ts           # Shared secret validation
├── pwa/
│   ├── index.html
│   ├── tsconfig.json         # PWA-specific TS overrides (jsx: preserve, jsxImportSource: solid-js)
│   ├── src/
│   │   ├── index.tsx         # SolidJS render entry
│   │   ├── App.tsx           # Root component with providers
│   │   ├── app.css           # Tailwind entry (@import "tailwindcss" + @theme)
│   │   ├── components/       # UI components (StatusBar, ChatArea, InputBar, etc.)
│   │   ├── context/          # SolidJS context providers (settings, connection, chat)
│   │   └── lib/              # Utilities (websocket, stt, tts, renderer, battery)
│   └── public/
│       ├── manifest.json
│       ├── sw.js             # Service worker (stays JS — runs outside Vite)
│       └── icons/
├── certs/                    # mkcert generated (gitignored)
├── scripts/
│   ├── setup.sh
│   └── generate-cert.sh
└── tether-requirements.md
```

## Commands

```bash
# Development (launches both server and PWA dev server together)
nx dev tether

# Build
nx build tether

# Lint
nx lint tether

# Test
nx test tether
```

## Ports

| Service | Port |
|---------|------|
| Relay server (Express + WSS) | 5100 |
| Vite dev server (PWA) | 5101 |

## Code Conventions

Inherits all TypeScript guidelines from root `CLAUDE.md`, plus:

- **SolidJS for PWA**: Use SolidJS signals, stores, and context for state. Components are `.tsx` files.
- **Tailwind only**: No custom CSS classes. All styling via Tailwind utility classes. Extend theme colors in `pwa/src/app.css` under `@theme`.
- **Single package.json**: Both server and PWA share one package.json at the app root.
- **Own certs**: Uses app-local mkcert certs in `certs/` — does not share with other apps.

### Interactive Element Hover States

Every clickable/tappable element **must** have a visible hover state. No exceptions.

| Element Type | Hover Pattern |
|-------------|---------------|
| Solid bg buttons (primary, error) | `hover:bg-{color}-hover` (darker shade) |
| Ghost/transparent buttons | `hover:bg-muted` (subtle fill appears) |
| Muted bg buttons | `hover:bg-surface-raised` |
| Icon-only buttons | `hover:bg-muted` + `rounded-md` for hit area |
| Text-styled buttons (✕, links) | `hover:bg-muted` + `hover:text-ink` |
| Circular buttons (mic, send) | `hover:bg-{color}-hover` |

Rules:
- Always include `transition-colors` or `transition-all` for smooth hover
- Never rely solely on cursor change — there must be a visual background or color shift
- `active:scale-95` can complement hover but does not replace it
- Disabled elements: no hover, use `opacity-50 cursor-not-allowed`

## Tailwind Theme Colors

Defined in `pwa/src/app.css` under `@theme`:

| Token | Hex | Usage |
|-------|-----|-------|
| bg | #0f1117 | Page background |
| surface | #1a1d27 | Cards, panels |
| surface-hover | #242733 | Interactive surface hover |
| accent | #6c5ce7 | Primary (mic button, links, highlights) |
| accent-hover | #5a4bd1 | Primary hover |
| recording | #ff4757 | Recording state, pulsing dot |
| connected | #2ed573 | Connection indicator |
| warning | #ffa502 | Permission prompts |
| error | #ff6b81 | Disconnected, errors |
| text-primary | #f1f2f6 | Main text |
| text-secondary | #747d8c | Timestamps, labels |
| text-dim | #57606f | Placeholders |
| border | #2f3542 | Borders, dividers |

## WebSocket Protocol

All messages are JSON. Direction indicated as Phone→Server or Server→Phone.

```
Phone→Server:  { type: "create-tab", tabId: string, cols?: number, rows?: number }
Phone→Server:  { type: "close-tab", tabId: string }
Phone→Server:  { type: "text", tabId: string, data: string }
Phone→Server:  { type: "permission_response", tabId: string, value: "yes"|"no"|"always" }
Phone→Server:  { type: "stop", tabId: string }
Phone→Server:  { type: "resize", tabId: string, cols: number, rows: number }
Phone→Server:  { type: "key", tabId: string, data: string }

Server→Phone:  { type: "tab-created", tabId: string, restored: boolean }
Server→Phone:  { type: "pty", tabId: string, data: string }
Server→Phone:  { type: "pty-replay", tabId: string, data: string }         # Scrollback replay on reattach
Server→Phone:  { type: "command-complete", tabId: string }
Server→Phone:  { type: "permission", tabId: string, action: string, options: string[] }
Server→Phone:  { type: "tab-exited", tabId: string, exitCode: number }
Server→Phone:  { type: "sessions-available", tabIds: string[] }            # Sent on connect when sessions exist
Server→Phone:  { type: "battery", level: number, charging: boolean }
```

## DOM Layer Architecture

Overlays use **DOM source order** for stacking — no z-index needed. Each layer is a direct child of `<body>` defined in `pwa/index.html`. Later elements paint on top of earlier ones. Components render into their target layer via SolidJS `<Portal mount={...}>`.

```html
<body>
  <div id="app"></div>          <!-- 1. Main app (terminal, toolbar, status bar) -->
  <div id="keys-layer"></div>   <!-- 2. Keyboard helper overlay -->
  <div id="toast-layer"></div>  <!-- 3. Toast notifications (highest priority) -->
</body>
```

### Rules

- **Never use z-index** — stacking is controlled purely by DOM order in the body
- **Each overlay gets its own layer** — add a new `<div id="...-layer">` in `index.html` at the appropriate position
- **Use `<Portal>`** — overlay components mount into their layer via `<Portal mount={document.getElementById("...-layer")!}>`
- **Layers are positioned `fixed`** — overlay content within a layer uses `fixed` positioning to float over the app
- **Overlay style** — floating overlays should have margin from edges (`left-3 right-3`), `rounded-lg`, `shadow-lg`, and `border border-edge` to visually separate from the main content
- **Priority order** (bottom to top): app content → floating panels/keys → toasts → modals → critical alerts

### Adding a New Layer

1. Add `<div id="my-layer"></div>` in `pwa/index.html` at the correct priority position
2. Create a component that uses `<Portal mount={document.getElementById("my-layer")!}>`
3. Use `fixed` positioning with appropriate offsets (e.g., `bottom-16` to clear the toolbar)
4. Conditionally render the component — when unmounted, the layer div is empty and invisible

## Key Design Decisions

- **Web Speech API for all speech processing** — no whisper.cpp, no server-side audio. STT and TTS both run in the browser. The server only handles text strings.
- **tmux-backed terminal sessions** — each tab maps to a tmux session (`tether-{tabId}`). Sessions survive client disconnects, browser reloads, and server restarts. node-pty attaches to tmux for I/O piping. Scrollback replayed via `tmux capture-pane -e` on reattach.
- **Singleton TerminalManager** — one instance shared across all WebSocket connections. On disconnect, sessions are detached (not destroyed). On reconnect, existing tmux sessions are reattached with scrollback replay. A 5-minute grace timer kills sessions with no connected client.
- **Permission detection** — parse stdout for `Allow ...? [y/n/a]` patterns and send structured messages to the PWA instead of raw text.
- **Single user, single session** — no multi-user auth needed for v1. Shared secret token for basic access control. New connections replace the previous client.
- **HTTPS required** — Web Speech API on mobile requires a secure context. Use mkcert for trusted local HTTPS.
- **Orphan cleanup** — on server start, any leftover `tether-*` tmux sessions from previous runs are killed. On SIGTERM/SIGINT, all sessions are destroyed cleanly.

## Implementation Phases

Build in this order — each phase is independently testable:

1. **Core (MVP)**: Server + WSS + PTY + ANSI strip + PWA shell + hold-to-talk + chat display + TTS
2. **Permissions & Polish**: Permission detection + card UI + voice permissions + stop button + reconnect + text input
3. **UX**: Waveform viz + battery display + settings panel + syntax highlighting + themes
4. **Hardening**: Auth + error handling + service worker caching + PWA install + session persistence
5. **Multi-Tab**: Multiple terminal tabs with per-pane tab bars + split-pane layout
6. **Persistence**: Pane/tab localStorage persistence + tmux-backed session persistence + graceful cleanup

## Requirements

Full specification: `tether-requirements.md` (in this directory)
