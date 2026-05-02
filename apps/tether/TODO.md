# Tether — TODO

## Architecture Note

The app has evolved from a chat-bubble UI to a **full remote terminal** using xterm.js.
The PWA now renders raw PTY output (including ANSI codes, colors, cursor movement) directly
in an xterm.js instance. The server spawns a full login shell (not just Claude) — users can
run any command including `sudo`, `cd`, `claude`, etc.

Key architecture decisions:
- No ANSI stripping — xterm.js renders ANSI natively
- No chat bubbles, markdown, or syntax highlighting — terminal handles all display
- Raw PTY data streamed via `{ type: "pty", data }` messages
- Keyboard input forwarded via xterm.js `onData` → WebSocket → PTY
- Voice input: click-to-toggle mic, transcription accumulates, review/edit before send
- TerminalToolbar: keyboard toggle (keys overlay via DOM layer), mic button
- Keys overlay renders into `#keys-layer` (DOM layer pattern, no z-index)
- Toast notifications render into `#toast-layer` (highest priority layer)
- Battery info comes from the server (laptop `pmset`) via WebSocket every 60s
- Terminal supports light/dark themes reactively
- **tmux-backed sessions** — each tab maps to a tmux session (`tether-{tabId}`), surviving disconnects, reloads, and server restarts. Singleton TerminalManager with 5-min grace timer. Scrollback replayed on reattach via `tmux capture-pane -e`.

## Phase 1: Core MVP — Done

| ID | Requirement | Status | Notes |
|----|-------------|--------|-------|
| FR-01 | Voice recording (click to toggle) | Done | VoiceInput with STT, auto-restart on pause |
| FR-02 | Real-time STT via Web Speech API | Done | stt.ts with continuous mode |
| FR-03 | Interim transcription shown live | Done | Displayed next to mic during recording |
| FR-06 | Recording duration timer | Done | MM:SS display next to mic during recording |
| FR-07 | Send transcribed text via WebSocket | Done | VoiceInput → review → send |
| FR-08 | Server forwards text to PTY stdin | Done | handler.ts |
| FR-09 | Spawn full login shell in PTY | Done | terminal.ts — any command, any folder |
| FR-10 | Stream stdout to PWA in real-time | Done | Raw PTY data via WebSocket → xterm.js |
| FR-11 | Render terminal output | Done | xterm.js renders ANSI/colors natively |
| FR-12 | Stop button (Ctrl+C) | Done | Ctrl+C in KeysOverlay |
| FR-13 | Multiple commands in same session | Done | Same PTY session (full shell) |

## Phase 2: Permissions & Polish — Done

| ID | Requirement | Status | Notes |
|----|-------------|--------|-------|
| FR-04 | Slide-to-cancel recording | N/A | Click-to-toggle replaced hold-to-talk |
| FR-14 | Detect permission prompts | Done | permissions.ts |
| FR-15 | Send structured permission messages | Done | terminal.ts |
| FR-16 | Permission card UI with buttons | N/A | Users respond via terminal keyboard directly |
| FR-17 | Voice-based permission responses | N/A | Users type directly via terminal keyboard/KeysOverlay |
| FR-18 | Forward permission response to PTY | Done | handler.ts (key messages) |
| FR-29 | Terminal keyboard input | Done | xterm.js native keyboard + KeysOverlay |
| FR-30 | Mobile helper keys | Done | KeysOverlay (Enter, Tab, Esc, arrows, Ctrl+C) |
| FR-34 | WebSocket connection | Done | websocket.ts |
| FR-35 | Connection status indicator | Done | StatusBar green/red dot |
| FR-36 | Auto-reconnect with backoff | Done | websocket.ts |
| FR-37 | Configure IP/port in settings | Done | ConnectScreen |
| — | Voice review/edit before send | Done | VoiceInput textarea with send/cancel |
| — | Toast notification system | Done | lib/toast.ts + ToastLayer (DOM layer) |
| — | Keys overlay (no layout shift) | Done | KeysOverlay via Portal into #keys-layer |

## Phase 3: UX Enhancements — Done

| ID | Requirement | Status | Notes |
|----|-------------|--------|-------|
| FR-05 | Waveform visualization | Done | Amplitude bars via Web Audio API AnalyserNode during recording |
| FR-19 | TTS on response complete | N/A | Replaced by completion chime (FR-50) — TTS impractical for raw terminal output |
| FR-20 | Auto-read toggle in settings | N/A | Replaced by completion chime toggle |
| FR-21 | Replay button on responses | N/A | No chat bubbles in terminal mode |
| FR-22 | TTS speed configurable | N/A | TTS dropped — not useful for terminal output |
| FR-25 | Syntax highlighted code blocks | N/A | xterm.js handles terminal colors natively |
| FR-31 | Server battery percentage | Done | Server sends via WebSocket (pmset) |
| FR-32 | Battery display | Done | StatusBar shows server battery |
| FR-33 | Charging indicator | Done | Server sends charging state |
| FR-38 | Settings panel via hamburger | Done | Slide-in panel with theme, font size, chime toggle, disconnect |
| FR-40 | Settings: TTS voice selection | N/A | TTS dropped |
| FR-41 | Settings: auto-read toggle | N/A | TTS dropped — replaced by chime toggle in FR-50 |
| FR-42 | Settings: TTS speed | N/A | TTS dropped |
| FR-50 | Completion chime | Done | Synthesized two-tone chime via Web Audio API on 2s idle after output |
| FR-43 | Settings: theme toggle | Done | StatusBar toggle + terminal light/dark themes |
| FR-44 | Settings: font size | Done | Segmented control (small/medium/large) in settings panel, reactively updates terminal |
| FR-45 | Settings persist in localStorage | Done | settings.tsx |
| — | Hover states on all interactive elements | Done | Documented in CLAUDE.md |
| — | DOM layer architecture | Done | No z-index, body child order |

## Phase 3b: New Features — Done

| ID | Requirement | Status | Notes |
|----|-------------|--------|-------|
| FR-50 | Completion chime | Done | Server-side process + idle detection, synthesized chime via Web Audio API |
| FR-50a | Chime toggle in settings | Done | Toggle switch in settings panel, persisted in localStorage |
| FR-51 | Shortcut command center panel | Done | Slide-up sheet via Portal into #shortcuts-layer, bolt icon in toolbar |
| FR-52 | Shortcut: label + command, stored in localStorage | Done | Shortcut interface in settings context, persisted via settings |
| FR-53 | Tap shortcut to send to terminal | Done | Tap sends command via WebSocket and closes panel |
| FR-54 | Long-press shortcut to edit before sending | Done | 500ms long-press opens inline edit prompt with editable command |
| FR-55 | Add/edit/reorder/delete shortcuts in settings | Done | Full CRUD with reorder arrows in settings panel Shortcuts section |
| FR-56 | Default shortcuts on first launch | Done | git status, claude, ls -la, cd ~ |
| FR-58 | Remote access | Done | Server + Vite bind 0.0.0.0; works on local WiFi, docs cover Tailscale/Cloudflare Tunnel for non-WARP machines |
| FR-59 | ConnectScreen accepts any IP/hostname | Done | Works out of the box — any IP/hostname accepted |
| FR-60 | Server binds to 0.0.0.0 | Done | server.listen(PORT, "0.0.0.0") |
| FR-61 | Remote access documentation | Done | setup.sh documents local WiFi, Cloudflare Tunnel, and Tailscale options |
| FR-62 | HTTPS cert for extra IPs | Done | generate-cert.sh accepts extra SANs as args (e.g. Tailscale IP) |

## Phase 4: Hardening — Done

| ID | Requirement | Status | Notes |
|----|-------------|--------|-------|
| NFR-06 | Shared secret auth | Done | Server validates token on WS upgrade, PWA sends via query param, secret field in ConnectScreen |
| FR-46 | Installable PWA | Done | Manifest + icons (192, 512, SVG, apple-touch, favicon) |
| FR-47 | Standalone mode | Done | manifest.json has standalone |
| FR-48 | Cache static assets | Done | SW caches shell (network-first), hashed assets + icons (cache-first), Google Fonts (cache-first); auto-purges old caches on version bump |
| FR-49 | App icon on home screen | Done | All sizes generated from SVG source |
| — | Error handling | Done | Toast system for user-facing errors |
| — | Session persistence | Moved | Subsumed by Phase 5 (FR-72) — tmux-backed tabs handle persistence |

## Phase 5: Multi-Tab Terminals — Done

| ID | Requirement | Status | Notes |
|----|-------------|--------|-------|
| FR-70 | Multiple terminal tabs | Done | Each tab owns an independent PTY via TerminalManager; server routes messages by tabId |
| FR-71 | Tab bar UI | Done | Create, switch, close tabs; active tab visually distinguished; double-click to rename |
| FR-73 | Tab metadata | Done | User-assignable label per tab via double-click rename in tab bar |

## Phase 6: Split-Pane Layout — Done

| ID | Requirement | Status | Notes |
|----|-------------|--------|-------|
| FR-80 | Binary tree split layout | Done | Recursive LeafNode/BranchNode tree in panes.tsx context; SplitContainer renders tree |
| FR-81 | Split via context menu | Done | Right-click tab → Split Right / Split Down; creates new pane with fresh terminal |
| FR-82 | Split via drag-and-drop | Done | Drag tab to edge of terminal body; blue dashed overlay previews split zone |
| FR-83 | Resizable dividers | Done | Divider component with pointer capture; ratio clamped 15%-85% |
| FR-84 | Active pane indicator | Done | Absolute-positioned border overlay (2px primary active, 1px edge inactive) on all 4 sides |
| FR-85 | Auto-collapse on narrow viewports | Done | matchMedia (max-width: 640px) forces vertical direction |
| FR-86 | Merge pane | Done | Context menu → Merge Pane with confirmation dialog showing target pane |
| FR-87 | Tab reorder within pane | Done | Drag tab within same tab bar; insertion line indicator tracks cursor via midpoint calculation |
| FR-88 | Tab move between panes | Done | Drag tab to another pane's tab bar to move it; handles empty-pane collapse |
| FR-89 | Close pane on last tab close | Done | closeTab removes pane when last tab closed; sibling fills space |
| FR-90 | Per-pane tab bar | Done | Overflow dropdown, add button, context menu (rename, split, merge, close others/left/right) |
| FR-91 | Global tab search | Done | Cmd/Ctrl+K or StatusBar icon; Portal into #search-layer; arrow keys + Enter + Escape |
| FR-92 | Search result highlighting | Done | Matched text highlighted; pane number shown when multiple panes exist |
| FR-93 | Search keyboard navigation | Done | Arrow keys, Enter to select, Escape to close |
| FR-94 | Search navigates to tab | Done | Selection activates target pane and switches to selected tab |

## Phase 7: Persistence — Done

| ID | Requirement | Status | Notes |
|----|-------------|--------|-------|
| FR-95 | Pane layout persistence | Done | Split tree (panes, directions, ratios) saved/restored from localStorage; ID counters synced on restore |
| FR-96 | Tab metadata persistence | Done | Tab labels, active tab per pane, active pane ID all persisted; reconnect re-creates all PTYs |
| FR-97 | Terminal session persistence | Done | tmux-backed sessions survive disconnect/reload/server crash; server replays scrollback via `capture-pane -e` on reattach; each tab maps to tmux session `tether-{tabId}` |
| FR-98 | Graceful cleanup | Done | Server sends `tab-exited` on tmux session exit; orphaned `tether-*` sessions cleaned on server start; 5-min grace timer kills sessions with no connected client; SIGTERM/SIGINT handlers destroy all sessions |

## Recommended build order

1. ~~FR-38/44 — Settings panel UI (theme, font size, chime toggle, shortcuts management)~~ ✓
2. ~~FR-51-56 — Shortcut command center~~ ✓
3. ~~FR-50 — Completion chime (idle timeout + prompt detection hybrid)~~ ✓
4. ~~FR-58-62 — Tailscale remote access (server bind + cert + docs)~~ ✓
5. ~~NFR-06 — Wire auth on WebSocket upgrade~~ ✓
6. ~~FR-05 — Waveform visualization~~ ✓
7. ~~FR-46/49 — Generate PWA icons~~ ✓
8. ~~FR-48 — Service worker asset caching~~ ✓
9. ~~FR-70 — Multi-tab terminals~~ ✓
10. ~~FR-71 — Tab bar UI~~ ✓
11. ~~FR-73 — Tab metadata (labels)~~ ✓
12. ~~FR-80-94 — Split-pane layout + global search~~ ✓
13. ~~FR-95/96 — Pane layout + tab metadata persistence (localStorage)~~ ✓
14. ~~FR-97 — Terminal session persistence (tmux-backed reattach)~~ ✓
15. ~~FR-98 — Graceful cleanup (PTY exit notification, orphan cleanup)~~ ✓
