# Tether - Requirements Document

**Project**: Tether
**Type**: PWA (Progressive Web App) + Relay Server
**Version**: 1.0
**Date**: 2025-04-25
**Author**: Lordson Vimal

---

## 1. Project Overview

Tether is a voice-driven mobile PWA that acts as a remote control for Claude Code running on a Mac terminal. Users can send voice commands from their phone, have them transcribed and executed by Claude Code, and receive both text and audio responses.

### 1.1 Problem Statement

Claude Code is a powerful CLI tool, but using it requires being physically at the terminal. There is no way to interact with it remotely, hands-free, or via voice.

### 1.2 Solution

A two-part system:
1. **PWA** on the user's phone — a full terminal emulator (xterm.js) with voice input support
2. **Relay Server** on the Mac that bridges the phone to Claude Code's terminal via a pseudo-terminal

The PWA renders raw PTY output directly in xterm.js, giving users the full terminal experience (colors, cursor movement, interactive prompts) rather than a simplified chat-bubble view.

### 1.3 High-Level Architecture

```
┌─────────────┐       WebSocket        ┌──────────────┐       stdin/stdout      ┌─────────────┐
│  Phone PWA  │ ◄────────────────────► │  Relay Server │ ◄────────────────────► │ Claude Code │
│  (Browser)  │       (same WiFi)      │  (Node.js)   │       (node-pty)       │  (Terminal)  │
└─────────────┘                        └──────────────┘                        └─────────────┘
```

---

## 2. Functional Requirements

### 2.1 Voice Recording & Speech-to-Text

| ID | Requirement | Priority |
|----|-------------|----------|
| FR-01 | User must be able to record voice by holding the mic button | Must |
| FR-02 | Speech must be transcribed in real-time (word-by-word) using Web Speech API as the primary STT engine | Must |
| FR-03 | Interim (partial) transcription results must be displayed live as the user speaks | Must |
| FR-04 | User must be able to cancel recording by sliding left (WhatsApp-style) | Should |
| FR-05 | Live waveform visualization must be shown during recording | Should |
| FR-06 | Recording duration timer must be visible | Must |
| FR-07 | On release, the final transcribed text must be sent to relay server via WebSocket (text string, not audio) | Must |
| FR-08 | Transcribed text must be forwarded to Claude Code's PTY as stdin by the relay server | Must |

### 2.2 Claude Code Integration

| ID | Requirement | Priority |
|----|-------------|----------|
| FR-09 | Relay server must spawn Claude Code in a pseudo-terminal (node-pty) | Must |
| FR-10 | Claude Code's stdout must be streamed to PWA in real-time via WebSocket | Must |
| FR-11 | Raw PTY output (including ANSI) must be rendered in xterm.js on the PWA | Must |
| FR-12 | User must be able to stop Claude mid-response (sends Ctrl+C to PTY) | Must |
| FR-13 | Multiple conversation turns must be supported in the same PTY session | Must |

### 2.3 Permission Handling

| ID | Requirement | Priority |
|----|-------------|----------|
| FR-14 | Relay server must detect Claude Code permission prompts in stdout (pattern: `Allow ...? [y/n/a]`) | Must |
| FR-15 | Permission prompts must be sent to PWA as structured messages with action description and options (Yes / No / Always) | Must |
| FR-16 | PWA must display permission prompts as a distinct card UI with tap buttons | Must |
| FR-17 | PWA must support voice-based permission responses using Web Speech API (on-device recognition for "yes", "no", "always") | Should |
| FR-18 | User's permission response must be sent to relay server and forwarded to PTY as the corresponding keystroke (y/n/a) | Must |

### 2.4 Text-to-Speech (TTS)

| ID | Requirement | Priority |
|----|-------------|----------|
| FR-19 | On response completion, Claude's output must be read aloud using Web Speech API (speechSynthesis) | Must |
| FR-20 | User must be able to toggle auto-read on/off in settings | Must |
| FR-21 | User must be able to replay any response via a play button | Should |
| FR-22 | TTS playback speed must be configurable (0.5x to 2.0x) | Should |

### 2.5 Terminal Display

| ID | Requirement | Priority |
|----|-------------|----------|
| FR-23 | PTY output must render in a full xterm.js terminal emulator | Must |
| FR-24 | Terminal must support ANSI colors, cursor movement, and interactive prompts | Must |
| FR-25 | Terminal must support light and dark themes | Done |
| FR-26 | Terminal must auto-scroll on new output | Must |
| FR-27 | Terminal must resize responsively (FitAddon) | Must |
| FR-28 | Terminal session persists within the WebSocket connection | Must |

### 2.6 Terminal Input

| ID | Requirement | Priority |
|----|-------------|----------|
| FR-29 | Users type directly into the terminal via xterm.js native keyboard | Must |
| FR-30 | Mobile toolbar must provide helper keys (Enter, Tab, Esc, arrows, Ctrl+C) | Must |

### 2.7 Server Information

| ID | Requirement | Priority |
|----|-------------|----------|
| FR-31 | Server (laptop) battery percentage must be displayed in the status bar via WebSocket | Should |
| FR-32 | Battery info sent from server every 10 seconds (macOS `pmset -g batt`) | Must |
| FR-33 | Charging status indicator must be shown when laptop is plugged in | Should |

### 2.8 Connection Management

| ID | Requirement | Priority |
|----|-------------|----------|
| FR-34 | PWA must connect to relay server via WebSocket on the same WiFi network | Must |
| FR-35 | Connection status must be visually indicated (green dot = connected, red = disconnected) | Must |
| FR-36 | PWA must auto-reconnect on connection drop with exponential backoff | Must |
| FR-37 | User must be able to configure Mac IP and port in settings | Must |

### 2.9 Settings

| ID | Requirement | Priority |
|----|-------------|----------|
| FR-38 | Settings panel must be accessible via hamburger menu | Must |
| FR-39 | Settings: Mac IP address and port | Must |
| FR-40 | Settings: TTS voice selection (from available system voices) | Should |
| FR-41 | Settings: Auto-read toggle | Must |
| FR-42 | Settings: TTS speed | Should |
| FR-43 | Settings: Theme (dark/light) | Should |
| FR-44 | Settings: Font size (small/medium/large) | Should |
| FR-45 | Settings must persist in localStorage | Must |

### 2.10 PWA Installation

| ID | Requirement | Priority |
|----|-------------|----------|
| FR-46 | App must be installable as a PWA on Android (manifest.json + service worker) | Must |
| FR-47 | App must run in standalone mode (no browser URL bar) | Must |
| FR-48 | App must cache static assets for offline shell loading | Should |
| FR-49 | App icon must appear on home screen after installation | Must |

---

## 3. Non-Functional Requirements

| ID | Requirement | Target |
|----|-------------|--------|
| NFR-01 | End-to-end latency (voice → transcription → display) | Real-time (~100-300ms per word via Web Speech API) |
| NFR-02 | Streaming response start (first token visible) | < 1 second after Claude starts |
| NFR-03 | PWA load time (cached) | < 1 second |
| NFR-04 | WebSocket reconnection | < 3 seconds |
| NFR-05 | Concurrent users supported | 1 (single user, single session) |
| NFR-06 | Authentication | Shared secret token (env var) |
| NFR-07 | Transport security | HTTPS with self-signed cert (required for Web Speech API on mobile) |
| NFR-08 | Supported browsers | Chrome Android 90+, Safari iOS 15+ (no battery) |

---

## 4. Tech Stack (Zero-Cost)

### 4.1 Chosen Stack

| Component | Technology | Cost | Reason |
|-----------|------------|------|--------|
| **PWA Framework** | SolidJS + TypeScript | Free | Reactive UI, type safety, smallest bundle, Vite for build |
| **Speech-to-Text** | Web Speech API (webkitSpeechRecognition) | Free | Real-time word-by-word transcription, ~95-98% accuracy, excellent Indian English/Hinglish support, Google-powered, zero setup, no API key |
| **Text-to-Speech** | Web Speech API (speechSynthesis) | Free | Browser-native, zero setup |
| **Permission Voice** | Web Speech API (webkitSpeechRecognition) | Free | On-device recognition for "yes/no/always" |
| **Relay Server** | Node.js + TypeScript + ws | Free | Type safety, best ecosystem for WebSocket + child process |
| **Terminal Bridge** | node-pty | Free | Full PTY emulation, preserves Claude Code's interactive behavior |
| **Terminal Emulator** | xterm.js + @xterm/addon-fit + @xterm/addon-web-links | Free | Full terminal rendering on PWA (ANSI, colors, cursor) |
| **HTTPS** | mkcert (local CA) | Free | Self-signed certs trusted by phone on same network |
| **PWA Assets** | manifest.json + service worker | Free | Browser-native PWA support |

**Key insight**: The PWA is a full remote terminal. xterm.js renders raw PTY output (ANSI, colors, interactive prompts) exactly as it would appear on the local machine. Speech processing happens in the browser via Web Speech API — the server does zero audio processing. This eliminates whisper.cpp, ffmpeg, Python, model downloads, ANSI stripping, markdown rendering, and chat-bubble logic.

### 4.2 How STT Works

```
Phone: User holds mic and speaks
         │
         ▼
  Web Speech API (in Chrome)
         │
         ├── "Show"                    → display live
         ├── "Show me"                 → display live
         ├── "Show me the"             → display live
         ├── "Show me the git"         → display live
         ├── "Show me the git status"  → display live
         │
  User releases mic
         │
         ▼
  Send "Show me the git status" → WebSocket → Server → PTY stdin
  (text string, ~100 bytes — no audio data leaves the phone)
```

### 4.3 Dependencies Summary

**Relay Server (Node.js + TypeScript)** — core dependencies
```json
{
  "dependencies": {
    "ws": "^8.x",
    "node-pty": "^1.x",
    "strip-ansi": "^7.x",
    "express": "^4.x"
  },
  "devDependencies": {
    "typescript": "^5.x",
    "@types/ws": "^8.x",
    "@types/express": "^4.x",
    "tsx": "^4.x"
  }
}
```

**PWA (SolidJS + TypeScript + Vite)**
- `xterm` + `@xterm/addon-fit` + `@xterm/addon-web-links` — terminal emulator
- `solid-js` — reactive UI framework
- Built with Vite (TypeScript compilation + bundling)
- Everything else is browser-native APIs (Web Speech, Service Worker)

**System dependencies**: Node.js + mkcert + TypeScript. That's it.

### 4.4 Stack Rejected (and Why)

| Option | Rejected Because |
|--------|-----------------|
| React/Vue/Svelte | Heavier alternatives; SolidJS chosen for fine-grained reactivity and smallest bundle |
| whisper.cpp | Not real-time, lower Indian English accuracy (~85-90%), requires server-side audio processing, ffmpeg, model downloads |
| faster-whisper | Requires Python runtime, CPU-only on Mac, higher memory, batch processing only |
| OpenAI Whisper API | Costs money ($0.006/min), not real-time |
| ElevenLabs / OpenAI TTS | Costs money, requires API key |
| Capacitor/TWA | Not needed for v1; PWA is sufficient |

---

## 5. System Architecture

### 5.1 Component Diagram

```
┌──────────────────────────────────────────────────────┐
│                  Mac (Host Machine)                    │
│                                                       │
│  ┌────────────────────────────────────────────────┐   │
│  │            Relay Server (Node.js)               │   │
│  │                                                 │   │
│  │  ┌─────────────────┐    ┌───────────────────┐  │   │
│  │  │  WebSocket       │    │  PTY Manager      │  │   │
│  │  │  Server (ws)     │    │  (node-pty)       │  │   │
│  │  │                  │    │                   │  │   │
│  │  │  - Auth          │    │  - Spawn Claude   │  │   │
│  │  │  - Receive text  │───►│  - Write stdin    │  │   │
│  │  │  - Stream output │◄───│  - Read stdout    │  │   │
│  │  │                  │    │  - Detect perms   │  │   │
│  │  │                  │    │  - Strip ANSI     │  │   │
│  │  └────────┬─────────┘    └────────┬──────────┘  │   │
│  │           │                       │              │   │
│  └───────────┼───────────────────────┼──────────────┘   │
│              │                       ▼                  │
│              │              ┌──────────────┐            │
│              │              │  Claude Code  │            │
│              │              │  (Terminal)   │            │
│              │              └──────────────┘            │
└──────────────┼──────────────────────────────────────────┘
               │ WSS (same WiFi)
               ▼
┌───────────────────────────┐
│   Phone (PWA)             │
│                           │
│  ┌─────────────────────┐  │
│  │ Web Speech API      │  │  ← STT: real-time, word-by-word
│  │ (speechRecognition) │  │  ← ~95-98% accuracy
│  │                     │  │  ← Excellent Indian English
│  │ Sends TEXT to server│  │  ← No audio leaves the phone
│  └─────────────────────┘  │
│                           │
│  ┌─────────────────────┐  │
│  │ Web Speech API      │  │  ← TTS: reads Claude responses
│  │ (speechSynthesis)   │  │
│  └─────────────────────┘  │
│                           │
│  ┌─────────────────────┐  │
│  │ xterm.js            │  │  ← Full terminal emulator
│  │ (renders raw PTY)   │  │  ← ANSI colors, cursor, prompts
│  └─────────────────────┘  │
│                           │
│  - TerminalToolbar        │  ← Mobile helper keys
│  - Server battery display │
└───────────────────────────┘
```

**Note**: The server does no audio or text processing. It pipes raw PTY bytes to the PWA and forwards keyboard input from the PWA to the PTY. All rendering happens in xterm.js, all speech processing in the browser.

### 5.2 Data Flow

```
[Voice Input]
Phone: Hold mic → Web Speech API starts recognition
Phone: Interim results displayed in real-time (word-by-word, ~100-300ms per word)
Phone: Release → Final transcription ready instantly
Phone: Send final TEXT string via WebSocket (~100 bytes, no audio data)
Server: Receive text → Write to PTY stdin (with newline)

[Keyboard Input]
Phone: User taps terminal → xterm.js focuses hidden textarea → mobile keyboard opens
Phone: User types → xterm.js onData fires → sends key data via WebSocket
Server: Receive key → Write raw bytes to PTY stdin

[Response Flow]
PTY: Claude Code writes to stdout (raw bytes with ANSI escape codes)
Server: Forward raw PTY data to Phone via WebSocket as { type: "pty", data: string }
Phone: xterm.js terminal.write(data) → renders colors, cursor, interactive prompts natively

[Permission Flow]
Phone: User sees permission prompt rendered in terminal (same as on desktop)
Phone: User types "y", "n", or "a" via keyboard or toolbar keys
Server: Key forwarded to PTY stdin
```

### 5.3 WebSocket Message Protocol

```json
// Phone → Server: Text prompt (from voice, written to PTY with newline)
{
  "type": "text",
  "data": "show me git status"
}

// Phone → Server: Raw key input (from xterm.js keyboard)
{
  "type": "key",
  "data": "a"          // single char, escape sequence, or control char
}

// Phone → Server: Terminal resize
{
  "type": "resize",
  "cols": 80,
  "rows": 24
}

// Phone → Server: Stop command
{
  "type": "stop"
}

// Server → Phone: Raw PTY output (rendered by xterm.js)
{
  "type": "pty",
  "data": "[32mOn branch main[0m\r\n..."
}

// Server → Phone: Battery status (every 10 seconds)
{
  "type": "battery",
  "level": 87,
  "charging": false
}
```

---

## 6. UI/UX Design

### 6.1 Design Principles

| Principle | Implementation |
|-----------|---------------|
| **Terminal-first, voice-assisted** | Full terminal experience with voice as a convenience for longer prompts |
| **One-thumb operation** | Mic button centered in toolbar, helper keys in thumb reach |
| **Minimal taps** | Type directly in terminal; hold mic to voice a prompt |
| **Always visible status** | Connection dot, battery percentage, recording duration always visible |
| **Forgiveness** | Slide-to-cancel recording, stop button during streaming, replay button |
| **Progressive disclosure** | Text input hidden behind a button, settings in a slide-out panel |
| **Immediate feedback** | Real-time terminal output as Claude types, waveform while recording |
| **Accessibility** | High contrast, large touch targets (48px min), readable font sizes |

### 6.2 Color Palette (Dark Theme - Default)

```
Background:         #0F1117    (deep dark)
Card / Surface:     #1A1D27    (slightly lifted)
Card hover:         #242733    (interactive feedback)
Primary accent:     #6C5CE7    (purple — mic button, highlights, links)
Primary hover:      #5A4BD1    (purple darkened)
Recording red:      #FF4757    (pulsing dot, recording state)
Connected green:    #2ED573    (connection indicator)
Warning amber:      #FFA502    (permission prompts)
Error red:          #FF6B81    (disconnected, errors)
Text primary:       #F1F2F6    (main text)
Text secondary:     #747D8C    (timestamps, labels)
Text dim:           #57606F    (placeholders)
Border:             #2F3542    (card borders, dividers)
```

### 6.3 Color Palette (Light Theme - Optional)

```
Background:         #F8F9FA
Card / Surface:     #FFFFFF
Primary accent:     #6C5CE7
Text primary:       #1A1D27
Text secondary:     #57606F
Border:             #DFE4EA
```

### 6.4 Typography

```
Font family:       'Inter', system-ui, -apple-system, sans-serif
                   (load from Google Fonts or bundle)

Status bar:        12px semi-bold
Timestamps:        11px regular, text-secondary
Chat text:         15px regular
Code blocks:       13px 'JetBrains Mono', monospace
Buttons:           14px semi-bold
Section headers:   13px semi-bold uppercase, text-secondary
```

### 6.5 Screen Layouts

#### Screen 1: Splash / Connect Screen

```
┌──────────────────────────────────┐
│                                  │
│                                  │
│                                  │
│         ┌──────────────┐         │
│         │   🎙  LOGO   │         │
│         │              │         │
│         │   Walkie     │         │
│         │   Talkie     │         │
│         └──────────────┘         │
│                                  │
│  ┌────────────────────────────┐  │
│  │ Mac IP   [192.168.1.__  ] │  │
│  │ Port     [3001          ] │  │
│  └────────────────────────────┘  │
│                                  │
│  ┌────────────────────────────┐  │
│  │         Connect            │  │
│  └────────────────────────────┘  │
│                                  │
│    Connecting...  (spinner)      │
│                                  │
└──────────────────────────────────┘
```

#### Screen 2: Main Terminal

```
┌──────────────────────────────────┐
│ ☰ Tether   ● 🔋 87%  │  ← Status bar (fixed top)
├──────────────────────────────────┤
│                                  │
│  $ claude                        │  ← xterm.js terminal (full PTY)
│                                  │
│  ╭────────────────────────────╮  │
│  │ I'll help you with that.   │  │  ← Claude Code output rendered
│  │                            │  │     with full ANSI colors
│  │ ```js                      │  │
│  │ app.get('/health', ...)    │  │
│  │ ```                        │  │
│  │                            │  │
│  │ Allow Edit src/app.ts?     │  │  ← Permission prompt in terminal
│  │ [y/n/a] █                  │  │  ← User types response directly
│  ╰────────────────────────────╯  │
│                                  │
├──────────────────────────────────┤
│ [Enter][Tab][Esc][↑][↓][←][→]   │  ← TerminalToolbar (helper keys)
│ [Ctrl+C]                         │
│                                  │
│             ┌─────────┐          │
│             │   🎙    │          │  ← Hold to voice a prompt
│             │  HOLD   │          │
│             └─────────┘          │
└──────────────────────────────────┘
```

#### Screen 6: Settings Panel (Slide-in from Left)

```
┌──────────────────────────────────┐
│ ←  Settings                      │
├──────────────────────────────────┤
│                                  │
│  CONNECTION                      │
│  ┌────────────────────────────┐  │
│  │ Mac IP    [192.168.1.42 ] │  │
│  │ Port      [3001         ] │  │
│  │ Status    ● Connected      │  │
│  └────────────────────────────┘  │
│                                  │
│  VOICE                           │
│  ┌────────────────────────────┐  │
│  │ Auto-read     ●━━━━━━ On  │  │  ← Toggle
│  │ TTS Voice     [System ▾]  │  │  ← Dropdown
│  │ Speed         [1.0x   ▾]  │  │  ← 0.5x to 2.0x
│  └────────────────────────────┘  │
│                                  │
│  DISPLAY                         │
│  ┌────────────────────────────┐  │
│  │ Theme         [Dark   ▾]  │  │
│  │ Font size     [Medium ▾]  │  │
│  └────────────────────────────┘  │
│                                  │
│  ABOUT                           │
│  ┌────────────────────────────┐  │
│  │ Version       1.0.0        │  │
│  │ Claude Code   Connected    │  │
│  └────────────────────────────┘  │
│                                  │
└──────────────────────────────────┘
```

#### Screen 7: Terminal with Mobile Keyboard Open

```
┌──────────────────────────────────┐
│ ☰ Tether   ● 🔋 87%  │
├──────────────────────────────────┤
│                                  │
│  $ show me git status█           │  ← xterm.js (resized via
│                                  │     visualViewport API)
│                                  │
├──────────────────────────────────┤
│ [Enter][Tab][Esc][↑][↓][←][→]   │  ← TerminalToolbar stays visible
│ [Ctrl+C]          🎙             │
├──────────────────────────────────┤
│                                  │
│  (mobile keyboard open)          │
│                                  │
└──────────────────────────────────┘
```

### 6.6 UI Component Specifications

#### Terminal (xterm.js)
- Fills available space between StatusBar and TerminalToolbar (flex-1)
- Font: JetBrains Mono, 14px, line-height 1.4
- Dark theme: background #0B1120, foreground #F1F5F9, cursor #60A5FA
- Light theme: background #FAFBFD, foreground #0F172A, cursor #1D4ED8
- Scrollback: 5000 lines
- Tap to focus (opens mobile keyboard)
- Resizes dynamically via FitAddon + ResizeObserver

#### TerminalToolbar
- Horizontal scrollable row of helper key buttons
- Keys: Enter, Tab, Esc, ↑, ↓, ←, →, Ctrl+C
- Button style: bg-muted, border-edge, font-mono, text-xs, rounded-md
- Uses onPointerDown + preventDefault to avoid stealing terminal focus
- Mic button centered below the key row

#### Mic Button
- Size: 64px diameter circle
- Color: Primary, white mic icon
- Active (recording): Error red, pulsing scale animation (1.0 → 1.1 → 1.0, 1s loop)
- Position: Centered within TerminalToolbar

#### Status Bar
- Height: 56px
- Background: surface, border-b edge
- Content: hamburger + title (left), theme toggle + connection dot + battery (right)
- Connection dot: 10px circle (green = connected, red = disconnected)
- Battery: server laptop battery percentage + charging indicator

### 6.7 Animations & Transitions

| Element | Animation | Duration |
|---------|-----------|----------|
| Recording dot | Opacity pulse 1.0 → 0.3 | 1s infinite |
| Mic button (recording) | Scale pulse 1.0 → 1.1 | 1s infinite |
| Slide to cancel | translateX on drag | Follows finger |
| Permission card appear | slideUp + fadeIn | 300ms ease-out |
| Chat bubble appear | fadeIn + slideUp 8px | 200ms ease-out |
| Settings panel | slideIn from left | 250ms ease-out |
| Streaming cursor | Opacity blink 1 → 0 | 800ms infinite |
| Connection status change | Color transition | 300ms |

### 6.8 Responsive Breakpoints

| Width | Adjustments |
|-------|-------------|
| < 360px | Smaller font (14px body), compact padding |
| 360-414px | Default (as designed above) |
| 414px+ | Slightly wider bubbles, comfortable spacing |
| Landscape | Not supported — lock to portrait in manifest |

---

## 7. Project File Structure

```
tether/
├── server/
│   ├── package.json
│   ├── tsconfig.json
│   ├── src/
│   │   ├── index.ts          # Entry point: Express + WebSocket server
│   │   ├── terminal.ts       # PTY manager: spawn Claude, pipe I/O
│   │   ├── permissions.ts    # Detect permission patterns in output
│   │   ├── ansi.ts           # ANSI escape code stripping
│   │   └── auth.ts           # Shared secret validation
│
├── pwa/
│   ├── index.html            # Single page app
│   ├── tsconfig.json
│   ├── vite.config.ts
│   ├── package.json
│   ├── src/
│   │   ├── app.ts            # Main app logic
│   │   ├── stt.ts            # Web Speech API wrapper (real-time transcription)
│   │   ├── tts.ts            # speechSynthesis wrapper
│   │   ├── waveform.ts       # Audio waveform visualization (Web Audio API)
│   │   ├── permissions.ts    # Voice recognition for yes/no/always
│   │   ├── websocket.ts      # WebSocket client + reconnect
│   │   ├── renderer.ts       # Markdown + syntax highlighting
│   │   ├── settings.ts       # localStorage settings manager
│   │   ├── battery.ts        # Battery API wrapper
│   │   └── style.css         # Custom styles
│   ├── public/
│   │   ├── manifest.json     # PWA manifest
│   │   ├── sw.js             # Service worker
│   │   └── icons/
│   │       ├── icon-192.png
│   │       └── icon-512.png
│
├── scripts/
│   ├── setup.sh              # Install node deps + mkcert
│   └── generate-cert.sh      # mkcert for HTTPS
│
└── README.md
```

---

## 8. Setup & Prerequisites

### 8.1 Mac Prerequisites

```bash
# 1. Node.js (v18+)
brew install node

# 2. mkcert (for local HTTPS — required for Web Speech API on mobile)
brew install mkcert
mkcert -install
mkcert localhost 192.168.1.X   # Replace with Mac's local IP

# 3. Claude Code CLI must be installed and authenticated
claude --version   # Verify
```

**That's it.** Three system dependencies. No Python, no whisper.cpp, no ffmpeg, no model downloads, no API keys.

### 8.2 Phone Prerequisites

- Android with Chrome 90+ (recommended)
- On same WiFi network as Mac
- Trust the self-signed certificate (one-time step)

---

## 9. Security Considerations

| Concern | Mitigation |
|---------|------------|
| Unauthorized access | Shared secret token in WebSocket handshake (configured via env var) |
| Network sniffing | HTTPS via mkcert self-signed certificate |
| Claude Code permissions | Users must explicitly approve via permission cards; "Always" is opt-in |
| Audio data | Web Speech API sends audio to Google servers for transcription — acceptable for coding commands (not sensitive data) |
| Transcribed prompts | Stay on local network, never leave Mac |
| PTY access | Single session only; relay server binds to configurable interface |

---

## 10. Implementation Phases

### Phase 1 — Core (MVP) ✓
- [x] Relay server with WebSocket + node-pty + HTTPS (mkcert)
- [x] xterm.js terminal emulator on PWA (renders raw PTY output)
- [x] Keyboard input forwarded from xterm.js → WebSocket → PTY
- [x] Web Speech API integration (STT — hold mic to voice prompts)
- [x] Server forwards voice text to PTY stdin
- [x] Raw PTY stdout streamed to PWA via WebSocket
- [x] Hold-to-talk mic button
- [x] Terminal resize support (FitAddon + ResizeObserver)

### Phase 2 — Polish & Mobile UX
- [x] TerminalToolbar with helper keys (Enter, Tab, Esc, arrows, Ctrl+C)
- [x] Mobile keyboard handling (visualViewport API for proper sizing)
- [x] Auto-reconnect with exponential backoff
- [x] Connection status indicator
- [ ] Slide-to-cancel recording
- [ ] Waveform visualization during recording
- [ ] Recording duration timer

### Phase 3 — Settings & Enhancements
- [x] Theme toggle (dark/light) with terminal theme switching
- [x] Server battery percentage + charging indicator
- [ ] Settings panel UI (hamburger → slide-in panel)
- [ ] TTS: auto-read toggle, voice selection, speed config
- [ ] Font size setting applied to terminal

### Phase 4 — Hardening
- [ ] Shared secret authentication on WebSocket upgrade
- [ ] Error handling and user-friendly error messages
- [ ] Service worker caching for offline shell
- [ ] PWA icons and install prompt
- [ ] Session persistence (reconnect to existing PTY session)

---

## 11. Known Limitations

| Limitation | Impact | Workaround |
|------------|--------|------------|
| Web Speech API requires internet | Low — WiFi is already required to connect to Mac | Use text input as fallback if WiFi has no internet |
| Web Speech API sends audio to Google | Low — sending coding commands, not sensitive data | Use text input for anything sensitive |
| iOS does not support Battery API | Cosmetic only | Gracefully hide battery display |
| iOS Safari has limited speechRecognition | Voice input may not work on iOS | Use text input; tap-only for permissions |
| Must be on same WiFi network | By design for v1 | Future: Tailscale/WireGuard tunnel |
| Single user only | By design for v1 | No multi-user auth needed |
| Web Speech API requires HTTPS on mobile | One-time setup | mkcert provides trusted local HTTPS |
| xterm.js on mobile has limited screen real estate | Usable with helper toolbar | TerminalToolbar provides arrow keys, Ctrl+C, Tab, etc. |

---

## 12. Future Enhancements (Post v1)

- **Remote access via Tailscale** — use from outside home network
- **Conversation history** — persist across sessions with SQLite
- **Multiple terminals** — manage more than one Claude Code session
- **File preview** — show files Claude is editing inline
- **Diff viewer** — show code changes visually
- **Upgrade TTS** — swap to Piper (local, natural voice) when quality matters
- **Play Store** — wrap with Bubblewrap TWA for Google Play listing
- **Wear OS companion** — quick voice input from smartwatch

---

*This document serves as the complete specification for building Tether. Each section can be used as a prompt context when building with Claude Code.*
