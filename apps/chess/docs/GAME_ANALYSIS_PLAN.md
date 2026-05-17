# Game Analysis — Design & Implementation Plan

## Goals

Provide a chess.com/Lichess-style "Game Review" experience covering:

- Evaluation bar (advantage in centipawns or mate-in-N)
- Per-move classification: Book / Best / Excellent / Good / Inaccuracy / Mistake / Blunder / Miss / Great / Brilliant / Forced
- Opening identification (ECO code + name + last book ply)
- Top-5 candidate moves (MultiPV) with scores
- Rule-based positional summary (material, king safety, structure, activity)
- Optional LLM-generated commentary / plans on demand
- Per-side accuracy %, list of critical moments, average centipawn loss

## Locked decisions

| # | Decision | Choice |
|---|----------|--------|
| 1 | Engine backend | **Bundle Stockfish** as a sidecar UCI process per platform |
| 2 | "Plans" / commentary | **Both**: rule-based summary shown by default, LLM commentary on user request |
| 3 | Live policy for online play | **Off until game ends** for players. Spectators may opt in mid-game. Solo / vs-bot games always on once enabled in settings |
| 4 | Solo game persistence | Persist completed solo games to existing `games`/`moves` tables (prerequisite — analysis can't outlive a tab otherwise) |

---

## Architecture

```
┌──────────────┐    enqueue     ┌──────────────────┐
│  HTTP/SSE    │──────────────▶│  analysis.Queue   │
│  handlers    │                │  (bounded chan)   │
└──────────────┘                └─────────┬─────────┘
       ▲                                  │
       │ SSE stream                       ▼
       │                        ┌──────────────────┐    UCI    ┌──────────┐
       │                        │ Worker pool      │──────────▶│ stockfish│
       │                        │ (NumCPU/2 max)   │           │ (subproc)│
       │                        └─────────┬────────┘           └──────────┘
       │                                  │
       │                          writes  ▼
       │                        ┌──────────────────┐
       └────────────────────────│ move_analyses    │
                                │ analysis_jobs    │
                                │ openings (ECO)   │
                                └──────────────────┘
```

### Packages

| Package | Responsibility |
|---------|----------------|
| `engine/uci/` | Spawn & speak UCI to Stockfish. Pool of persistent processes. Streaming `Analyze(fen, opts) <-chan Update`. |
| `analysis/` | Job queue, worker loop, classification math, win% conversion, MultiPV interpretation. |
| `analysis/opening/` | ECO book loader + zobrist→opening lookup. |
| `analysis/features/` | Rule-based position-feature extractor (material, king safety, pawn structure, mobility). |
| `analysis/commentary/` | LLM client + prompt templates. Behind interface so it can be no-op'd. |
| `server/analysis_handlers.go` | HTTP + SSE endpoints. |
| `ui/components/analysis*.templ` | Eval bar, top-5 panel, classification chips, commentary panel. |

---

## Stockfish bundling

- Binaries committed under `assets/engines/stockfish-<os>-<arch>` (or downloaded by `scripts/fetch-engines.sh` on first build; lock to a specific SF version).
- Platforms: `darwin-arm64`, `darwin-amd64`, `linux-amd64`. Windows = install instructions in README for now.
- Resolution at runtime: `os.Executable()` → sibling `engines/` dir, fallback to `$PATH stockfish`, fallback to env `CHESSLEAP_STOCKFISH_PATH`.
- Process pool: `min(NumCPU()/2, 4)` long-lived processes. Each handles one `Analyze` call at a time; queue serialises. `setoption name Threads value 1` per process so pool size = parallelism.
- Health: if a process exits unexpectedly, respawn on next request; failed job marked `failed` with error text.
- License: Stockfish is GPL — note in `LICENSE-THIRD-PARTY.md`, keep source link in About page.

---

## Data model

New migration `db/migrations/000003_analysis.up.sql`:

```sql
CREATE TABLE move_analyses (
  game_id        TEXT    NOT NULL REFERENCES games(id) ON DELETE CASCADE,
  seq            INTEGER NOT NULL,                -- 0 = initial position eval
  depth          INTEGER NOT NULL,
  multipv        INTEGER NOT NULL,                -- rank (1..5)
  score_cp       INTEGER,                          -- nullable when mate
  mate_in        INTEGER,                          -- signed; null when score_cp set
  pv_uci         TEXT    NOT NULL,                 -- space-separated UCI moves
  pv_san         TEXT    NOT NULL,                 -- precomputed for display
  engine_name    TEXT    NOT NULL,                 -- "stockfish"
  engine_version TEXT    NOT NULL,
  computed_at    INTEGER NOT NULL,
  PRIMARY KEY (game_id, seq, multipv, engine_version)
);

CREATE TABLE move_classifications (
  game_id        TEXT    NOT NULL REFERENCES games(id) ON DELETE CASCADE,
  seq            INTEGER NOT NULL,
  classification TEXT    NOT NULL,
  cp_loss        INTEGER,
  wp_loss        REAL,                              -- win% delta, 0..100
  best_uci       TEXT    NOT NULL,
  engine_version TEXT    NOT NULL,
  PRIMARY KEY (game_id, seq, engine_version)
);

CREATE TABLE analysis_jobs (
  game_id    TEXT    PRIMARY KEY REFERENCES games(id) ON DELETE CASCADE,
  status     TEXT    NOT NULL CHECK(status IN ('queued','running','done','failed','canceled')),
  progress   INTEGER NOT NULL DEFAULT 0,
  total      INTEGER NOT NULL,
  error      TEXT,
  requested_by TEXT,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);

CREATE TABLE openings (
  zobrist  INTEGER PRIMARY KEY,
  eco      TEXT NOT NULL,
  name     TEXT NOT NULL,
  pgn      TEXT NOT NULL
);

CREATE TABLE game_openings (
  game_id        TEXT    PRIMARY KEY REFERENCES games(id) ON DELETE CASCADE,
  eco            TEXT    NOT NULL,
  name           TEXT    NOT NULL,
  last_book_seq  INTEGER NOT NULL
);

CREATE TABLE game_commentary (
  game_id    TEXT    NOT NULL REFERENCES games(id) ON DELETE CASCADE,
  seq        INTEGER NOT NULL,
  kind       TEXT    NOT NULL CHECK(kind IN ('rules','llm')),
  body       TEXT    NOT NULL,
  model      TEXT,                                  -- nullable; only for kind=llm
  created_at INTEGER NOT NULL,
  PRIMARY KEY (game_id, seq, kind)
);
```

Notes:
- `seq = 0` row in `move_analyses` is the eval of the starting position (so the eval bar has a value before any move).
- `engine_version` lets a future Stockfish bump invalidate cleanly without dropping old rows.
- `move_analyses` has 5 rows per ply (MultiPV); `move_classifications` has 1.

### Solo game persistence (prerequisite)

- Today `g.Batch` is nil for solo games (`game.go:357`). Extend `NewGame` to call `InitBatch` against a dedicated session row with `session_type='solo'` (add to enum in a small migration), or — minimally invasive — defer persistence until `signalGameOver` and write all rows in one transaction at the end.
- Recommendation: **persist on game-end only** for solo. Avoids touching the live write path, and analysis only needs the final state anyway.

---

## Opening book

- Source: Lichess [chess-openings](https://github.com/lichess-org/chess-openings) (CC0). TSVs `a.tsv`…`e.tsv`.
- Ingest script: `scripts/ingest-openings.go` parses TSV, replays PGN per line, computes Zobrist after each ply, inserts into `openings`. Run once at build; ship the populated rows via a migration data file or a separate sqlite attach.
- Lookup: after every move, `SELECT eco, name FROM openings WHERE zobrist = ?`. While hits continue, update `game_openings.last_book_seq`. First miss → frozen.
- During the live game (when analysis is off), opening detection is **still on** — it's cheap and not engine-derived, so it doesn't constitute analysis assistance for the player. Show in the game info header.

---

## Classification math

Convert centipawn score to win% (Lichess formula):

```
wp(cp) = 50 + 50 * (2 / (1 + exp(-0.00368208 * cp)) - 1)
```

Mate scores convert to 100 / 0. Clamp to [0, 100].

Per ply (after the move is played):
- `wp_before` = win% from mover's POV at the position **before** the move, using the **best** PV score.
- `wp_after_played` = win% from mover's POV at the position **after** the played move (= `100 - wp_opponent_best`).
- `wp_loss = max(0, wp_before - wp_after_played)`.

Base classification by `wp_loss`:

| `wp_loss` | Label |
|----------:|-------|
| < 2       | Best / Excellent |
| 2 – 5     | Good |
| 5 – 10    | Inaccuracy |
| 10 – 20   | Mistake |
| ≥ 20      | Blunder |

Modifiers requiring MultiPV context:

| Label | Condition |
|-------|-----------|
| Book | Position is in `openings` table |
| Forced | Only one legal move |
| Great | Played move's `wp_loss < 5` AND every other legal move's `wp_loss ≥ 10` (only safe move) |
| Brilliant | Great + the move sacrifices material (compare static piece values on `from`/`to` vs. PV outcome) |
| Miss | Played move ≤ Good (wp_loss ≥ 5) AND best line was a forced mate or wp_before ≥ 90 (winning advantage thrown) |

Tuning: hold these thresholds behind constants in `analysis/classify.go`. Calibrate on ~50 sample games before shipping.

### Accuracy %

Per side: `103.1668 * exp(-0.04354 * mean_wp_loss) - 3.1669`, clamped to [0, 100]. Displayed alongside the game result banner.

---

## SSE / HTTP API

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/:mode/:gameID/analysis` | Enqueue analysis. Idempotent (returns existing job if queued/running/done). 409 if game still ongoing AND policy says off. |
| `GET`  | `/:mode/:gameID/analysis` | Full result (job + per-ply rows + classifications + opening + accuracy). JSON. |
| `GET`  | `/:mode/:gameID/analysis/events` | SSE: streams `job_progress`, `ply_analysis`, `done`, `failed`. |
| `POST` | `/:mode/:gameID/analysis/commentary/:seq` | Request LLM commentary for one ply. Returns stored row or generates and stores. |
| `DELETE` | `/:mode/:gameID/analysis` | Cancel/delete (owner only). |

Per-ply SSE payload order (each is a separate event so UI fills progressively):

1. `opening` — instant from book lookup (only while in book).
2. `eval` — depth ~12, single-PV. Quick visual update for the eval bar.
3. `top_moves` — depth ~20, MultiPV=5. Replaces the eval reading.
4. `classification` — derived from `top_moves` of this position + the next position.
5. `rules_commentary` — feature-extractor sentence.
6. `llm_commentary` — only if user requested for this ply.

Authorization:
- Solo / vs-bot: session cookie must own the game.
- Play games: any participant (white/black/spectator) of the session may view post-game. Spectators may also view mid-game once they opt in.

---

## Live-game policy

State machine for whether analysis is exposed:

```
game status = ongoing
  ├── solo / vs-bot       → exposed to the owner (user setting: default on)
  ├── play, viewer = player    → NOT exposed (force off)
  └── play, viewer = spectator → exposed iff spectator opted in
game status != ongoing    → exposed to all participants & spectators
```

Implementation:
- `analysis.PolicyAllows(game, viewer) bool` consulted in both the HTTP handler and the SSE handler before opening the stream.
- Players see a placeholder card during their own game: "Analysis available when the game ends." Eval bar UI is hidden, not zeroed.
- Spectator opt-in lives in session cookie (`analysis_optin=1`), so it persists across reloads.

---

## Engine pool & performance budget

- Pool size: `min(NumCPU()/2, 4)` Stockfish processes. Each `Threads=1`, `Hash=64` MB.
- Per-ply budget: depth 20 OR 2000 ms wall, whichever first. Tuned per ply count — long games may drop to depth 18.
- Whole-game budget: stop and mark `done` after 5 minutes wall regardless of progress; partial results are kept.
- Live mode (when policy permits): only analyse the **current** position. New move arriving for the same game cancels the in-flight job for the prior FEN — only the latest matters. Implement as job dedupe keyed by `game_id`.
- Throttle: max 1 concurrent analysis per user (prevents one user from saturating the pool).
- Don't analyse and run the bot search simultaneously for the same user — bot opponent moves take priority; analysis resumes after.

---

## Rule-based features (`analysis/features/`)

Extract per position from the `engine.Board`:

| Feature | Source |
|---------|--------|
| Material balance | piece counts × piece values |
| Mobility delta | `GeneratePseudoLegalMoves` count per side |
| King safety | attackers on squares adjacent to own king; pawn shield intact |
| Pawn structure | isolated / doubled / passed counts per side |
| Open / semi-open files | rook on file with no own pawn |
| Bishop pair | bool per side |
| Space | rank advancement sum for pawns past 4th rank |
| Tempo | side to move |

Templated commentary picks the most salient diff and emits one sentence:

> "Black has the bishop pair and a passed pawn on d5; White's king is exposed on the kingside."

Bounded vocabulary, deterministic, no network. Always shown.

---

## LLM commentary

- Provider: Claude via `anthropic` SDK. Model configurable; default `claude-haiku-4-5` for cost (commentary is short).
- Triggered per-ply on user click; result cached in `game_commentary`.
- Prompt input: `{fen, last_move_san, eval_cp_or_mate, top_5_pv_san, opening_name, rules_summary, classification}`.
- System prompt constrains: ≤ 2 sentences, name concrete squares/pieces, no "as an AI", no engine-vs-human framing, factual claims must be visible in the PV.
- API key from env `ANTHROPIC_API_KEY`. If absent, the LLM button is hidden and the feature degrades silently to rules-only.
- Rate limit: per-user 30 commentary requests / 5 min.
- Prompt caching: cache the system prompt + opening explainers across requests.

---

## UI components

| Component | File | Notes |
|-----------|------|-------|
| Eval bar | `ui/components/evalbar.templ` | Vertical, left of board. Smooth animation. Mate shown as `M5`. Hidden when policy=off. |
| Top-5 panel | `ui/components/topmoves.templ` | List of SAN + score; hover draws arrow on board overlay. |
| Classification chip | inline in `movenotationpanel.templ` | Icon + tooltip. Colors match severity. |
| Opening header | inline in `gameinfopanel.templ` | "Sicilian Defense, Najdorf Variation (B90)". |
| Commentary panel | `ui/components/commentary.templ` | Collapsible. Rules summary always; "Explain with AI" button for LLM. |
| Accuracy badges | inline in `gameresultribbon.templ` | "White 89.2% · Black 76.1%". |
| Job progress | inline in result ribbon | Bar + percent while running. |

All driven by datastar signals so SSE updates patch without re-render.

---

## Phased rollout

| Phase | Scope | Done when |
|-------|-------|-----------|
| 0 | Solo persistence on game-end; migration 000003 | Completed solo games visible in `games` table |
| 1 | `engine/uci/` package + Stockfish bundling + process pool | Unit test: spawn, `Analyze`, parse 5 PVs |
| 2 | Opening ingest + `openings` table + lookup | First N plies of any game labeled with ECO |
| 3 | Worker queue, jobs table, single-PV eval per ply | Eval bar populates post-game on demand |
| 4 | MultiPV + classification + accuracy | Move chips render with correct labels on a hand-graded test game |
| 5 | SSE streaming endpoint + progressive UI fill | Eval/top-5/classification appear in order without UI jank |
| 6 | Rule-based feature extractor + templated commentary | One sentence per ply on demand |
| 7 | LLM commentary + caching + rate limit | "Explain" button works, falls back gracefully without API key |
| 8 | Live policy + spectator opt-in | Players see placeholder during own game; analysis unlocks at game end |
| 9 | Calibration pass — tune thresholds against ~50 graded games | Classification agrees with reviewer on ≥ 90% of plies |

## Open questions (defer to implementation)

- Should historical games (pre-feature) be backfilled in a one-shot job, or analysed lazily on first view?
- Where do we surface the "engine version changed, re-analyse?" affordance — automatic or manual?
- Do coaches in class sessions get analysis tools mid-class? (Probably yes — they're not playing.) If so, add `role='coach'` to the policy allow-list.
- Bot opponent strength: should the bot also use Stockfish (replacing in-tree search), or keep them separate? Affects engine pool sizing.
