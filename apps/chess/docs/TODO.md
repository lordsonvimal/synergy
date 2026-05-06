# ChessLeap — TODO

---

## Phase 1 — Foundation

### 1. Testing (carry-over)

- [ ] Add `data-testid` attributes to all interactive elements
- [ ] Add `.env.example` documenting expected environment variables
- [ ] Add Go unit tests for `engine/` package: legal move generation, check/checkmate detection, castling, en-passant, promotion; and for `/game` HTTP handlers: move validation, clock start/stop, game-over detection

---

### 2. Board UI Enhancements

- [ ] Add flip board button (toggles board orientation for analysis)
- [ ] Board coordinates (a–h, 1–8) flip when board orientation flips

---

### 3. Move Notation Panel

- [ ] Implement SAN encoder in the engine package (`Move` → standard algebraic notation string)
- [ ] Track move history on `Game` struct (slice of SAN strings + FEN snapshot per move for position reconstruction)
- [ ] Build `MoveNotationPanel` component: two-column layout (White left, Black right), `1. e4 e5  2. Nf3 Nc6 …`
- [ ] Render notation in real time as moves are played; broadcast updated panel via SSE
- [ ] Move navigation: clicking a past move switches the board to a read-only snapshot of that position
- [ ] Arrow key `←` `→` navigation through move history
- [ ] "Live" button returns from history view to the current position
- [ ] Navigation is view-only — does not affect the live game state or clocks

---

### 4. SQLite Persistence

- [ ] Add SQLite driver dependency (`modernc.org/sqlite` — pure Go, no CGo)
- [ ] Design and document schema: `sessions` (class + play), `participants`, `moves` (game + teaching), `annotations` (arrow/circle per move), `chat_messages`; write SQL migration files
- [ ] Implement schema migrations with version tracking (run on startup)
- [ ] Migrate `GameStore` from in-memory map to SQLite-backed store
- [ ] Replace filesystem game event log files (`.gamewal`) with SQLite rows in the `moves` table
- [ ] Enable WAL mode (`PRAGMA journal_mode=WAL`) for concurrent reads alongside writes
- [ ] Add Litestream config for continuous replication to S3-compatible storage

---

### 5. Online Play (`/play`)

- [ ] Add routes: `GET /play`, `POST /play`, `GET /play/:gameID`, `GET /play/:gameID/events`, `POST /play/:gameID/select/:square`
- [ ] Game creation page at `/play`: time control selector + color preference (White / Black / Random)
- [ ] On creation, server returns two distinct links (White link + Black link) each with a single-use token
- [ ] "Created" page shows both links with individual copy buttons
- [ ] Role claiming: first browser to open a color link claims that role; role bound to session cookie
- [ ] Refreshing the page restores role from session cookie
- [ ] Token invalidation: opening a token URL after role is claimed → assigned Spectator
- [ ] Each player sees the board oriented to their own color
- [ ] Clock starts only when both players have opened their links and are marked online
- [ ] Online status indicator per player (green dot = connected, grey = disconnected)
- [ ] Opponent disconnect banner; 60-second reconnection grace period before clock continues
- [ ] In-game chat panel: players send; spectators read only; messages prefixed with color (`[White] Good luck!`)
- [ ] Draw offer: "Offer Draw" → opponent notification (Accept / Decline, 30s expiry); one active offer at a time
- [ ] Resign: button → "Are you sure?" confirmation → game ends immediately
- [ ] Claim draw: button appears automatically on threefold repetition or 50-move rule
- [ ] Rematch: both players see button after game ends; if both accept within 60s, a new game is created with colors swapped and the same page reloads with the new game state — no new links need to be shared
- [ ] Spectator count shown in session header
- [ ] Retire `/game` route (redirect to `/play` or remove)

---

## Phase 2 — Live Class MVP

### 6. Live Class Session

- [ ] Add routes: `GET /class`, `POST /class`, `GET /class/:sessionID` (student), `GET /class/:sessionID?role=coach&token=...` (coach)
- [ ] On creation, server returns two links: coach link (single-use token) and student join link (shared broadly)
- [ ] Role claiming: first browser to open coach link claims Coach role; all subsequent joins are Students
- [ ] Roles bound to session cookie; refreshing restores role
- [ ] Session state machine: `waiting` → `live` → `ended`
- [ ] Idle session expiry after 4 hours; ended sessions archived for replay access
- [ ] Persist session state in SQLite (participants, board state, chat history, mode transitions)

---

### 7. WebRTC Infrastructure

- [ ] Add [Pion WebRTC](https://github.com/pion/webrtc) dependency to `go.mod`
- [ ] Build SFU: each participant opens one connection to the server; server forwards streams (no peer-to-peer mesh)
- [ ] WebSocket signaling endpoint at `/ws/class/:sessionID` (offer / answer / ICE candidate exchange)
- [ ] Configure STUN: `stun.l.google.com:19302`
- [ ] Add Coturn TURN server config files (`turnserver.conf`) and docker-compose entry; configure HMAC-based time-limited credentials rotated per session [ops task — separate from app code]
- [ ] Client reconnection loop: retry every 3s for up to 2 minutes; show "Connection lost" error on timeout
- [ ] Connection quality classification per participant (green / yellow / red) based on packet loss + RTT

---

### 8. Video & Audio Controls

- [ ] Mute/unmute toggle (microphone icon, red when muted)
- [ ] Camera enable/disable toggle (crossed-out icon when off; show avatar/initials fallback tile)
- [ ] "Leave Class" button with confirmation dialog
- [ ] Coach remote mute: mutes any student's mic; student receives toast "Coach muted your mic"
- [ ] Coach unmute request: sends prompt to student; student must accept before unmuting
- [ ] Coach remove participant from session (with confirmation)
- [ ] Speaking indicator: animated border ring on active speaker's video tile
- [ ] Volume meter displayed on each tile when audio is active

---

### 9. Layout Management

- [ ] Layout selector rendered as `[⊞▼]` button in the video panel header; options: Speaker View, Grid View, Strip (default), Coach Broadcast Mode
- [ ] **Speaker View**: active speaker fills 70% of video panel height; others in thumbnail strip below; 1.5s debounce on auto-switch
- [ ] **Grid View**: equal-sized tiles in responsive grid; up to 20 tiles; reflow as participants join/leave
- [ ] **Strip (default)**: vertical list of tiles in the left video panel; board expands to fill remaining width
- [ ] **Coach Broadcast Mode**: only coach's video shown; student streams still transmitted but not rendered; coach can spotlight one student into their tile
- [ ] Pin: any participant can pin any video tile; pin persists regardless of active speaker; pin icon shown; one active pin per client
- [ ] Coach pins from participants panel; students pin by clicking the tile

---

### 10. Teaching Board — Teaching Mode

- [ ] Board mode selector in board toolbar (Teaching / Training Game / Drill / Sandbox); switching broadcasts mode transition to all participants via SSE
- [ ] Coach moves any piece freely (no turn or rules enforcement); moves broadcast via SSE
- [ ] Takeback: undo last move via button or `Ctrl+Z`; multi-step undo supported; board state broadcast after each undo
- [ ] FEN setup modal — FEN text input: validate on change, apply to board on valid FEN, show error inline on invalid
- [ ] FEN setup modal — side to move, castling rights, and en-passant square selectable alongside the FEN input
- [ ] FEN setup modal — "Reset to starting position" button clears the FEN field and resets the board
- [ ] Piece placement editor: blank board with piece palette; drag pieces onto squares; click a placed piece to remove it
- [ ] Colored arrow annotations: click-and-drag source → destination; color picker (Green, Red, Yellow, Blue; default Green); broadcast to all
- [ ] Circle highlights: right-click square to toggle; same color picker; broadcast to all
- [ ] "Clear all annotations" button + `Escape` shortcut
- [ ] Annotations stored per move: navigating to a past move restores its saved annotations
- [ ] Engine evaluation toggle: eval bar (±centipawns, vertical), best-move arrow (purple), top 3 candidate moves with scores
- [ ] Engine visible to coach only by default; "Show engine to students" toggle
- [ ] Engine runs server-side (existing alpha-beta); depth configurable, default 15
- [ ] Move notation panel (same as Online Play: SAN, two-column, per-client navigation, Live button)
- [ ] Board coordinates visible; flip with board orientation
- [ ] Student spotlight: coach assigns board control to one student at a time; coach can revoke at any time

---

### 11. Training Game Mode

- [ ] Coach selects two participants (students or coach + student) as Player 1 (White) and Player 2 (Black)
- [ ] Time control selector: 3+0, 5+2, 10+5, 15+10
- [ ] "Start Game" transitions board to training game layout: clocks shown above/below the board; chat stays open
- [ ] All other participants become spectators (read-only board + notation)
- [ ] Full rules enforcement; moves validated server-side; clocks run server-side with SSE tick events (reuse existing clock + watchdog)
- [ ] Coach-only controls: Pause / Resume clocks; Abort game; Force Takeback (10s rejection window for players; coach can override)
- [ ] Game-over detection: result banner shown; board auto-returns to Teaching Mode with the final position loaded for analysis
- [ ] Training game move sequence included in session recording

---

### 12. Drill Mode (Drill Framework)

**Shared infrastructure:**
- [ ] Add drill routes: `POST /class/:sessionID/drill/start`, `POST /class/:sessionID/drill/submit`, `POST /class/:sessionID/drill/reveal`, `DELETE /class/:sessionID/drill`
- [ ] Drill type selector UI in the coach's board area (radio list of 7 types + timer picker)
- [ ] Coach setup flow: type selection → type-specific config → Start Drill
- [ ] "Start Drill" broadcasts private drill copy to each student via SSE
- [ ] Participants panel shows live submission status per student: Thinking… / Submitted (no answer content shown)
- [ ] Optional timer (30s, 60s, 90s, none): countdown on each student board; auto-submit "No answer" on expiry

**Drill type 1 — Find the Move:**
- [ ] Coach sets position (FEN or piece placement) + optional prompt text
- [ ] Student: click a destination square to submit; first click locks the move
- [ ] Auto-validate: server compares submitted move to engine's top move (✓ / ✗)

**Drill type 2 — Find the Mate:**
- [ ] Coach sets position + specifies Mate in N (1–5)
- [ ] Student: plays moves on their board; board auto-submits on checkmate; "Submit" button available
- [ ] Auto-validate: ✓ if checkmate in ≤ N; ≈ if checkmate in > N; ✗ if no checkmate

**Drill type 3 — Arrange the Board:**
- [ ] Coach selects a target position (FEN or piece placement); target hidden from students
- [ ] Student: receives blank board + piece palette; drag pieces to squares; "Submit" locks the arrangement
- [ ] Timer bonus indicator shown on reveal tile if completed within time
- [ ] Auto-validate: ✓ exact match; tile shows misplaced piece count otherwise

**Drill type 4 — Name This Piece:**
- [ ] Coach selects a position + highlights one or more squares; configures free-text or multiple-choice mode
- [ ] Student: types piece name or selects from list; Enter or "Submit" locks the answer
- [ ] Auto-validate: ✓ case-insensitive match including abbreviations (N, R, B, Q, K, P)

**Drill type 5 — Write the Notation (SAN):**
- [ ] Coach sets a position + records one or more moves (hidden from students); writes prompt text
- [ ] Student: types SAN string in a text input; Enter or "Submit" locks the answer
- [ ] Auto-validate: ✓ exact SAN match against the coach's recorded answer

**Drill type 6 — Spot the Illegal Move:**
- [ ] Coach provides a list of 3–5 SAN moves for a given position; marks which one is illegal
- [ ] Student: radio button list of moves; clicks the illegal one and submits
- [ ] Auto-validate: ✓ / ✗ — single correct answer

**Drill type 7 — Play the Opening:**
- [ ] Coach pastes a PGN or selects from a built-in ECO opening list; specifies depth N (moves to play); the book line is stored server-side for the drill session
- [ ] Student: plays moves on their board; each correct book move confirmed with green highlight; first deviation flagged inline ("Off-book — expected Bb5")
- [ ] Auto-validate: ✓ completed all N book moves correctly; reveal tile shows the deviation move otherwise

**Private preview (shared, all types):**
- [ ] [👁 Preview] button on each Submitted student in the participants panel
- [ ] Clicking Preview loads that student's answer onto the coach's board privately (full sequence for multi-move types)
- [ ] Persistent "Private preview — [Name]'s answer (not visible to students)" banner on coach's board
- [ ] Student is never notified that their answer is being previewed
- [ ] [← Prev] / [Next →] cycles through all submitted answers in preview mode without leaving it
- [ ] [Skip] advances to the next student without revealing the current answer

**Staged reveal (shared, all types):**
- [ ] [Show to Class]: reveals currently previewed student's answer to all participants; adds tile to reveal grid with auto-validation result (✓ / ✗ / ≈)
- [ ] Reveal selected: coach selects multiple students from participants panel and reveals as a group
- [ ] Reveal All: broadcasts all submitted answers simultaneously; board panel switches to full-width reveal grid
- [ ] Revealed tiles show student name + answer + validation symbol; unrevealed tiles show name in greyed state
- [ ] Students see hidden tile names but cannot see answers until coach reveals
- [ ] No-answer students marked "No answer" in reveal grid

**Post-drill discussion (shared, all types):**
- [ ] Coach can click any revealed tile to load that student's answer onto the main board for discussion
- [ ] Coach annotations on main board after reveal broadcast to all participants
- [ ] "End Drill" discards all unrevealed submissions (never shown to class) and returns to Teaching Mode

---

### 13. Student Sandbox Mode

- [ ] "Open Sandbox" detaches each student's board from the main board into an independent copy
- [ ] Students move pieces freely on their own board (no rules enforcement, no turns)
- [ ] Students cannot see each other's sandbox boards
- [ ] Coach's board panel switches to a thumbnail grid showing all student boards in real time
- [ ] Coach can click a thumbnail to enlarge it for close watching (student is not notified)
- [ ] "Pull to Main Board": copies selected student's sandbox position to the main teaching board; all participants return to Teaching Mode
- [ ] "Close Sandbox": all student boards reset to the main board's current position; Teaching Mode resumes

---

### 14. Chat & Interactions

- [ ] Text chat panel throughout the session: all participants send; role badge (Coach / Student) + timestamp per message
- [ ] Coach can clear chat for all participants
- [ ] Coach can mute a student's chat; muted student sees "Your chat has been muted by the coach"
- [ ] Raise Hand button: hand icon appears on student's video tile; coach sees notification + ordered queue in participants panel
- [ ] Coach acknowledges hand (removes from queue); student can lower their own hand at any time
- [ ] Emoji reactions (Thumbs Up, Confused, Clap, Surprised): floating animation over sender's tile; fades after 3s; no audio interruption

---

### 15. Screen Sharing

- [ ] Coach "Share Screen" → browser screen picker; shared stream replaces board area or appears as large video tile (coach chooses)
- [ ] When sharing in board area: board moves to minimized panel; "Return to board" button swaps back
- [ ] Annotation overlay on shared screen: coach draws arrows and highlights via a canvas layer
- [ ] "Stop Sharing" ends the share; board view restored automatically
- [ ] Student "Request Screen Share" button; coach receives notification and approves/denies; only one student shares at a time
- [ ] Approved student's screen shown in the board area for the coach; coach can broadcast it to all students (replaces the board area for students too; board moves to a minimised panel)
- [ ] Screen share ends automatically if coach revokes or student leaves

---

### 16. Participants Panel

- [ ] Slide-out panel: name, role badge, audio status, video status, hand raised indicator, connection quality per participant
- [ ] Coach actions per participant (context menu / hover): mute mic, send unmute request, spotlight on video, grant board control, remove from session
- [ ] Training Game Mode: show Player 1 / Player 2 / Spectator assignment; "Assign as Player 1 / Player 2" actions
- [ ] Drill Mode: show Thinking… / Submitted status per student; [👁 Preview] action on each Submitted entry
- [ ] Student Sandbox Mode: show "Watch board" and "Pull board to main" actions
- [ ] Participant count shown in toolbar at all times

---

### 17. Waiting Room

- [ ] Students joining before coach or before "Start Class" land in the waiting room
- [ ] Waiting room shows: class title, coach name, start time, countdown timer
- [ ] Students preview their own camera and microphone (video preview + mic level indicator)
- [ ] Coach sees list of waiting students; can admit all at once or individually

---

### 18. Session Recording

- [ ] Coach starts/stops recording manually; "REC" indicator (red dot + label) visible to all when active
- [ ] All participants notified when recording starts and stops
- [ ] Coach can pause/resume recording (creates timeline chapters)
- [ ] Server-side mixing: record individual Pion RTP tracks to disk during the session; post-session, combine via FFmpeg into a single timestamped MP4 (one video track per participant, mixed audio)
- [ ] Record board move sequence with timestamps
- [ ] Record annotations (arrows, circles) per move
- [ ] Record board mode transitions with timestamps (used as chapter labels in replay)
- [ ] Record chat messages with timestamps
- [ ] Storage: local filesystem by default; S3-compatible via `RECORDING_STORAGE` env var
- [ ] 30-day retention by default (configurable); access restricted to session participants
- [ ] Coach can download MP4; coach can delete before retention period expires

---

### 19. Session Replay

- [ ] Replay page at `/class/:sessionID/replay`; page deleted after 30 days
- [ ] Video player synced with board replay: use the video element's `currentTime` as the authoritative clock; on each `timeupdate` event, look up the move whose timestamp is ≤ `currentTime` and render that board position
- [ ] Chat log displayed alongside, scrolling in sync with video
- [ ] Timeline scrubber: clicking a point jumps both video and board to that moment
- [ ] Board mode transitions shown as labelled chapters on the timeline
- [ ] Annotations (arrows, circles) per move restored as replay advances
- [ ] PGN download of the full board session (annotation comments embedded)
- [ ] Coach can share replay link with students

---

### 20. Class End

- [ ] "End Class" button → confirmation dialog → session state set to `ended`; no new joins allowed
- [ ] All participants notified: "The class has ended"
- [ ] Active recording auto-stopped and processing begun on session end
- [ ] Post-class summary for coach: duration, attendee count, peak attendees, attendee list, recording link (once processed), PGN download

---

## Priority Order

1. Testing (carry-over)
2. Board UI Enhancements
3. Move Notation Panel
4. SQLite Persistence
5. Online Play (`/play`)
6. Live Class Session
7. WebRTC Infrastructure
8. Video & Audio Controls
9. Layout Management
10. Teaching Board — Teaching Mode
11. Training Game Mode
12. Drill Mode
13. Student Sandbox Mode
14. Chat & Interactions
15. Screen Sharing
16. Participants Panel
17. Waiting Room
18. Session Recording
19. Session Replay
20. Class End
