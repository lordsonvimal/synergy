# ChessLeap — Full Platform Requirements

## Context

ChessLeap is being expanded from a single-player proof-of-concept into a **professional online chess learning platform** for coaches and students — comparable in quality to lichess.org and chess.com but with a strong educational and coaching focus. The owner is a chess coach and businessman targeting recurring revenue through student subscriptions, live class bookings, and tournament entry fees.

**What already exists:**
- Full chess engine (bitboards, legal move gen, alpha-beta search) — Go
- Game clock with increment and lag compensation
- SSE-based real-time UI updates via DataStar
- Single-player game UI (chessboard, clocks, promotion overlay)
- Go/Gin backend, Templ SSR, TailwindCSS — no auth, no users, no multiplayer

---

## Module 1 — Authentication & User Management

**Requirement:** Every feature downstream depends on identity.

### 1.1 Registration & Login
- Email + password registration with email verification
- Google OAuth sign-in
- Password reset via email

### 1.2 Roles
| Role | Capabilities |
|------|-------------|
| `student` | Play games, access learning modes, book classes |
| `coach` | All student features + create classes, manage curriculum, view student analytics |
| `arbiter` | Start/pause/abort official games, assign players |
| `admin` | Full platform management |

### 1.3 User Profile
- Display name, avatar, bio
- Internal Elo rating (tracked per game)
- Achievements & badge display
- Visible stats: games played, win/draw/loss, puzzle rating, streak

---

## Module 2 — Board UI Enhancements

**Requirement:** The board must feel professional at the level of lichess/chess.com.

### 2.1 Visual Enhancements
- **Board coordinates**: file letters (a–h) along the bottom, rank numbers (1–8) along the left — flip with board orientation
- **Player color indicator**: colored circle (white/black) rendered next to each player's name at their side of the board
- **Player name display**: full display name shown in the player card
- **Flip board**: button to flip board orientation for analysis

### 2.2 Move Notation Panel
- **Standard Algebraic Notation (SAN)**: render each move as it is played (e.g., `1. e4 e5  2. Nf3 Nc6`)
- Two-column format: White moves on left, Black moves on right
- **Move navigation**: clicking any past move switches board to a read-only view of that position
- Arrow keys (← →) to step through moves
- "Live" button to return to the current game position
- Navigation does not affect the live game — it is view-only

### 2.3 Game Result Banner
- Show result text after game ends: "White wins by checkmate", "Draw by stalemate", etc.
- Rematch button (for casual games)

---

## Module 3 — Real Online Game with Arbiter

**Requirement:** Two remote players connect to play a rated or casual game. An arbiter controls the start.

### 3.1 Game Room
- Arbiter creates a game room and selects time control
- Arbiter assigns White player and Black player by username (search/invite)
- Players receive a join link (or notification when logged in)
- Game lobby page shows assigned players, their online status, and a countdown
- **Game only starts when arbiter clicks "Start Game"**
- Arbiter can abort the game at any point

### 3.2 Player Experience
- Each player sees the board oriented to their color (White at bottom for White, Black at bottom for Black)
- Spectators see the board from White's perspective by default (flippable)
- Moves submitted only from the player whose turn it is — other player cannot interact
- Move is validated server-side and broadcast to both players via SSE
- Both players' clocks run on the server; SSE pushes tick events to all clients

### 3.3 Spectator Mode
- Unlimited spectators can watch any live game
- Spectators see the board and move notation but cannot interact
- Spectator count displayed in game panel

### 3.4 Ratings
- Rated games update both players' internal Elo after completion
- Provisional rating period: first 20 games
- Separate rating pools: Blitz, Rapid, Classical

---

## Module 4 — Learning Modes

### 4.1 Piece Identification Puzzles (Beginner)
- A board position is shown with a highlighted square
- Student identifies the piece on that square (multiple choice or type-in)
- Tracks accuracy over time
- Difficulty: starts with major pieces, progresses to all pieces in complex positions

### 4.2 Board Arrangement Puzzles (Beginner)
- Show a target position image; student places pieces on a blank board to recreate it
- Drag-and-drop interaction
- Timer-based scoring for advanced learners

### 4.3 Tactics Puzzles (Intermediate / Advanced)
- Standard "find the best move" format
- Positions sourced from a curated puzzle database (import from Lichess open puzzle database or manual curation)
- Multi-move combinations (not just one-movers)
- Puzzle rating (Elo-based): puzzle difficulty adjusts to student's rating
- Hint system: request hint costs rating points
- Spaced repetition: failed puzzles resurface on a schedule
- Categories: Fork, Pin, Skewer, Discovery, Checkmate in N, Endgame technique

### 4.4 Opening Theory (Intermediate)
- Structured opening tree: e4, d4, c4, Nf3 openings broken down to 10–15 moves deep
- Each move has a text annotation explaining the idea
- Practice mode: student plays the opening against the engine (engine plays book moves)
- Deviation handling: if student deviates, show the "theory move" and explanation
- Openings organized by: name, ECO code, color (playing as White/Black)
- Coach can assign specific openings to students

### 4.5 Grandmaster Game Database
- Curated library of famous games (starting with ~500 key games, expandable)
- Each game has metadata: players, event, year, result, opening name
- **Per-move annotations**: text notes explaining the idea of key moves
- Step-through interface: click or arrow-key through moves; annotation panel updates per move
- **Search**: by player name, opening (ECO), year range, result
- Difficulty tag: games marked for Beginner / Intermediate / Advanced learners
- Coach can bookmark games and assign them to students as homework
- Import PGN files with annotations (coach uploads their own analysis)

---

## Module 5 — Live Classes

**Architecture: Custom WebRTC** (peer-to-peer video/audio, signaling via backend)

### 5.1 Session Infrastructure
- **Signaling server**: Go backend handles WebRTC offer/answer/ICE exchange via WebSocket
- **STUN**: use public Google STUN for NAT traversal
- **TURN**: deploy a Coturn server (self-hosted) as relay for restrictive NAT environments
- Coach is the "host"; students join as participants
- Max class size: configurable (default 20 students)

### 5.2 Shared Board

The board has four modes, switchable by the coach at any time:

**Teaching Mode** (default) — Coach moves any piece freely; no rules or turn enforcement. Supports takeback, FEN/piece-placement setup, colored arrow and circle annotations (broadcast to all in real time), engine evaluation panel (coach-only by default), and a move notation panel each participant can navigate independently. Coach can grant a student spotlight to take over the board.

**Training Game Mode** — Coach selects two participants (any combination of students or coach + student) and a time control. A rules-enforced timed game runs on the shared board while the rest of the class spectates. Coach can pause/resume clocks, abort, or force a takeback. When the game ends the board automatically returns to Teaching Mode with the final position loaded for analysis.

**Position Drill Mode** — Coach sets a position, then starts a drill. Each student receives a private copy and submits their best single move independently (optional countdown timer). Submissions stay hidden until the coach clicks "Reveal" — all answers appear simultaneously on a per-student board grid for class discussion.

**Student Sandbox Mode** — Each student receives an independent copy of the current position to explore freely (no rules enforcement). Coach sees a live thumbnail grid of all student boards, can watch any board enlarged, and can pull any student's board position to the main teaching board to discuss with the class.

### 5.3 Interaction
- Text chat panel (all participants)
- Students can raise hand to ask a question
- Coach can mute/unmute individual students
- Reactions: thumbs up, confused (emoji reactions without disrupting audio)

### 5.4 Class Management

**MVP (link-based, no login required):** Coach creates a class session and receives two links — a coach link and a student join link. The student link is shared directly (WhatsApp, email, etc.). No scheduling, no booking, no accounts needed. This is the initial implementation described in `REQUIREMENTS_PART1_PART2.md`.

**Full version (requires Module 1 auth, added in a later phase):**
- Coach schedules a class: title, date/time, duration, topic, max seats, price
- Students browse upcoming classes and book (payment required if paid)
- Reminder notification (24h and 1h before class)
- Class recording: opt-in per session, stored for 30 days, accessible to enrolled students
- Recordings library in student dashboard

### 5.5 Post-Class
- Coach can export the annotated board positions from the session as a PGN
- Students receive a session summary with key positions

---

## Module 6 — Tournament System

### 6.1 Tournament Formats
- **Swiss system**: recommended for 8–64 players
- **Round robin**: for small groups (4–8 players)
- **Knockout / Single elimination**: for quick events

### 6.2 Tournament Management (Arbiter)
- Arbiter creates tournament: name, format, time control, rounds, entry fee, start time
- Registration period with optional entry fee collection (Stripe)
- Player registration list with check-in confirmation
- Automatic pairing generation (Swiss: Monrad system)
- Arbiter starts each round; games created automatically per pairing
- Live standings table: points, tiebreaks (Buchholz for Swiss), game results

### 6.3 Player Experience
- Tournament lobby: see upcoming pairings, results, standings
- Notification when paired game is ready
- Game is auto-launched from the tournament context
- Medal display on profile: 1st, 2nd, 3rd place badges

---

## Module 7 — Coach Tools

### 7.1 Student Management
- Coach dashboard: list of enrolled students
- View each student's: game history, puzzle rating trend, class attendance
- Assign homework: specific puzzles, opening lines, GM games to study
- Notes per student (private to coach)

### 7.2 Curriculum Builder
- Coach creates a custom lesson: title, positions, annotations, exercises
- Lessons organized into courses (e.g., "Beginner Tactics Course — 8 lessons")
- Assign course to individual students or a group
- Student progresses through lessons sequentially

### 7.3 Analytics
- Coach sees aggregate class analytics: average puzzle solve rate, common mistakes, attendance
- Individual student progress reports (exportable PDF for parents)

---

## Module 8 — Monetization & Payments

### 8.1 Subscription Tiers

| Tier | Price | Includes |
|------|-------|---------|
| **Free** | $0 | Casual games, 5 puzzles/day, basic game database (no annotations) |
| **Student** | $9/mo | Unlimited puzzles, full GM database, opening theory, class recordings |
| **Premium** | $19/mo | All Student features + live class bookings, priority pairing |
| **Academy** | $49/mo | Coach license: manage up to 50 students, curriculum builder, analytics |

### 8.2 Live Class Bookings
- Per-class pricing set by coach (e.g., $15 per session)
- Package deals: 4-class bundle at a discount
- Platform takes 15% commission; coach receives 85%
- Stripe Connect for direct coach payouts

### 8.3 Tournament Entry Fees
- Platform collects entry fee via Stripe
- Prize pool: configurable (percentage of entry fees, or fixed sponsorship amount)
- Platform fee: 10% of entry fees

### 8.4 Stripe Integration
- Subscription management (create, upgrade, cancel)
- One-time payments (class booking, tournament entry)
- Webhooks for payment confirmation and subscription lifecycle events
- Invoices and receipts emailed automatically

---

## Additional Revenue-Generating Features (Recommended)

| Feature | Revenue Model | Rationale |
|---------|--------------|-----------|
| **Coach Marketplace** | 15% commission | Students discover and book 1-on-1 sessions with vetted coaches; coaches build a client base |
| **Daily Puzzle Challenge** | Engagement → retention → subscription | One puzzle per day with a global leaderboard; drives daily active users |
| **Achievement & Streak System** | Retention → reduces churn | Badges for streaks, milestones ("50 puzzles solved"), visible on profile |
| **Parent Dashboard** | Upsell to Academy | Parents of young learners see weekly reports, screen time, progress; drives coach-license upgrades |
| **Certificates** | Premium upsell | Completion certificates for courses and tournaments; parents love printing these for kids |
| **White-Label for Academies** | Enterprise license $199/mo | Chess clubs and schools use the platform under their own branding |
| **Blindfold Chess Mode** | Premium feature | Pieces are hidden; player must rely on memory — popular training method |
| **Endgame Practice** | Subscription feature | King + Pawn vs King, Rook endings — structured endgame training with a tablebase |

---

## Technical Architecture Decisions

1. **Auth**: Go session cookies (server-side sessions stored in SQLite) or JWT (stateless)
2. **Database**: **SQLite** (WAL mode) for all persistence — users, games, puzzles, courses, payments; replaces in-memory store. Use [litestream](https://litestream.io) for continuous replication/backup to S3-compatible storage (near-zero cost, crash-safe).
3. **WebRTC signaling**: WebSocket endpoint in Go Gin (`/ws/class/:sessionID`)
4. **TURN server**: Coturn self-hosted; configure credentials rotated per session
5. **Payments**: Stripe Checkout + Stripe Connect for coach payouts
6. **PGN parser**: Go library for importing/exporting game notation (parse Lichess puzzle DB)
7. **Move notation (SAN)**: Implement SAN encoder in the existing engine package
8. **Rating system**: Implement Elo calculation in game completion handler

### Deployment Strategy (Cost-Effective + High Performance)
- **Single binary**: Go compiles the entire app (server + templates + embedded SQLite) into one binary — no separate DB process
- **SQLite WAL mode**: enables concurrent reads alongside writes; handles thousands of users on modest hardware
- **Litestream**: streams SQLite WAL changes to S3/R2/Backblaze in real time — free disaster recovery, no managed DB fees
- **Target host**: Hetzner CX22 (2 vCPU, 4 GB RAM, ~$4.50/mo) or Fly.io (auto-scales, free tier available)
- **CDN**: Cloudflare free tier for static assets and DDoS protection
- **Vertical before horizontal**: SQLite + single process scales to ~10k concurrent users on a $20/mo VPS; shard only when needed
- **Estimated infra cost at launch**: $5–20/month total (VPS + object storage for backups + Cloudflare free)

---

## Scope Summary — What is New vs. What Exists

| Component | Status |
|-----------|--------|
| Chess engine (rules, move gen, clock) | ✅ Exists |
| Single-player game UI | ✅ Exists |
| Board coordinates & player names | 🔨 Needs enhancement |
| Move notation panel + navigation | 🔨 New UI component |
| Authentication & user accounts | ❌ Build new |
| Real online multiplayer (2 players) | ❌ Build new |
| Arbiter game room & controls | ❌ Build new |
| Spectator mode | ❌ Build new |
| Learning modes (puzzles, openings, DB) | ❌ Build new |
| Live classes (WebRTC) | ❌ Build new |
| Tournament system | ❌ Build new |
| Coach tools & curriculum | ❌ Build new |
| Payments (Stripe) | ❌ Build new |
| SQLite persistence (WAL + Litestream) | ❌ Build new |

---

## Suggested Build Order (Phased Delivery)

Live Class is the primary product. The build order prioritises getting a fully functional Live Class into coaches' hands before layering on authentication, learning modes, and monetisation.

**Phase 1 — Foundation (3–4 weeks)**
Board UI enhancements → Move notation panel + navigation → SQLite persistence (session storage only) → Online Play (link-based, no login)

**Phase 2 — Live Class MVP (5–7 weeks)**
WebRTC infrastructure (SFU + signaling + TURN) → Teaching board all 4 modes (Teaching, Training Game, Position Drill, Sandbox) → Chat, raise hand, reactions → Session recording + replay → Waiting room + layout management

**Phase 3 — Authentication & Multiplayer (4–6 weeks)**
User accounts (email + OAuth) → User profiles + Elo ratings → Online multiplayer with arbiter → Spectator mode → Scheduled class bookings (full §5.4)

**Phase 4 — Learning (4–6 weeks)**
Tactics puzzles → GM game database with annotations → Opening theory

**Phase 5 — Revenue (3–4 weeks)**
Stripe subscriptions → Paid class bookings → Coach curriculum builder → Coach marketplace

**Phase 6 — Growth & Retention (4–6 weeks)**
Tournament system → Daily challenges → Achievement system → Parent dashboard → Certificates → Analytics
