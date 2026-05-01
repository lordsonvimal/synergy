# Walkie Talkie — TODO

## Phase 1: Core MVP — 90% done

| ID | Requirement | Status | Notes |
|----|-------------|--------|-------|
| FR-01 | Hold mic to record | Done | MicButton with pointer events |
| FR-02 | Real-time STT via Web Speech API | Done | stt.ts |
| FR-03 | Interim transcription shown live | Done | MicButton shows interim |
| FR-06 | Recording duration timer | TODO | No timer display |
| FR-07 | Send transcribed text via WebSocket | Done | InputBar → send |
| FR-08 | Server forwards text to PTY stdin | Done | handler.ts |
| FR-09 | Spawn Claude Code in PTY | Done | terminal.ts |
| FR-10 | Stream stdout to PWA in real-time | Done | App.tsx message bridge |
| FR-11 | Strip ANSI codes | Done | ansi.ts |
| FR-13 | Multiple conversation turns | Done | Same PTY session |
| FR-19 | TTS on response complete | TODO | tts.ts exists but never called from App.tsx |
| FR-23 | User chat bubbles | Done | ChatBubble |
| FR-24 | Claude bubbles with markdown | Done | ChatBubble + renderer.ts |
| FR-26 | Timestamps on messages | Done | ChatBubble |
| FR-27 | Auto-scroll | Done | ChatArea |
| FR-28 | Conversation history within session | Done | chat context |

## Phase 2: Permissions & Polish — 30% done

| ID | Requirement | Status | Notes |
|----|-------------|--------|-------|
| FR-04 | Slide-to-cancel recording | TODO | No gesture handler |
| FR-12 | Stop button (Ctrl+C) | TODO | Server stopTerminal() exists, no UI |
| FR-14 | Detect permission prompts | Done | permissions.ts |
| FR-15 | Send structured permission messages | Done | terminal.ts |
| FR-16 | Permission card UI with buttons | TODO | Currently just text in a chat bubble |
| FR-17 | Voice-based permission responses | TODO | No speech recognition for yes/no |
| FR-18 | Forward permission response to PTY | Done | handler.ts |
| FR-29 | Text input fallback | Done | InputBar text mode |
| FR-30 | Keyboard icon to switch mode | Done | InputBar toggle |
| FR-34 | WebSocket connection | Done | websocket.ts |
| FR-35 | Connection status indicator | Done | StatusBar green/red dot |
| FR-36 | Auto-reconnect with backoff | Done | websocket.ts |
| FR-37 | Configure IP/port in settings | Done | ConnectScreen |

## Phase 3: UX Enhancements — 10% done

| ID | Requirement | Status | Notes |
|----|-------------|--------|-------|
| FR-05 | Waveform visualization | TODO | No Web Audio API integration |
| FR-20 | Auto-read toggle in settings | TODO | Setting exists, not wired |
| FR-21 | Replay button on responses | TODO | No play button in ChatBubble |
| FR-22 | TTS speed configurable | TODO | Setting exists, not wired |
| FR-25 | Syntax highlighted code blocks | TODO | highlight.js imported but marked highlight option broken (API change) |
| FR-31 | Battery percentage | Done | StatusBar |
| FR-32 | Graceful hide on iOS | Done | battery.ts |
| FR-33 | Charging indicator | TODO | Battery API wrapper doesn't expose charging |
| FR-38 | Settings panel via hamburger | TODO | Button exists, no panel |
| FR-39 | Settings: IP/port | Done | ConnectScreen |
| FR-40 | Settings: TTS voice selection | TODO | Setting exists, no UI |
| FR-41 | Settings: auto-read toggle | TODO | Setting exists, no UI |
| FR-42 | Settings: TTS speed | TODO | Setting exists, no UI |
| FR-43 | Settings: theme toggle | TODO | Setting exists, no UI or light theme |
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
| — | Session persistence | TODO | Chat lost on reload |

## Recommended build order

Each step is independently testable:

1. FR-19 — Wire TTS auto-read on response complete
2. FR-12 — Stop button UI during streaming
3. FR-16 — Permission card component with Yes/No/Always buttons
4. FR-06 — Recording timer
5. FR-25 — Fix syntax highlighting (migrate to markedHighlight extension)
6. FR-38/40-44 — Settings panel UI
7. FR-05 — Waveform visualization
8. FR-04 — Slide-to-cancel
9. FR-17 — Voice permission responses
10. NFR-06 — Wire auth on WebSocket upgrade
11. FR-46/49 — Generate PWA icons
12. FR-48 — Service worker asset caching
13. Session persistence — IndexedDB for chat history
