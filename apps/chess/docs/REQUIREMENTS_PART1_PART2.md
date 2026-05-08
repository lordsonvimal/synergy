# ChessLeap — Detailed Requirements: Part 1 & Part 2

---

## Part 1 — Live Class

### Overview

A unified live class room that combines an interactive teaching board with WebRTC video and audio. Video and audio are always on — there is no board-only mode. Both coach and students join via shareable links with no login required.

---

### 1.1 Session Creation

- Coach visits `/class` and clicks "New Class"
- Server creates a session with a unique ID and returns two links:
  - **Coach link**: `/class/:sessionID?role=coach&token=<token>`
  - **Student join link**: `/class/:sessionID` (one link shared with all students)
- The token in the coach link is single-use for role claiming; the first browser to open it becomes the coach
- Any participant opening the student join link is assigned the **Student** role (up to the configured max)
- Roles are bound to a server-side session cookie; refreshing restores the role
- Session states: `waiting` → `live` → `ended`
- Idle sessions expire after 4 hours; ended sessions are archived and accessible for replay

---

### 1.2 WebRTC Infrastructure

#### Architecture

**MVP: LiveKit Cloud** — managed SFU that handles all WebRTC complexity (STUN, TURN, ICE, simulcast, adaptive bitrate). No separate server to deploy or maintain for MVP.

```
Browser ──WebRTC (video/audio)──► LiveKit Cloud
   │                                    ▲
   │ HTTP/SSE (board, chat)             │ RoomService API
   ▼                                    │
Your Go/Gin Server ─────────────────────┘
(issues JWT tokens, manages session state)
```

- Your Go server never handles WebRTC media — it only issues signed JWT access tokens per participant and calls LiveKit's RoomService API for coach controls (mute, remove)
- The browser connects directly to LiveKit Cloud for all video and audio
- Supports up to 20 participants per session (configurable in LiveKit room settings)
- STUN, TURN, ICE negotiation, and stream forwarding are all managed by LiveKit — no Coturn, no signaling WebSocket, no Pion

**Video subscription model (star topology):**
- Students subscribe to: coach video + coach audio + active speaker audio only
- Students do not subscribe to each other's video — they watch the board, not each other
- Coach subscribes to: all student audio + spotlight student video if spotlighted
- This reduces server outbound bandwidth ~10× compared to full-mesh forwarding

**Scaling path:** When class volume exceeds LiveKit Cloud's free tier (100k participant-minutes/month ≈ 1,600 class-hours), swap the `LIVEKIT_URL` env var to point at a self-hosted LiveKit Server running in Docker on the same VPS — no code changes required.

#### Reconnection
- LiveKit JS SDK handles reconnection automatically with exponential backoff
- If reconnection fails after 2 minutes, the client shows a "Connection lost" error
- Board state and chat history are restored from the Go server on rejoin (not from LiveKit)

#### Session Lifecycle
- Students who join before the coach land in the **Waiting Room** (§1.9)
- Session starts when the coach clicks "Start Class" — students in the waiting room see a countdown
- Session ends when the coach clicks "End Class" or all participants disconnect; Go server closes the LiveKit room via API

---

### 1.3 Video & Audio Controls

#### Per-Participant Controls (self)
- **Mute/unmute microphone**: toggle button with visual indicator (microphone icon, red when muted)
- **Enable/disable camera**: toggle button (camera icon, crossed out when off); replaces video tile with avatar/initials
- **Leave class**: confirmation dialog before disconnecting

#### Coach Controls (over others)
- Mute any student's microphone (student notified with a toast: "Coach muted your mic")
- Cannot remotely disable a student's camera (privacy — student controls their own video)
- Unmute request: coach can send an unmute prompt; student must accept
- Remove participant from session (with confirmation)

#### Audio Indicators
- Speaking indicator: animated border ring on the active speaker's video tile
- Volume meter displayed on each tile when audio is active

---

### 1.4 Layout Management

Four layouts, switchable at any time via a layout button in the toolbar:

#### Speaker View
- Active speaker's video tile fills 70% of the video panel
- Other participants shown as a horizontal thumbnail strip below
- Active speaker switches automatically based on audio activity (1.5s debounce to avoid rapid switching)
- Manual override: coach or participant can pin a specific person to stay in the large tile

#### Grid View
- All participants displayed in equal-sized tiles in a responsive grid
- Up to 4×5 = 20 tiles maximum
- Tiles reflow as participants join or leave

#### Board-Focused View
- Chess board and notation panel take 75% of the screen width
- Video panel collapses to a narrow vertical strip of small tiles on the right
- Clicking a tile in the strip expands it temporarily to a larger floating overlay

#### Coach Broadcast Mode
- Only the coach's video is displayed prominently (full video panel)
- Students' video streams are still transmitted but not rendered
- Students see the coach's video and the board
- Coach can switch any student into spotlight from this mode (temporarily shows that student's video in place of the coach's)

#### Pinning
- Any participant can pin any other participant's video tile
- Pinned tile stays in the primary position regardless of who is speaking
- Pin indicator (pin icon) shown on the pinned tile
- One active pin per client; pinning a new tile unpins the previous one
- Coach pins from the participants panel; students pin by clicking the tile

---

### 1.5 Teaching Board

The board panel is always present alongside the video. The coach controls the board mode via a mode selector in the board toolbar. Switching modes broadcasts the transition to all participants.

#### 1.5.1 Board Modes

##### a) Teaching Mode (default)

The coach has free, unrestricted control of the board for analysis and instruction.

- Coach can move any piece of either color freely — no turn enforcement, no rules restriction
- Moves are broadcast in real time to all participants via SSE
- **Takeback**: undo moves one at a time (button or `Ctrl+Z`); multi-step undo supported; board state broadcast after each undo
- **FEN / Position Setup**: "Set Position" button opens a modal with:
  1. **FEN input** — paste a FEN string; board updates immediately on valid FEN
  2. **Piece placement editor** — blank board with a piece palette; drag pieces onto squares; click a placed piece to remove it
  - Side to move, castling rights, and en-passant square configurable in the FEN panel
  - "Reset to starting position" button
- **Arrows & Circle Annotations**:
  - Draw colored arrows by clicking and dragging from one square to another
  - Draw circle highlights by right-clicking a square
  - Color picker: Green, Red, Yellow, Blue (default Green)
  - Annotations broadcast to all participants instantly
  - "Clear all annotations" button (keyboard shortcut `Escape`)
  - Annotations stored per move: navigating to a past move restores its saved annotations
- **Engine Evaluation**:
  - Toggle button to show/hide engine evaluation
  - When enabled: evaluation bar (vertical, ±centipawns), best-move arrow in purple, top 3 candidate moves with scores
  - Engine runs server-side (existing alpha-beta engine); default depth 15
  - Visible to coach only by default; coach can toggle "Show engine to students"
- **Move Notation Panel**:
  - SAN notation rendered as moves are played: `1. e4 e5  2. Nf3 Nc6 …`
  - Two-column format: White on left, Black on right
  - Clicking any move navigates the board to that position (view only — does not affect the live board)
  - Arrow keys `←` `→` navigate moves
  - "Live" button returns board view to the current position
  - Navigation is per-client: each participant can independently step through moves
- **Board Coordinates**: file letters (a–h) along the bottom; rank numbers (1–8) along the left; flip with board orientation
- **Student Spotlight on Board**: coach can grant any student control of the board
  - Spotlighted student can move pieces and draw annotations
  - Coach retains override: can revoke control at any time
  - Only one student spotlighted at a time

##### b) Training Game Mode

The coach pairs two participants to play a rules-enforced timed game while the rest of the class spectates.

- Coach opens the mode selector and chooses "Training Game"
- Coach selects two participants (any combination of students, or coach + one student) as Player 1 (White) and Player 2 (Black)
- Coach selects a time control: 3+0, 5+2, 10+5, 15+10
- Coach clicks "Start Game" — board transitions to training game layout (clocks visible above/below the board, chat remains open)
- All other participants become spectators: they see the board and can follow the notation but cannot interact with the board
- Full chess rules enforced; moves validated server-side; clocks run server-side with SSE tick events
- **Coach game controls** (visible only to the coach):
  - **Pause / Resume**: freezes both clocks; players see "Game paused by coach" banner
  - **Abort**: ends the game without a result; all participants notified
  - **Force Takeback**: reverts the last move; both players notified; either player can reject within 10 seconds
- When the game ends (checkmate, time, resign, draw offer accepted): a result banner is shown; the board automatically returns to Teaching Mode with the final position loaded — coach can continue analysis immediately
- Game result and move sequence are saved in the session recording

##### c) Drill Mode

The coach runs interactive exercises with the class. Seven drill types are supported — each has a different student input method and answer format, but all share the same session flow: coach configures and starts the drill, students respond independently on their own boards, the coach privately reviews submissions before anything is shown to the class, and answers are revealed only when the coach decides. Auto-validation (correct / incorrect / partial) is applied on reveal where the answer is unambiguous.

**Drill types:**

| # | Type | Student Input | Auto-validate |
|---|------|---------------|---------------|
| 1 | **Find the Move** | Click destination square (single best move) | ✓ |
| 2 | **Find the Mate** | Play moves until checkmate is delivered (Mate in N) | ✓ |
| 3 | **Arrange the Board** | Drag pieces onto a blank board to recreate the target | ✓ |
| 4 | **Name This Piece** | Type piece name or select from a list | ✓ |
| 5 | **Write the Notation** | Type SAN string for a highlighted move | ✓ |
| 6 | **Spot the Illegal Move** | Click the illegal move from a displayed list | ✓ |
| 7 | **Play the Opening** | Play the correct book moves for N moves | ✓ |

**Shared session flow:**

1. Coach opens the Drill Mode panel and selects a drill type
2. Coach completes the type-specific setup (see below)
3. Coach optionally sets a timer (30s, 60s, 90s, or none) and clicks "Start Drill"
4. Each student receives a private, independent copy of the drill on their board
5. Student interacts using the type-specific input; first complete answer locks in the submission
6. Participants panel shows live status per student: Thinking… / Submitted (answer content not shown)
7. Coach privately previews submissions (see Private Preview below)
8. Coach reveals answers selectively or all at once (see Reveal below)
9. Coach clicks "End Drill" — student boards reset, Teaching Mode resumes; unrevealed answers are discarded and never shown to the class

**Type-specific setup and student experience:**

**1. Find the Move**
- Setup: coach sets a position (FEN or piece placement); optionally adds a prompt ("Find the best move for White")
- Student: receives the position on their private board; clicks a destination square to submit; first click locks the move
- Auto-validate: server compares against the engine's top move — ✓ correct / ✗ incorrect shown on reveal tile

**2. Find the Mate**
- Setup: coach sets a forced-checkmate position; specifies "Mate in N" (1–5 moves)
- Student: plays moves on their board until checkmate is delivered or all moves are exhausted; sequence auto-submits on checkmate or on clicking "Submit"
- Auto-validate: ✓ if checkmate achieved in N or fewer; ≈ if checkmate achieved in more moves; ✗ if no checkmate found

**3. Arrange the Board**
- Setup: coach selects a target position (FEN or piece placement); target is hidden from students
- Student: receives a blank board and a piece palette; drags pieces to squares to recreate the target; clicks "Submit" when done; timer is recommended — completing within time earns a bonus indicator shown on reveal
- Auto-validate: ✓ for exact position match; reveal tile shows count of misplaced pieces otherwise

**4. Name This Piece**
- Setup: coach selects a board position and highlights one or more squares; chooses free-text entry or multiple-choice (configurable)
- Student: sees the board with the highlighted square(s); types the piece name (e.g., "Knight") or selects from the list; Enter or "Submit" locks the answer
- Auto-validate: ✓ for case-insensitive match including abbreviations (N, R, B, Q, K, P)

**5. Write the Notation (SAN)**
- Setup: coach sets a position and makes one or more moves on the board; those moves are hidden from students; prompt text shown (e.g., "Write the SAN for White's best reply")
- Student: types the SAN string in a text input (e.g., "Nf3", "O-O", "exd5+"); submits with Enter or "Submit" button
- Auto-validate: ✓ for exact SAN match against the coach's recorded answer

**6. Spot the Illegal Move**
- Setup: coach provides a list of 3–5 moves (SAN) in a given position; marks which one is illegal
- Student: sees the move list with radio buttons; selects the move they believe is illegal and submits
- Auto-validate: ✓ / ✗ — single correct answer

**7. Play the Opening**
- Setup: coach selects an opening and specifies the depth (e.g., "Play the first 5 moves of the Ruy Lopez as White"); the book line is stored server-side
- Student: plays moves on the board; each correct book move is confirmed with a green highlight; first deviation is flagged immediately on the student's board ("Deviated — try the book move")
- Auto-validate: ✓ if the student completed all N book moves correctly; reveal tile shows the deviation move if they went off-book

**Private preview (shared across all drill types):**
- Coach clicks [👁 Preview] on any submitted student in the participants panel
- The coach's board shows that student's answer privately — for multi-move types (Mate, Opening) the full played sequence is shown
- A persistent banner reads: "Private preview — [Name]'s answer (not visible to students)"; the student is never notified
- Coach cycles through all submitted answers using [← Prev] / [Next →] without leaving preview mode
- [Skip] advances to the next student without revealing the current one to the class
- Private preview does not affect student boards in any way

**Reveal options (three paths, usable in any combination):**
- **Reveal one**: while previewing, coach clicks "Show to Class" — that student's board tile appears in the class reveal grid, visible to all participants; auto-validation result (✓ / ✗ / ≈) shown on the tile
- **Reveal selected**: coach selects multiple students from the participants panel and reveals them as a group
- **Reveal All**: broadcasts all submitted answers simultaneously; the board panel switches to a full-width grid; students who did not submit are marked "No answer"

**Post-reveal discussion:**
- Revealed tiles remain visible; coach can click any tile to load that student's answer onto the main board for annotation and discussion; annotations broadcast to all
- Unrevealed answers remain hidden until the coach explicitly reveals them or uses "Reveal All"
- No ratings or points — purely pedagogical

**Ending the drill:**
- Coach clicks "End Drill" — all student boards reset and Teaching Mode resumes
- Any unrevealed submissions are discarded without being shown to the class

##### d) Student Sandbox Mode

Each student gets a private copy of the current board position to explore variations freely, while the coach monitors all boards.

- Coach clicks "Open Sandbox" — each student's board detaches from the main board and becomes independent
- Students can move pieces freely (no rules enforcement, no turns) to explore lines
- Students cannot see each other's sandbox boards
- Coach's board panel switches to a thumbnail grid showing all student boards in real time
- Coach can click any thumbnail to enlarge it (watch mode) — the student does not know they are being watched
- Coach can "Pull to Main Board": copies a student's current sandbox position to the main teaching board and switches all participants back to Teaching Mode for discussion
- When the coach clicks "Close Sandbox": all student boards reset to the main teaching board's current position; Teaching Mode resumes

---

### 1.6 Screen Sharing

#### Coach Screen Share
- Coach clicks "Share Screen" → browser native screen picker (entire screen / application window / browser tab)
- Shared screen stream replaces the board area OR appears as a large tile in the video panel (coach chooses)
- When sharing screen in board area: board moves to a minimized panel; "Return to board" button swaps back
- Coach can annotate on the shared screen using an overlay canvas (arrows and highlights)
- Screen share ends when coach clicks "Stop Sharing" or closes the browser share prompt

#### Student Screen Share
- Students can request to share their screen via a "Request Screen Share" button
- Coach receives a notification and approves or denies
- Only one student can share at a time
- Approved student's screen shown in the board area for the coach; coach can make it visible to all students
- Automatically ended if coach revokes or student leaves

---

### 1.7 Chat & Interaction

#### In-Class Chat
- Text chat panel open throughout the session
- All participants (coach + students) can send messages
- Messages show sender name, timestamp, and role badge (Coach / Student)
- Text only in v1 (no file attachments)
- Coach can clear the chat (clears for all participants)
- Coach can mute a student's chat (student sees "Your chat has been muted by the coach")

#### Raise Hand
- Students click "Raise Hand"; a hand icon appears on their video tile
- Coach sees a notification: "Student X raised their hand"
- Hands queued in order raised — coach sees the queue in the participants panel
- Coach acknowledges a hand (removes it from the queue); optionally spotlights the student
- Student can lower their own hand at any time

#### Reactions
- Emoji reaction bar: Thumbs Up, Confused, Clap, Surprised
- Reaction appears as a floating emoji animation over the sender's video tile (visible to all, fades after 3 seconds)
- Does not interrupt audio

---

### 1.8 Participants Panel

Slide-out panel showing all participants with:
- Name, role badge (Coach / Student)
- Audio status (muted / unmuted icon)
- Video status (camera on / off icon)
- Hand raised indicator
- Connection quality indicator (green / yellow / red)

**Coach actions per participant** (context menu or hover):
- Mute microphone
- Send unmute request
- Spotlight on video
- Grant board control (Teaching Mode)
- Assign as Player 1 / Player 2 (Training Game Mode)
- Remove from session

**Board-mode-specific status columns** (visible in the panel):
- Training Game Mode: player assignment (Player 1 / Player 2 / Spectator)
- Position Drill Mode: submission status (Thinking… / Submitted)
- Student Sandbox Mode: "Watch board" and "Pull board to main" actions

Participant count shown in the toolbar at all times.

---

### 1.9 Waiting Room

- Students who join before the coach, or before the coach clicks "Start Class", land in the waiting room
- Waiting room shows: class title, coach name, scheduled start time, countdown timer
- Students can check their camera and microphone (preview of own video, mic level indicator)
- Coach sees a list of students in the waiting room from the class setup page
- Coach can admit all at once or individually

---

### 1.10 Session Recording

#### What Is Recorded
- All video and audio streams (mixed to a single MP4 file server-side)
- Board move sequence with timestamps
- Annotations (arrows, circles) per move
- Board mode transitions with timestamps (e.g., "Training Game started", "Drill revealed")
- Chat messages with timestamps
- Screen shares (captured as part of the video stream)

#### Recording Controls
- Coach starts/stops recording manually (not automatic)
- "Recording" indicator visible to all participants when active (red dot + "REC" label)
- Participants notified when recording starts and stops
- Coach can pause and resume recording (creates chapters in the replay)

#### Storage & Access
- Recordings stored server-side (local filesystem or S3-compatible object storage via configurable env var)
- Retained for **30 days** by default (configurable per session)
- Accessible only to the coach and students who attended the session (enforced by session membership check)
- Coach can download the video file (MP4)
- Coach can delete a recording before the retention period expires

---

### 1.11 Session Replay

- Replay accessible at `/class/:sessionID/replay` for 30 days, then deleted
- Video player synced with board replay: as the video plays, the board advances through moves at the correct timestamps
- Chat log displayed alongside, scrolling in sync with video
- Timeline scrubber: clicking a position in the timeline jumps both video and board to that moment
- Timeline shows board mode transitions as labelled chapters (e.g., "Training Game", "Position Drill")
- Annotations (arrows, circles) per move are restored as the replay advances
- Coach can download the session as a PGN file (with annotation comments embedded)
- Coach can share the replay link with students who missed the class

---

### 1.12 Class End

- Coach clicks "End Class" → confirmation dialog
- All participants notified: "The class has ended"
- Session state set to `ended`; no new participants can join
- If recording was active, it is automatically stopped and processing begins
- Post-class summary shown to coach:
  - Duration, participant count, peak attendees
  - List of attendees
  - Link to recording (once processed)
  - Download PGN of board session

---

### 1.13 Technical Constraints & Edge Cases

| Scenario | Behaviour |
|----------|-----------|
| Student joins mid-class | Admitted immediately if coach has started; sees board at current position and live video |
| Student reconnects | Rejoins same session; board state and chat history restored from server |
| Coach disconnects | 60-second grace period; class paused automatically; students see "Coach disconnected" banner; coach reconnects and resumes |
| Browser tab hidden | Video encoding pauses (browser throttling); audio continues; reconnects automatically when tab is foregrounded |
| Poor connection (student) | SFU drops video resolution; audio preserved; connection quality indicator turns yellow/red |
| Screen share tab closed | Screen share ends automatically; board view restored |
| Max capacity reached | New join attempts shown a "Class is full" page |
| Student in Training Game disconnects | Their clock continues running; opponent notified; coach can abort the game |
| Student in Drill Mode disconnects | Their submission slot marked "No answer" |
| Student in Sandbox Mode disconnects | Sandbox board lost; on reconnect, student receives a fresh copy of the current main board position |

---

## Part 2 — Online Play

### Overview

A standalone link-based play session for two players. No login required. Intended for casual online games outside of a class context.

---

### 2.1 Session Creation

- Player visits `/play` and clicks "New Game"
- Selects time control (Blitz 3+0, Rapid 5+2, Rapid 10+5, Classical 15+10)
- Optionally selects color preference (White / Black / Random)
- Server creates a game session and returns **two distinct links**:
  - **White link**: `/play/:gameID?role=white&token=<token>`
  - **Black link**: `/play/:gameID?role=black&token=<token>`
- Creator is shown both links with a copy button for each
- Creator shares the opponent's color link directly (e.g., paste in WhatsApp/email)
- Anyone opening the game URL after both roles are claimed becomes a **Spectator**

---

### 2.2 Role Claiming

- Role (White/Black) is claimed by the first person to open that color's link
- Role is bound to a session cookie for that browser; refreshing the page restores the role
- Token in the URL is single-use for role claiming; subsequent opens of the same token URL are treated as spectator if the role is already claimed
- Each player sees the board oriented to their color

---

### 2.3 Gameplay

- Full rules enforcement (existing engine)
- Clock starts when both players have opened their links and are marked online
- Online status indicator per player (green dot = connected, grey = disconnected)
- If a player disconnects, their clock continues running; an "Opponent disconnected" banner appears
- Reconnecting within 60 seconds resumes the game from the current state

---

### 2.4 In-Game Chat

- Chat panel visible to both players and spectators
- Players can send messages at any time during the game
- Spectators can read chat but cannot send messages
- Messages prefixed with player color: `[White] Good luck!`
- Chat history persists for the session duration

---

### 2.5 Draw Offer & Resign

- **Draw offer**: player clicks "Offer Draw"; opponent sees a notification banner with Accept / Decline buttons; offer expires after 30 seconds
- Only one active draw offer allowed at a time; offering player cannot offer again until the previous offer is resolved
- **Resign**: player clicks "Resign" → confirmation dialog ("Are you sure?") → game ends immediately
- **Claim draw**: button appears automatically when threefold repetition or 50-move rule applies

---

### 2.6 Rematch

- After game ends, both players see a "Rematch" button
- Requesting rematch sends a notification to the opponent
- If both players accept within 60 seconds, a new game starts with colors swapped
- New game gets new links (same session page, no need to reshare)

---

### 2.7 Spectators

- Spectators see the board and move notation panel in real time
- Spectators can independently navigate move history (view only)
- Spectator count displayed in the session header

---

## Summary: Features at a Glance

### Part 1 — Live Class
| Feature | Included |
|---------|---------|
| Link-based, no login | ✅ |
| WebRTC video + audio (SFU, always on) | ✅ |
| Waiting room with camera/mic preview | ✅ |
| Layout modes: Speaker / Grid / Board-focused / Broadcast | ✅ |
| Pin participant video | ✅ |
| Coach mute / unmute request | ✅ |
| Teaching board — free piece movement | ✅ |
| Teaching board — FEN / position setup | ✅ |
| Teaching board — takeback (multi-step) | ✅ |
| Teaching board — colored arrows & circles | ✅ |
| Teaching board — engine evaluation panel | ✅ |
| Teaching board — move notation + navigation | ✅ |
| Teaching board — board coordinates | ✅ |
| Student spotlight on board | ✅ |
| Training Game Mode (rules-enforced game within class) | ✅ |
| Drill Mode — 7 types (Find Move, Find Mate, Arrange Board, Name Piece, Write SAN, Illegal Move, Play Opening) | ✅ |
| Student Sandbox Mode (independent boards per student) | ✅ |
| Screen sharing (coach + students with approval) | ✅ |
| Screen annotation overlay | ✅ |
| Chat (all participants) | ✅ |
| Raise hand queue | ✅ |
| Emoji reactions | ✅ |
| Participants panel with coach controls | ✅ |
| Server-side recording (video + board + chat) | ✅ |
| Synced replay page (video + board + chat) | ✅ |
| Board mode chapters in replay timeline | ✅ |
| Recording download (MP4) | ✅ |
| PGN export of board session | ✅ |
| 30-day recording retention | ✅ |
| Graceful reconnection | ✅ |

### Part 2 — Online Play
| Feature | Included |
|---------|---------|
| Link-based, no login | ✅ |
| Two role links (White / Black) | ✅ |
| Full rules enforcement + clock | ✅ |
| In-game chat | ✅ |
| Draw offer / resign / claim draw | ✅ |
| Rematch (colors swapped) | ✅ |
| Spectator mode | ✅ |
