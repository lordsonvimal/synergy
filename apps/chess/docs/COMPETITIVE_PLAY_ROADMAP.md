# Competitive Play Roadmap

This document tracks the improvements needed to make the chess app viable for rated competitive play across regions. The three quick wins (lag compensation, SSE reconnect, per-move DB writes) have already been implemented.

---

## Already implemented

| # | Change | Where |
|---|---|---|
| 1 | **Lag compensation** — client stamps `clientTsNs` on each move POST; server subtracts one-way transit time (capped at 200 ms) from the mover's clock via `Clock.Stop` | `ui_store/chessboard_signals.go`, `chesssquare.templ`, `server/sse.go` |
| 2 | **SSE reconnect watchdog** — server sends a `keepaliveTs` signal every 10 s per connection; client reloads the page if 25 s pass without one | `server/play_handlers.go`, `assets/js/initClock.js` |
| 3 | **Per-move DB writes** — `Batch.Append` now spawns an immediate goroutine flush; every move hits SQLite within milliseconds, not up to 30 s later | `game/batch.go` |

---

## Remaining improvements

### P0 — Required for rated competitive play

#### Optimistic move UI
**What:** Client predicts and renders the resulting board position immediately on click, before the server round-trip completes. If the server rejects the move (race condition, illegal move), the client reverts.

**Why it matters:** At 150–250 ms RTT, the mover currently sees their piece freeze in place for 250–500 ms. This is the single biggest UX difference versus chess.com / Lichess.

**Implementation sketch:**
- Port the core move application logic from `engine/` to WebAssembly (Go → WASM) or re-implement move validation in JS using the FEN from the last server-confirmed state.
- On click: speculatively apply the move client-side and render the new board; simultaneously POST to server.
- On server SSE response: if the server board matches the speculative board, keep it. If not, hard-reset from the server-authoritative board HTML.
- Complexity: **high** (requires a JS/WASM chess engine or duplicated move logic).

#### Periodic NTP re-sync mid-game
**What:** `measureClockOffset` in `assets/js/sync.js` runs once at page load. If RTT changes mid-game (route flaps, proxy changes), the measured offset becomes stale and clock display drifts.

**Why it matters:** On 1+0 bullet, a 250 ms drift is 25% of the time increment. Players lose on flag while their display shows time remaining.

**Implementation sketch:**
- Re-run `measureClockOffset()` every 60 s in a `setInterval`.
- Gradually apply the new offset via the existing `correctionNs` mechanism in `ClientClock.sync()` rather than hard-jumping.
- Complexity: **low** — ~10 lines of JS.

---

### P1 — Quality of life for competitive play

#### Move delta payloads (reduce per-move bandwidth)
**What:** Currently every move broadcasts full board HTML + notation panel (~5–10 KB). Switch to sending only the changed squares and the new move entry.

**Why it matters:** On slow links (mobile cross-region), 5–10 KB per move adds visible latency. Over a 60-move blitz game both players exchange ~600–800 KB of HTML. Delta payloads would cut this to ~200–500 bytes per move.

**Implementation sketch:**
- Replace `hubBroadcastBoard` with a signal patch carrying `{from, to, promo, san, fen}`.
- Client JS applies the move to the DOM directly: swap piece elements on from/to squares, append notation row.
- Complexity: **medium** — requires client-side DOM manipulation to replace templ rendering for moves.

#### Bound the in-memory game store
**What:** `store/gamestore.go` holds all active games in an unbounded map. Games are never evicted.

**Why it matters:** At scale (10 k+ concurrent games), memory grows indefinitely. A server restart after memory exhaustion loses all in-flight games.

**Implementation sketch:**
- Add an LRU eviction policy with a configurable max (e.g. 5 000 games). On eviction, ensure the game is fully flushed to DB via `Batch.FlushAndStop`.
- On a cache miss (game not in map), restore from DB via the existing `loadPlayGameFromDB` path.
- Add a configurable `MAX_GAMES` env var to `config/env.go`.
- Complexity: **low-medium** — ~50 lines of Go; needs careful handling of ongoing games.

#### Bound the hub subscriber buffer
**What:** Each subscriber channel has capacity 32 (`game/hub.go:32`). `Broadcast` drops messages non-blocking when the buffer is full — a slow cross-region client silently misses board updates.

**Why it matters:** Dropped messages mean the client's board falls behind with no indication. Combined with no SSE reconnect (now fixed via reload), this could leave a player on a stale position.

**Implementation sketch:**
- Replace the non-blocking `select` drop with a `select` + short timeout (e.g. 50 ms). If the subscriber still can't receive after the timeout, unsubscribe it forcibly so it picks up the full state on the next reconnect.
- Or increase the buffer to 128 and log when it fills.
- Complexity: **low**.

---

### P2 — Scale and infrastructure

#### Multi-region deployment
**What:** SQLite is a single-writer file. All players must connect to the same server process. Cross-region players suffer full propagation delay on every move POST.

**Options (escalating complexity):**
1. **Edge SSE termination only (recommended first step):** Deploy to Fly.io with regions. SSE connections terminate at the nearest region (low-latency push), but all move POSTs still route to the primary region. This halves the perceived lag for the *server→client* direction only.
2. **Read replicas via Litestream:** Stream the SQLite WAL to S3 / R2; restore replicas in secondary regions for read-heavy paths (game history, spectators). Writes still go to primary.
3. **Active-active writes:** Requires replacing SQLite with a distributed store (CockroachDB, Turso, Spanner). Full active-active across regions; no single writer. Significant rewrite of `db/` layer.

**Complexity:** Low (step 1) → Very high (step 3).

#### SQLite backup and point-in-time recovery
**What:** The DB is a single file with no offsite backup. A disk failure loses all game history.

**Implementation sketch:**
- Add a cron that runs `VACUUM INTO '/backup/chess-$(date).db'` hourly, then syncs to object storage (S3 / R2 / B2).
- Or use Litestream for continuous WAL streaming — simpler and handles crash recovery automatically.
- Complexity: **low**.

---

### P3 — Anti-cheat and integrity

#### Move-time telemetry
**What:** The `think_time_ns` column is already populated per move. Use it.

**Implementation sketch:**
- Build a post-game analysis endpoint that computes: mean think time, think time variance, percentage of moves matching top-3 engine choices (requires running Stockfish on the stored FEN/UCI sequence).
- Flag games where think time < 0.5 s on 20+ consecutive moves (engine-like play).
- Complexity: **medium** — mostly analysis infrastructure; engine integration is the hard part.

#### IP / device fingerprint per game session
**What:** Currently there is no record of what IP or device submitted moves. Hard to investigate collusion or account sharing.

**Implementation sketch:**
- Log `X-Forwarded-For` (or `RemoteAddr`) and a browser fingerprint (User-Agent + Canvas hash via JS) to a `game_sessions` table on session start.
- Associate moves with their source IP in `moves` table.
- Complexity: **low** — mostly schema and logging.

#### Rating and pairing system
**What:** No rating, no fair matchmaking. Required before opening to competitive public play.

**Implementation sketch:**
- Implement Glicko-2 (or Elo as a first pass): update ratings after each completed rated game.
- Add a matchmaking queue: players select time control → enter queue → server pairs the two closest-rated players.
- Schema additions: `players` table (rating, rd, volatility), `rated_games` linking game to players.
- Complexity: **high** — matchmaking queue requires a persistent queue (can use a DB-backed table + polling; no Redis needed for low volume).
