# Competitive Play Roadmap

This document tracks the improvements needed to make the chess app viable for rated competitive play across regions.

---

## Already implemented

| # | Change | Where |
|---|---|---|
| 1 | **Lag compensation** — client stamps `clientTsNs` on each move POST; server subtracts one-way transit time (capped at 200 ms) from the mover's clock via `Clock.Stop` | `ui_store/chessboard_signals.go`, `chesssquare.templ`, `server/sse.go` |
| 2 | **SSE reconnect watchdog** — server sends a `keepaliveTs` signal every 10 s per connection; client reloads the page if 25 s pass without one | `server/play_handlers.go`, `assets/js/initClock.js` |
| 3 | **Per-move DB writes** — `Batch.Append` now spawns an immediate goroutine flush; every move hits SQLite within milliseconds, not up to 30 s later | `game/batch.go` |
| 4 | **Optimistic move UI (was P0)** — chessops bundled into `assets/js/board.js` handles selection highlights and move preview entirely client-side; piece DOM is moved before the POST fires, then the server's board morph reconciles (idempotent on the common case). Castling/en-passant/promotion all handled. The 150–250 ms RTT freeze on the mover's piece is gone. | `assets/js/board.js`, `engine/board.go` (FEN), `ui/ui_store/chessboard_signals.go` (Fen field), `server/sse.go` (`applyMoveRequest`, atomic POST response), `server/routes.go` (new `/move/:from/:to`) |
| 5 | **Atomic move delivery to mover (online)** — POST response now carries the full board HTML + notation + signals in one SSE write (was: signals on POST + board via hub). Removes the perceived gap between "highlight clears" and "piece appears in new square". | `server/sse.go::broadcastBoard` / `applyMoveRequest` |
| 6 | **Content-hashed asset pipeline + gzip** — `dist/` is built by esbuild (`scripts/build-js.mjs`) plus a small asset hasher (`scripts/build-assets.mjs`). `dist/manifest.json` maps logical names → fingerprinted filenames; `helpers.Asset()` resolves them in templ; `server/static.go` serves precompressed `.gz` siblings when the client accepts gzip and sets `immutable` cache headers on hashed names. Importmap lets every bundle share one datastar runtime. | `scripts/build-{js,assets}.mjs`, `ui/helpers/assets.go`, `server/static.go`, `ui/layouts/layout.templ`, `apps/chess/package.json` |
| 7 | **Hot manifest reload** — `helpers.LoadAssetManifest` re-reads on mtime change, so an asset rebuild against a running dev server no longer strands hashed URLs that no longer exist on disk. | `ui/helpers/assets.go` |

### Bug fixes (during the optimistic-UI rollout)

| # | Bug | Where |
|---|---|---|
| B1 | **Castling rights cleared for the wrong colour** — `ApplyMove`'s rook-corner switch mapped white squares to black bits and vice-versa, so moving black's h8 rook silently revoked white's kingside right (and the inverse: moving your own rook left the bit set, allowing a "castle" with a missing rook). | `engine/board.go::ApplyMove` |
| B2 | **`UnapplyMove` corrupted the board on castling undo** — same colour-swap pattern AND wrote to the wrong colour's rook bitboard, producing impossible positions (e.g. white rook on h8) on every castle-then-undo. Latent in production (no callers undo) but breaks search/perft and any future analysis path. | `engine/board.go::UnapplyMove` |
| B3 | **FEN emitted `a9` instead of `-` for no-en-passant** — sentinel comparison was against `255`, but `engine.NoSquare == 64`; `64 / 8 = 8 → '9'`, producing invalid squares that chessops rejected with `ERR_EP_SQUARE`. Surfaced as "selection stops working after first moves" because the client left its position unanchored. | `engine/board.go::FEN`, `engine/board.go::Reset` |
| B4 | **Selection flicker / move latency under fast clicks** — fixed via the new optimistic UI; the earlier `selectionSeq` defence-in-depth machinery (server gate, freshness gate, pending state) was removed since selection is now fully local. | `assets/js/board.js`, `server/sse.go` |

Engine regression tests for B1–B3 live in `engine/engine_test.go` (`TestRookMove*`, `TestCastleAndUndoRoundTrip_*`, `TestFEN*`).

---

## Remaining improvements

### P0 — Required for rated competitive play

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

**Status update:** the gzip layer added with the asset pipeline doesn't apply to SSE board patches (gzip + SSE breaks streaming on many proxies). So the 5–10 KB/move is wire-true. Optimistic UI hides the latency for the *mover* but the opponent still waits on full HTML.

**Implementation sketch:**
- Replace `hubBroadcastBoard` with a signal patch carrying `{from, to, promo, san, fen}`. Client already has chessops loaded (see "Optimistic move UI") — `assets/js/board.js::applyDom` already does the DOM swap, just route the opponent's incoming move through the same path.
- Notation row can be a small element-patch appending one `<tr>` instead of re-rendering the whole panel.
- Complexity: **low–medium** now that the client engine exists.

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
