# Walkie Talkie — TODO

## Architecture Note

The app has evolved from a chat-bubble UI to a **full terminal emulator** using xterm.js.
The PWA now renders raw PTY output (including ANSI codes, colors, cursor movement) directly
in an xterm.js instance. Users interact with Claude Code's terminal natively — typing goes
to the PTY, output renders as-is. Voice input (mic button) sends transcribed text as a prompt.

Key changes from original requirements:
- No ANSI stripping needed — xterm.js renders ANSI natively
- No chat bubbles, markdown rendering, or syntax highlighting — terminal handles display
- No separate "streaming response" or "output_complete" messages — raw PTY data streamed
- Text input goes directly to terminal (xterm keyboard), not via a separate input box
- TerminalToolbar provides mobile helper keys (Enter, Tab, Esc, arrows, Ctrl+C)
- Battery info comes from the server (laptop) via WebSocket, not local device

## Phase 1: Core MVP — Done

| ID | Requirement | Status | Notes |
|----|-------------|--------|-------|
| FR-01 | Hold mic to record | Done | MicButton with pointer events |
| FR-02 | Real-time STT via Web Speech API | Done | stt.ts |
| FR-03 | Interim transcription shown live | Done | MicButton shows interim |
| FR-06 | Recording duration timer | TODO | No timer display |
| FR-07 | Send transcribed text via WebSocket | Done | MicButton → send |
| FR-08 | Server forwards text to PTY stdin | Done | handler.ts |
| FR-09 | Spawn Claude Code in PTY | Done | terminal.ts |
| FR-10 | Stream stdout to PWA in real-time | Done | Raw PTY data via WebSocket → xterm.js |
| FR-11 | Render terminal output | Done | xterm.js renders ANSI/colors natively |
| FR-12 | Stop button (Ctrl+C) | Done | Ctrl+C in TerminalToolbar |
| FR-13 | Multiple conversation turns | Done | Same PTY session |

## Phase 2: Permissions & Polish — 60% done

| ID | Requirement | Status | Notes |
|----|-------------|--------|-------|
| FR-04 | Slide-to-cancel recording | TODO | No gesture handler |
| FR-14 | Detect permission prompts | Done | permissions.ts |
| FR-15 | Send structured permission messages | Done | terminal.ts |
| FR-16 | Permission card UI with buttons | N/A | Users respond via terminal keyboard directly |
| FR-17 | Voice-based permission responses | TODO | Could auto-type "y"/"n" from voice |
| FR-18 | Forward permission response to PTY | Done | handler.ts (key messages) |
| FR-29 | Terminal keyboard input | Done | xterm.js native keyboard + TerminalToolbar |
| FR-30 | Mobile helper keys | Done | TerminalToolbar (Enter, Tab, Esc, arrows, Ctrl+C) |
| FR-34 | WebSocket connection | Done | websocket.ts |
| FR-35 | Connection status indicator | Done | StatusBar green/red dot |
| FR-36 | Auto-reconnect with backoff | Done | websocket.ts |
| FR-37 | Configure IP/port in settings | Done | ConnectScreen |

## Phase 3: UX Enhancements — 40% done

| ID | Requirement | Status | Notes |
|----|-------------|--------|-------|
| FR-05 | Waveform visualization | TODO | No Web Audio API integration |
| FR-19 | TTS on response complete | TODO | tts.ts exists but not wired (needs rethink for terminal mode) |
| FR-20 | Auto-read toggle in settings | TODO | Setting exists, not wired |
| FR-21 | Replay button on responses | N/A | No chat bubbles in terminal mode |
| FR-22 | TTS speed configurable | TODO | Setting exists, not wired |
| FR-25 | Syntax highlighted code blocks | N/A | xterm.js handles terminal colors natively |
| FR-31 | Server battery percentage | Done | Server sends via WebSocket (pmset) |
| FR-32 | Battery display | Done | StatusBar shows server battery |
| FR-33 | Charging indicator | Done | Server sends charging state |
| FR-38 | Settings panel via hamburger | TODO | Button exists, no panel |
| FR-40 | Settings: TTS voice selection | TODO | Setting exists, no UI |
| FR-41 | Settings: auto-read toggle | TODO | Setting exists, no UI |
| FR-42 | Settings: TTS speed | TODO | Setting exists, no UI |
| FR-43 | Settings: theme toggle | Done | StatusBar toggle + terminal light/dark themes |
| FR-44 | Settings: font size | TODO | Setting exists, no UI |
| FR-45 | Settings persist in localStorage | Done | settings.tsx |

## Phase 4: Hardening — 0% done

| ID | Requirement | Status | Notes |
|----|-------------|--------|-------|
| NFR-06 | Shared secret auth | TODO | auth.ts exists but never called |
| FR-46 | Installable PWA | TODO | Manifest exists, icons missing |
| FR-47 | Standalone mode | Done | manifest.json has standalone |
| FR-48 | Cache static assets | TODO | sw.js only caches / and /manifest.json |
| FR-49 | App icon on home screen | TODO | icons/ directory empty |
| — | Error handling | TODO | No error boundaries or user-facing errors |
| — | Session persistence | TODO | Terminal state lost on reload |

## Recommended build order

1. FR-06 — Recording timer on MicButton
2. FR-38/40-44 — Settings panel UI
3. FR-05 — Waveform visualization
4. FR-04 — Slide-to-cancel
5. FR-19 — TTS (rethink: could read last N lines of terminal output)
6. NFR-06 — Wire auth on WebSocket upgrade
7. FR-46/49 — Generate PWA icons
8. FR-48 — Service worker asset caching
