# ChessLeap — Internet-Grade Chess Clock Design

## Problem

A chess clock must be accurate, smooth, and resilient over the internet where latency is variable and client/server clocks are not synchronized. Naively counting down locally from a snapshot leads to:

- **Clock drift**: client and server diverge over time
- **Jumpy corrections**: periodic hard resyncs snap the display up or down
- **False flag fall**: client flags too early because it didn't account for transit delay
- **Authoritative conflict**: who wins when server and client disagree on flag fall?

---

## Architecture Overview

```
Client                              Server
  │                                   │
  │── GET /ping ──────────────────────▶ t2
  │     t1 (before)                   │── 200 {"server_ns": t2}
  │◀──────────────────────────────────│
  │     t3 (after)                    │
  │  offset = t2 - (t1+t3)/2          │
  │  (3 round trips, take median)     │
  │                                   │
  │── GET /game/:id/events ───────────▶
  │   (persistent SSE connection)     │── clock_tick (1 Hz) ─▶
  │◀──────────────────────────────────│   {white_remaining_ns, black_remaining_ns,
  │                                   │    server_ts_ns, active_side}
  │◀──────────────────────────────────│── board_update (on each move)
  │◀──────────────────────────────────│── game_over (flag fall / checkmate / etc.)
  │                                   │
  │  [server watchdog goroutine]       │
  │                                   │── polls every 100ms
  │                                   │── flags when remaining_ns ≤ 0
  │                                   │── broadcasts game_over SSE event
```

---

## Components

### 1. `GET /ping` — NTP-style RTT Measurement

Called 3× at page load, sequentially. Returns `{"server_ns": <unix_nano>}`.

Client computes:
```
rtt    = t3 - t1                         // round-trip time in ms
offset = server_ns - (t1_ns + t3_ns) / 2 // clock delta (client - server)
```

Take the median offset from the 3 samples (discard outliers caused by congestion spikes). Store as `clockOffset` (nanoseconds as a JS BigInt or high-precision float).

### 2. `GET /game/:id/events` — Persistent SSE Stream

Replaces the per-request DataStar SSE for clock delivery. One long-lived connection per game page.

#### Event types

| Event | Payload | When |
|-------|---------|------|
| `clock_tick` | `ClockTickEvent` | 1 Hz from server watchdog |
| `board_update` | DataStar `patchElements` + `patchSignals` | After each move |
| `game_over` | `GameOverEvent` | Flag fall, checkmate, resign, draw |

#### `ClockTickEvent` (JSON)
```json
{
  "type": "clock_tick",
  "white_remaining_ns": 298000000000,
  "black_remaining_ns": 300000000000,
  "active_side": 0,
  "server_ts_ns": 1714934512000000000
}
```

`server_ts_ns` is the Unix nanosecond timestamp at the moment the server computed `white_remaining_ns` / `black_remaining_ns`. The client uses it for transit compensation (see §4).

#### `GameOverEvent` (JSON)
```json
{
  "type": "game_over",
  "state": 3,
  "winner": 1,
  "state_text": "Clock flagged"
}
```

### 3. Server Watchdog Goroutine

One goroutine per active game, started when the first SSE client connects and stopped when the game ends or all clients disconnect.

```
every 100ms:
  lock game
  if active clock remaining_ns - elapsed <= 0:
    set GameClockFlagged
    set Winner = opponent
    broadcast game_over SSE event
    stop watchdog
  else if 1s since last tick:
    broadcast clock_tick SSE event
  unlock
```

The watchdog is the **authoritative** source of flag fall. The client shows an optimistic "0:00" but only treats the game as over when it receives `game_over` from the server.

### 4. Client `sync()` — Transit-Compensated Smooth Correction

When a `clock_tick` event arrives:

```js
// Step 1: compute adjusted remaining at the moment the packet was sent
const serverSentNs = event.server_ts_ns            // BigInt, unix ns
const clientNowNs  = BigInt(Date.now()) * 1000000n  // approximate unix ns

// Step 2: apply clock offset measured during ping phase
const clientNowAdj = clientNowNs - clockOffsetNs   // remove client/server delta

// Step 3: how long has elapsed since the server computed this tick?
const transitNs = Number(clientNowAdj - serverSentNs)  // ~RTT/2

// Step 4: true remaining at this instant
const trueRemaining = remaining_ns - transitNs

// Step 5: smooth correction
const currentEstimate = clock.remaining - elapsed_since_last_sync
const diff = trueRemaining - currentEstimate

if (Math.abs(diff) < 500_000_000) {   // < 500ms
  // lerp over ~350ms (about 1 RAF frame budget × 20 frames)
  clock.correction = diff
  clock.correctionSteps = 21
} else {
  // Hard resync only on large divergence (network pause / tab sleep)
  clock.remaining = trueRemaining
  clock.baseMono   = performance.now()
  clock.correction = 0
}
```

Each RAF tick applies `correction / correctionSteps` and decrements `correctionSteps`. This spreads a ±500ms correction invisibly over ~350ms — no visible jump.

**Never jump forward**: if `diff > 0` (server says more time remains than client thinks), still lerp — do not snap upward. Only clamp to 0 from below.

### 5. Flag Fall — Dual Detection

| Agent | Role |
|-------|------|
| Client | Optimistic display: shows "0:00" when its estimate hits 0. Does NOT end the game. |
| Server watchdog | Authoritative: broadcasts `game_over` when server confirms flag fall. |

This means in the worst case (250ms one-way latency) the client shows 0:00 for up to 250ms before the `game_over` event arrives — acceptable and matches how Lichess/Chess.com behave.

---

## Data Flow Diagram

```
Move submitted by player
  │
  ▼
POST /game/:id/select/:square
  │── validates move
  │── g.ApplyMove()  ← stops active clock, starts opponent clock
  │── broadcasts board_update SSE via persistent stream
  │── (no clock in DataStar signals any more)
  │
  ▼
Server watchdog (100ms poll)
  │── detects clock tick boundary (1s elapsed)
  │── computes remaining for active side
  │── broadcasts clock_tick SSE event with server_ts_ns
  │── if remaining ≤ 0: sets GameClockFlagged, broadcasts game_over
```

---

## SSE Hub

Each game has a fan-out hub: one writer goroutine, N subscriber channels (one per connected browser tab). When the game ends or the client disconnects, the subscriber is removed.

```go
type GameHub struct {
  mu   sync.Mutex
  subs map[chan []byte]struct{}
}

func (h *GameHub) Subscribe() chan []byte   // returns a buffered channel
func (h *GameHub) Unsubscribe(ch chan []byte)
func (h *GameHub) Broadcast(msg []byte)
```

The hub is stored on the `Game` struct. The watchdog calls `hub.Broadcast()`. The SSE handler calls `hub.Subscribe()` and streams messages to the client.

---

## File Map

| File | Change |
|------|--------|
| `game/clock.go` | Add `RemainingAt(now int64)` helper; add `GameClock.ActiveSide()` |
| `game/events.go` | Define `ClockTickEvent`, `GameOverEvent` structs |
| `game/hub.go` | New: `GameHub` fan-out SSE hub |
| `game/watchdog.go` | New: per-game watchdog goroutine |
| `game/game.go` | Add `Hub *GameHub`; start watchdog on `NewGame`; uncomment flag fall |
| `server/handlers.go` | Add `PingHandler`; add `GameEventsHandler` (SSE stream) |
| `server/routes.go` | Register `GET /ping` and `GET /game/:id/events` |
| `server/sse.go` | Refactor to write to hub, not directly to `c.Writer` |
| `assets/js/clock.js` | Rewrite `sync()` with transit compensation + lerp correction |
| `assets/js/sync.js` | Add `measureOffset()` (3× ping, median); wire SSE event listener |
| `assets/js/initClock.js` | Connect SSE stream; dispatch to `onClockSync` / board updates |

---

## Non-Goals

- No opponent real-time sync (this is local two-player for now)
- No reconnect backoff / session resumption (page refresh is sufficient)
- No sub-100ms clock accuracy (±200ms at P99 internet latency is acceptable for correspondence/blitz)
