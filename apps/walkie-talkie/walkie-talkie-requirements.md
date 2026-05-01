# Walkie Talkie - Requirements Document

**Project**: Walkie Talkie
**Type**: PWA (Progressive Web App) + Relay Server
**Version**: 1.0
**Date**: 2025-04-25
**Author**: Lordson Vimal

---

## 1. Project Overview

Walkie Talkie is a voice-driven mobile PWA that acts as a remote control for Claude Code running on a Mac terminal. Users can send voice commands from their phone, have them transcribed and executed by Claude Code, and receive both text and audio responses.

### 1.1 Problem Statement

Claude Code is a powerful CLI tool, but using it requires being physically at the terminal. There is no way to interact with it remotely, hands-free, or via voice.

### 1.2 Solution

A two-part system:
1. **PWA** on the user's phone for voice input/output and display
2. **Relay Server** on the Mac that bridges the phone to Claude Code's terminal via a pseudo-terminal

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
| FR-11 | ANSI escape codes must be stripped before sending to PWA | Must |
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

### 2.5 Chat Display

| ID | Requirement | Priority |
|----|-------------|----------|
| FR-23 | User messages must be displayed as chat bubbles with the transcribed text | Must |
| FR-24 | Claude responses must be displayed as chat bubbles with markdown rendering | Must |
| FR-25 | Code blocks in responses must be syntax-highlighted | Should |
| FR-26 | Timestamps must be shown on all messages | Must |
| FR-27 | Chat must auto-scroll to the latest message | Must |
| FR-28 | Conversation history must persist within the session | Must |

### 2.6 Text Input Fallback

| ID | Requirement | Priority |
|----|-------------|----------|
| FR-29 | User must be able to type a text prompt instead of using voice | Must |
| FR-30 | Text input must be accessible via a keyboard icon/button | Must |

### 2.7 Device Information

| ID | Requirement | Priority |
|----|-------------|----------|
| FR-31 | Battery percentage must be displayed in the status bar (Android only, via Battery API) | Should |
| FR-32 | Battery display must gracefully hide on unsupported platforms (iOS) | Must |
| FR-33 | Charging status indicator must be shown when device is plugged in | Should |

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
| **PWA Framework** | Vanilla HTML + TypeScript (no framework) | Free | Type safety, smallest bundle, Vite for build |
| **Speech-to-Text** | Web Speech API (webkitSpeechRecognition) | Free | Real-time word-by-word transcription, ~95-98% accuracy, excellent Indian English/Hinglish support, Google-powered, zero setup, no API key |
| **Text-to-Speech** | Web Speech API (speechSynthesis) | Free | Browser-native, zero setup |
| **Permission Voice** | Web Speech API (webkitSpeechRecognition) | Free | On-device recognition for "yes/no/always" |
| **Relay Server** | Node.js + TypeScript + ws | Free | Type safety, best ecosystem for WebSocket + child process |
| **Terminal Bridge** | node-pty | Free | Full PTY emulation, preserves Claude Code's interactive behavior |
| **ANSI Stripping** | strip-ansi (npm) | Free | Clean output for phone display |
| **Markdown Render** | marked (npm) | Free | Lightweight markdown → HTML |
| **Syntax Highlight** | highlight.js | Free | Code block highlighting in responses |
| **HTTPS** | mkcert (local CA) | Free | Self-signed certs trusted by phone on same network |
| **PWA Assets** | manifest.json + service worker | Free | Browser-native PWA support |

**Key insight**: All speech processing happens in the browser via Web Speech API. The relay server does zero audio processing — it just receives text strings and pipes them to Claude Code. This eliminates whisper.cpp, ffmpeg, Python, model downloads, and all server-side transcription code.

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

**PWA (TypeScript + Vite)**
- `marked` — markdown rendering
- `highlight.js` — syntax highlighting
- Built with Vite (TypeScript compilation + bundling)
- Everything else is browser-native APIs (Web Speech, Battery, Service Worker)

**System dependencies**: Node.js + mkcert + TypeScript. That's it.

### 4.4 Stack Rejected (and Why)

| Option | Rejected Because |
|--------|-----------------|
| React/Vue/Svelte | Overkill for single-page app; adds build step, larger bundle |
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
│  - Markdown render        │
│  - Battery API            │
└───────────────────────────┘
```

**Note**: The server has no audio processing at all. It receives text strings from the phone and pipes them to Claude Code. All speech processing happens in the browser.

### 5.2 Data Flow

```
[Voice Input]
Phone: Hold mic → Web Speech API starts recognition
Phone: Interim results displayed in real-time (word-by-word, ~100-300ms per word)
Phone: Release → Final transcription ready instantly
Phone: Send final TEXT string via WebSocket (~100 bytes, no audio data)
Server: Receive text → Forward directly to PTY stdin

[Response Flow]
PTY: Claude Code writes to stdout
Server: Read stdout → Strip ANSI codes
Server: Check for permission pattern → If yes, send as permission message
Server: Otherwise, stream text to Phone via WebSocket
Phone: Render markdown, auto-scroll
Phone: On completion signal → speechSynthesis.speak()

[Permission Flow]
Server: Detect "Allow ...? [y/n/a]" in stdout
Server: Send structured permission message to Phone
Phone: Show permission card with Yes/No/Always buttons
Phone: (Optional) Activate speechRecognition for voice input
Phone: User taps or says response
Phone: Send permission_response via WebSocket
Server: Write corresponding key (y/n/a) to PTY stdin
```

### 5.3 WebSocket Message Protocol

```json
// Phone → Server: Text prompt (from Web Speech API or typed)
{
  "type": "text",
  "data": "show me git status"
}

// Phone → Server: Permission response
{
  "type": "permission_response",
  "value": "yes" | "no" | "always"
}

// Phone → Server: Stop command
{
  "type": "stop"
}

// Server → Phone: Claude output (streaming)
{
  "type": "output",
  "text": "On branch main...",
  "streaming": true
}

// Server → Phone: Claude response complete
{
  "type": "output_complete",
  "fullText": "On branch main.\nChanges not staged:..."
}

// Server → Phone: Permission request
{
  "type": "permission",
  "action": "Edit to src/server.ts",
  "options": ["yes", "no", "always"]
}

// Server → Phone: Connection status
{
  "type": "status",
  "connected": true,
  "claudeReady": true
}
```

---

## 6. UI/UX Design

### 6.1 Design Principles

| Principle | Implementation |
|-----------|---------------|
| **Voice-first, text-second** | Primary action is hold-to-talk; text input is a secondary fallback |
| **One-thumb operation** | Mic button at bottom-right, reachable by thumb in portrait mode |
| **Minimal taps** | Hold to record, release to send — single gesture for the core workflow |
| **Always visible status** | Connection dot, battery percentage, recording duration always visible |
| **Forgiveness** | Slide-to-cancel recording, stop button during streaming, replay button |
| **Progressive disclosure** | Text input hidden behind a button, settings in a slide-out panel |
| **Immediate feedback** | Waveform while recording, streaming text as Claude types, auto-TTS on complete |
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

#### Screen 2: Main Chat — Idle State

```
┌──────────────────────────────────┐
│ ☰  Walkie Talkie    🔋 87% │  ← Status bar (fixed top)
│  ● Connected                     │  ← Green dot
├──────────────────────────────────┤
│                                  │
│  ┌────────────────────────────┐  │  ← Scrollable chat area
│  │ 🎙  You  ·  10:32 AM       │  │
│  │ "Show me the git status"   │  │
│  └────────────────────────────┘  │
│                                  │
│  ┌────────────────────────────┐  │
│  │ 🤖 Claude  ·  10:32 AM     │  │
│  │                            │  │
│  │ On branch main.            │  │
│  │ Changes not staged:        │  │
│  │   modified: src/app.ts     │  │
│  │   modified: package.json   │  │
│  │                            │  │
│  │ 🔊 ▶ Play                  │  │  ← Replay TTS button
│  └────────────────────────────┘  │
│                                  │
│  (more messages scroll here)     │
│                                  │
│                                  │
├──────────────────────────────────┤  ← Fixed bottom bar
│                                  │
│  ┌─────────┐       ┌─────────┐  │
│  │ ⌨ Type  │       │   🎙    │  │  ← Text fallback + Mic
│  └─────────┘       │  HOLD   │  │
│                    └─────────┘  │
│                                  │
└──────────────────────────────────┘
```

#### Screen 3: Main Chat — Recording State

```
┌──────────────────────────────────┐
│ ☰  Walkie Talkie    🔋 87% │
│  ● Connected                     │
├──────────────────────────────────┤
│                                  │
│  (previous messages dimmed 50%)  │
│                                  │
│                                  │
│                                  │
│                                  │
│                                  │
│                                  │
│                                  │
├──────────────────────────────────┤
│ ┌──────────────────────────────┐ │
│ │                              │ │
│ │    ◉ Recording  0:03         │ │  ← Pulsing red dot + timer
│ │                              │ │
│ │    ∿∿∿∿∿∿∿∿∿∿∿∿∿∿∿∿∿∿∿∿   │ │  ← Live waveform (canvas)
│ │                              │ │
│ │  ╳ Slide left to cancel      │ │  ← Cancel gesture hint
│ │                              │ │
│ │    ▲ Release to send         │ │  ← Send hint
│ │                              │ │
│ └──────────────────────────────┘ │
└──────────────────────────────────┘
```

#### Screen 4: Main Chat — Streaming Response

```
┌──────────────────────────────────┐
│ ☰  Walkie Talkie    🔋 87% │
│  ● Connected                     │
├──────────────────────────────────┤
│                                  │
│  ┌────────────────────────────┐  │
│  │ 🎙  You  ·  10:35 AM       │  │
│  │ "Add a health check        │  │
│  │  endpoint to the server"   │  │
│  └────────────────────────────┘  │
│                                  │
│  ┌────────────────────────────┐  │
│  │ 🤖 Claude  ·  10:35 AM     │  │
│  │                            │  │
│  │ I'll add a health check   │  │
│  │ endpoint to the server.   │  │
│  │                            │  │
│  │ ```js                      │  │  ← Syntax highlighted
│  │ app.get('/health',         │  │
│  │   (req, res) => {          │  │
│  │ █                          │  │  ← Blinking cursor
│  └────────────────────────────┘  │
│                                  │
├──────────────────────────────────┤
│  ┌──────────────────────────┐    │
│  │      ⏹  Stop Claude       │   │  ← Sends Ctrl+C
│  └──────────────────────────┘    │
└──────────────────────────────────┘
```

#### Screen 5: Permission Prompt

```
┌──────────────────────────────────┐
│ ☰  Walkie Talkie    🔋 87% │
│  ● Connected                     │
├──────────────────────────────────┤
│                                  │
│  (previous messages)             │
│                                  │
│  ┌────────────────────────────┐  │
│  │ ⚠  Permission Request      │  │  ← Amber accent card
│  │                            │  │
│  │ Claude wants to:           │  │
│  │                            │  │
│  │ Edit src/server.ts         │  │  ← Action description
│  │                            │  │
│  │ ┌──────┐ ┌────┐ ┌──────┐  │  │
│  │ │ Yes  │ │ No │ │Always│  │  │  ← Tap buttons
│  │ └──────┘ └────┘ └──────┘  │  │
│  │                            │  │
│  │    🎙 or say "yes" / "no"  │  │  ← Voice hint
│  └────────────────────────────┘  │
│                                  │
├──────────────────────────────────┤
│  (mic button disabled during     │
│   permission prompt)             │
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

#### Screen 7: Text Input Mode (Expanded)

```
┌──────────────────────────────────┐
│ ☰  Walkie Talkie    🔋 87% │
│  ● Connected                     │
├──────────────────────────────────┤
│                                  │
│  (chat messages above)           │
│                                  │
│                                  │
├──────────────────────────────────┤
│  ┌────────────────────────────┐  │
│  │ Type your prompt...        │  │  ← Text input field
│  │                            │  │
│  └────────────────────────────┘  │
│                                  │
│  ┌──────┐              ┌─────┐  │
│  │  🎙  │              │ ➤  │  │  ← Back to mic / Send
│  └──────┘              └─────┘  │
│                                  │
│  (keyboard open)                 │
└──────────────────────────────────┘
```

### 6.6 UI Component Specifications

#### Mic Button
- Size: 64px diameter circle
- Color: Primary (#6C5CE7), white mic icon
- Active (recording): Recording red (#FF4757), pulsing scale animation (1.0 → 1.1 → 1.0, 1s loop)
- Touch target: 80px (16px padding around visible button)
- Position: Fixed bottom-right, 24px from edges

#### Chat Bubbles
- User bubble: Background #6C5CE7 (primary), text white, right-aligned, max-width 85%
- Claude bubble: Background #1A1D27 (card), text #F1F2F6, left-aligned, max-width 90%
- Border radius: 16px (2px on the corner nearest the sender)
- Padding: 12px 16px
- Margin between bubbles: 12px

#### Permission Card
- Background: #1A1D27 with left border 3px solid #FFA502 (amber)
- Button sizes: 44px height, equal width distributed
- Yes: #2ED573 background, dark text
- No: #FF4757 background, white text
- Always: #6C5CE7 background, white text

#### Status Bar
- Height: 44px
- Background: #0F1117
- Position: Fixed top
- Content: hamburger (left), title (center), battery (right)
- Connection dot: 8px circle, positioned after title

#### Waveform Visualization
- Canvas element, 100% width, 80px height
- Bar style: vertical bars, 3px wide, 2px gap
- Color: #6C5CE7 (idle), #FF4757 (recording)
- Animation: requestAnimationFrame with AnalyserNode from Web Audio API

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
walkie-talkie/
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

### Phase 1 — Core (MVP)
- [ ] Relay server with WebSocket + node-pty + HTTPS (mkcert)
- [ ] Web Speech API integration on PWA (STT — real-time, word-by-word)
- [ ] Send transcribed text to server via WebSocket
- [ ] Server forwards text to Claude Code PTY stdin
- [ ] ANSI stripping and stdout streaming back to PWA
- [ ] PWA with hold-to-talk mic button
- [ ] Chat display with markdown rendering
- [ ] Web Speech API TTS on response complete

### Phase 2 — Permissions & Polish
- [ ] Permission prompt detection and structured messaging
- [ ] Permission card UI with tap buttons
- [ ] Voice-based permission responses ("yes" / "no" / "always")
- [ ] Stop button (Ctrl+C to PTY)
- [ ] Auto-reconnect with exponential backoff
- [ ] Slide-to-cancel recording
- [ ] Text input fallback mode

### Phase 3 — UX Enhancements
- [ ] Waveform visualization during recording
- [ ] Battery percentage display (Android)
- [ ] Settings panel (TTS voice, speed, theme, font size)
- [ ] Syntax highlighting in code blocks
- [ ] Light theme option

### Phase 4 — Hardening
- [ ] Shared secret authentication
- [ ] Error handling and user-friendly error messages
- [ ] Service worker caching for offline shell
- [ ] PWA install prompt handling
- [ ] Session persistence (conversation survives page reload)

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
| ANSI stripping may lose some formatting | Cosmetic only | Accept tradeoff; phone is for readability |

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

*This document serves as the complete specification for building Walkie Talkie. Each section can be used as a prompt context when building with Claude Code.*
