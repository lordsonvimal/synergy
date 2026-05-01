# Walkie Talkie

Voice-driven PWA that acts as a remote control for Claude Code running on a Mac terminal. Users send voice commands from their phone, have them transcribed and executed by Claude Code, and receive text + audio responses.

## Architecture

Two components sharing a single `package.json`:

- **Relay Server** (`server/src/`) — Node.js + TypeScript. Express serves the PWA over HTTPS (port 5100), WebSocket (`ws`) bridges the phone to a Claude Code pseudo-terminal (`node-pty`).
- **PWA** (`pwa/src/`) — SolidJS + TypeScript, built with Vite (port 5101). Tailwind CSS v4 for styling. Uses browser-native Web Speech API for STT/TTS. No audio leaves the phone — only transcribed text is sent over WebSocket.

```
Phone (PWA) ◄──WSS (same WiFi)──► Relay Server :5100 ◄──node-pty──► Claude Code (Terminal)
```

## Tech Stack

| Layer | Tech |
|-------|------|
| Server | Node.js, TypeScript, Express, ws, node-pty, strip-ansi |
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
walkie-talkie/
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
│       ├── terminal.ts       # PTY manager: spawn Claude, pipe I/O
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
└── walkie-talkie-requirements.md
```

## Commands

```bash
# Development (launches both server and PWA dev server together)
nx dev walkie-talkie

# Build
nx build walkie-talkie

# Lint
nx lint walkie-talkie

# Test
nx test walkie-talkie
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
Phone→Server:  { type: "text", data: string }              # Voice/typed prompt
Phone→Server:  { type: "permission_response", value: "yes"|"no"|"always" }
Phone→Server:  { type: "stop" }                            # Ctrl+C to PTY

Server→Phone:  { type: "output", text: string, streaming: boolean }
Server→Phone:  { type: "output_complete", fullText: string }
Server→Phone:  { type: "permission", action: string, options: string[] }
Server→Phone:  { type: "status", connected: boolean, claudeReady: boolean }
```

## Key Design Decisions

- **Web Speech API for all speech processing** — no whisper.cpp, no server-side audio. STT and TTS both run in the browser. The server only handles text strings.
- **node-pty for Claude Code** — full PTY emulation preserves Claude's interactive behavior (permissions, streaming, colors).
- **ANSI stripping** — Claude Code output contains ANSI escape codes; strip them before sending to the phone.
- **Permission detection** — parse stdout for `Allow ...? [y/n/a]` patterns and send structured messages to the PWA instead of raw text.
- **Single user, single session** — no multi-user auth needed for v1. Shared secret token for basic access control.
- **HTTPS required** — Web Speech API on mobile requires a secure context. Use mkcert for trusted local HTTPS.

## Implementation Phases

Build in this order — each phase is independently testable:

1. **Core (MVP)**: Server + WSS + PTY + ANSI strip + PWA shell + hold-to-talk + chat display + TTS
2. **Permissions & Polish**: Permission detection + card UI + voice permissions + stop button + reconnect + text input
3. **UX**: Waveform viz + battery display + settings panel + syntax highlighting + themes
4. **Hardening**: Auth + error handling + service worker caching + PWA install + session persistence

## Requirements

Full specification: `walkie-talkie-requirements.md` (in this directory)
