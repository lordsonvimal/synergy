# ChessLeap — Build Plan

Source of truth for delivery order, architectural decisions, and per-slice scope. Supersedes the order described in `TODO.md` (now stale) and the phased order in `ROADMAP.md` §"Suggested Build Order".

`ROADMAP.md` remains the long-form requirements doc; this file is the execution plan.

---

## Business Model — Foundation Assumption

**Model: Freemium subscription + pay-per-class for live coaching.**

| Layer | Pricing | Includes |
|---|---|---|
| **Free** | $0 | Online play (rated + casual), basic puzzles (limited/day), profile, Elo |
| **Plus** | $9–12/mo | Game analysis (Stockfish), unlimited puzzles, full GM database, opening theory, class recordings |
| **Live classes** | Per-session, set by coach | Sold separately from Plus; platform takes commission |

**Why this model:** Subscription captures passive learners (predictable MRR). Per-class captures the high-willingness-to-pay parents/serious students paying for coaching. Coach (you) launches with captive student base — they already pay for classes; subscription is the *new* revenue line monetising self-study hours.

Tournament entry fees, coach marketplace commission, and Academy tier are deferred to growth phase.

---

## Roles vs Subscriptions — Architectural Separation

Two orthogonal axes. **Never collapse into one column.**

| Axis | Column | Values | Source of truth |
|---|---|---|---|
| **Role** (permissions) | `users.role` | `student \| coach \| arbiter \| admin` | Admin-assigned or subscription webhook |
| **Subscription tier** (access) | `users.subscription_tier` | `free \| plus` | Stripe |

**Bridge rules** (enforced in Go, not DB constraints):
- `arbiter` and `admin` are admin-assigned only — no subscription path
- `coach` role exists for the platform owner and (later) marketplace coaches; not granted by Plus subscription
- Plus tier unlocks **features**, not roles

Roles form a hierarchy *only* for admin (admin can do anything). `coach` and `arbiter` are siblings with disjoint capabilities.

### Role grant paths
1. **Bootstrap admin** — first login matching `BOOTSTRAP_ADMIN_EMAIL` env auto-promoted to `admin`
2. **Admin UI** — `/admin/users` page, admin-only, dropdown to change role, writes `role_audit` row
3. **CLI** — `chess admin promote <email> <role>` for scripting / emergencies
4. **Stripe webhook (future)** — reuses the same `PromoteUser()` function when coach-subscription tier lands

Demotion is symmetric and audited.

---

## Build Order

| # | Slice | Duration | First user-visible win |
|---|---|---|---|
| 1 | Auth + Profile (Google OAuth, roles, Plus column) | ~2 weeks | "Sign in with Google" + profile page |
| 2 | Identity-aware multiplayer + Elo (K-factor) | 2–3 weeks | Rated games show on profile |
| 3 | Stockfish WASM analysis + paywall scaffold | 1–2 weeks | First Plus-gated feature |
| 4 | Stripe Plus subscription | 1–2 weeks | First real revenue |
| 5 | Live Classes MVP | 4–6 weeks | Coach product launches |
| 6 | Learning modes (puzzles, GM DB, openings) | 4–6 weeks | Engagement / retention loop |
| 7 | Tournaments, coach marketplace, growth features | TBD | Growth phase |

**Total to first revenue (Slices 1–4):** ~6–9 weeks.

Slices 5–7 stay as outlines until Slice 4 is landing — re-plan in detail against real product learnings.

---

## Slice 1 — Auth + Profile

### Scope
Google OAuth login, user profile, role + subscription_tier columns (no Stripe yet), shared Go auth package, admin user management. Existing anonymous `/play` flow continues to work as "Guest".

### 1.1 New Go module: `packages/auth-go/`

Add to `go.work`. Module path: `github.com/lordsonvimal/synergy/packages/auth-go`.

```
packages/auth-go/
  oauth/
    google.go          # provider config, code exchange, ID token verification
    provider.go        # Provider interface (future: github, microsoft)
  session/
    jwt.go             # signed cookie session (extends pattern from server/session.go)
    middleware.go      # gin middleware: CurrentUser(c) helper
  user/
    repository.go      # UserRepository interface — apps implement
    model.go           # User, Role, SubscriptionTier types
  role/
    middleware.go      # RequireRole(...), CanCoach(), CanArbitrate()
  config/
    config.go          # OAuthConfig (clientID, secret, redirectURLs, sessionSecret)
  README.md
```

**Boundary rule:** the package owns *types, interfaces, OAuth flow, JWT signing, middleware*. Apps own *DB schema + UserRepository implementation*. Keeps the package DB-agnostic so other apps in the monorepo (`core`, `sis`) can adopt it with Postgres if they want.

### 1.2 Chess app changes

**New migration `000003_users.up.sql`:**
```sql
CREATE TABLE users (
  id TEXT PRIMARY KEY,
  email TEXT NOT NULL UNIQUE,
  display_name TEXT NOT NULL,
  avatar_url TEXT,
  google_sub TEXT UNIQUE,
  role TEXT NOT NULL DEFAULT 'student'
    CHECK (role IN ('student','coach','arbiter','admin')),
  subscription_tier TEXT NOT NULL DEFAULT 'free'
    CHECK (subscription_tier IN ('free','plus')),
  role_granted_at INTEGER,
  role_granted_by TEXT REFERENCES users(id),
  elo_blitz INTEGER NOT NULL DEFAULT 1200,
  elo_rapid INTEGER NOT NULL DEFAULT 1200,
  elo_classical INTEGER NOT NULL DEFAULT 1200,
  games_played INTEGER NOT NULL DEFAULT 0,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);
CREATE INDEX idx_users_google_sub ON users(google_sub);

CREATE TABLE role_audit (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  user_id TEXT NOT NULL REFERENCES users(id),
  old_role TEXT,
  new_role TEXT NOT NULL,
  granted_by TEXT REFERENCES users(id),
  reason TEXT,
  created_at INTEGER NOT NULL
);
```

**New files:**
- `db/sqlite/user.go` — implements `auth.UserRepository`
- `server/auth_handlers.go` — `/auth/google`, `/auth/google/callback`, `/auth/logout`
- `server/admin_handlers.go` — `/admin/users` (list + promote/demote)
- `server/middleware_auth.go` — wires `CurrentUser` into request context
- `scripts/admin_cli.go` (or extend `main.go` with subcommand) — `chess admin promote <email> <role>`

**Modified files:**
- `server/routes.go` — add auth routes; wrap admin routes with `RequireRole("admin")`
- `server/session.go` — extend `PlaySessionClaims` to optionally carry `UserID` (logged-in seat); guest path still works with empty UserID
- `server/play_handlers.go` — when claiming a seat, prefer `CurrentUser` over guest token
- `config/env.go` — add `GOOGLE_OAUTH_CLIENT_ID`, `GOOGLE_OAUTH_SECRET`, `OAUTH_REDIRECT_BASE`, `BOOTSTRAP_ADMIN_EMAIL`
- `main.go` — wire auth package, register OAuth provider, run bootstrap admin check on startup

### 1.3 UI

- **Header** (all pages): "Sign in with Google" when logged out; avatar + dropdown (Profile, Admin if admin, Logout) when logged in
- **Profile page `/me`**: display name, avatar, email, role badge, tier badge ("Free" / "Plus"), Elo per time control, games played
- **Admin page `/admin/users`**: table of users, search by email, role dropdown per row, "Promote/Demote" action with audit reason modal
- **Play seat label**: show display name + Elo for logged-in seats; "Guest (White)" for anonymous

### 1.4 OAuth setup checklist (operator task, not code)
1. Google Cloud Console → OAuth Consent Screen → External → app name, support email
2. Credentials → Create OAuth Client ID → Web application
3. Authorized redirect URIs: `http://localhost:8080/auth/google/callback` and `https://<prod-domain>/auth/google/callback`
4. Copy Client ID + Secret into local `.env` (never commit)
5. Add domain to consent screen "Authorized domains"

### 1.5 Testing
- **Unit**: `auth-go/oauth` mocks Google token endpoint; verifies ID token claim extraction
- **Unit**: role middleware allows/denies per role matrix
- **Integration**: full login flow with `httptest` server emulating Google
- **Manual**: real Google login on localhost

### 1.6 Deliverable order (within the slice)
1. `packages/auth-go` skeleton — types + interfaces — 1 day
2. Google OAuth flow + session cookie + middleware — 2 days
3. Chess SQLite `UserRepository` + migration — 1 day
4. Login/logout UI + header avatar — 1 day
5. Profile page — 1 day
6. Admin user page + role audit — 2 days
7. Bootstrap admin + CLI promote command — 0.5 day
8. Wire `CurrentUser` into `/play` seat claim (guest preserved) — 1 day
9. Tests + `packages/auth-go/README.md` + `.env.example` updates — 1.5 days

### 1.7 Non-goals
- No email/password login (Google-only)
- No password reset, email verification
- No Stripe / subscription writes (column exists, always `'free'`)
- No Elo calculation (columns exist, never updated)
- No coach signup or invite flow
- No avatar upload (use Google's avatar URL)

---

## Slice 2 — Identity-Aware Multiplayer + Elo

### Scope
Bind `/play` seats to logged-in users. Rated games for logged-in vs logged-in matches. Arbiter-controlled rated game rooms.

### Dependencies
- Slice 1 — `users.role`, `elo_*` columns, `CurrentUser` middleware

### Schema additions
- `games.white_user_id`, `games.black_user_id` (nullable for guest games)
- `games.rated` boolean — true only when both seats are logged-in users
- `games.time_control_category` — `blitz | rapid | classical` (derived from initial time)
- `game_results` table — final result + rating delta for rated games (audit trail)

### Rating system
**Vanilla Elo with K-factor.** Decision locked in (simpler, FIDE-aligned). Defaults:
- K = 40 during provisional period (first 20 games per category)
- K = 20 after provisional
- Separate rating per category: blitz / rapid / classical
- Guest-involved games never rated

Revisit if rating drift complaints come from infrequent players → migrate to Glicko-2.

### New surface
- Arbiter room: `/arbiter/games/new` — pick two users by email, time control, "Start"
- Profile page gains rating graph + recent rated games table
- Game-end handler computes Elo delta, writes `game_results`, updates user `elo_*` and `games_played`

### Open decisions for this slice
- Anti-abuse: same-user-both-sides lockout, IP-based collusion detection — defer or now?
- Provisional games visible with `?` marker on rating

### Non-goals
- Matchmaking queue / seek lobby (deferred to tournaments)
- Reconnection beyond what `/play` already handles
- Spectator mode beyond what exists

---

## Slice 3 — Stockfish WASM Analysis + Paywall Scaffold

### Scope
Post-game analysis page powered by Stockfish in a Web Worker. Logged-in Plus users only — first feature behind the paywall (paywall returns "Upgrade to Plus — Coming Soon" until Slice 4 wires Stripe).

### Dependencies
- Slice 1 — `subscription_tier` gate
- *Independent of Slice 2* — can ship in parallel

### Architecture decision: Stockfish WASM in browser
- Zero server compute cost
- Matches CLAUDE.md local-first architecture rule and $0–10/mo infra target
- ~10MB one-time download per user
- Weaker on mobile — acceptable trade for now

### New surface
- `/analysis/:gameID` — loads PGN from `moves` table, runs Stockfish in Web Worker
- Eval bar, best-move arrows, blunder/mistake/inaccuracy annotations
- `game_analysis` table — caches depth + eval per ply so re-opens are instant
- Paywall middleware `RequireTier("plus")` — returns "Upgrade to Plus" page for free users

### Non-goals
- Opening book lookup
- Endgame tablebase
- Cloud eval sharing between users

---

## Slice 4 — Stripe Plus Subscription

### Scope
Wire real Stripe state behind the paywall middleware shipped in Slice 3. Single SKU: Plus subscription ($9–12/mo, final price TBD before launch).

### Dependencies
- Slice 3 — paywall middleware exists; this slice flips it from "Coming Soon" to a real upgrade flow

### Architecture decisions
- **Stripe Customer Portal** (hosted) over custom upgrade/cancel UI — saves weeks, looks professional, handles dunning
- **7-day free trial** — standard, low fraud risk
- Annual discount deferred to v2

### New surface
- `subscriptions` table: `stripe_customer_id`, `stripe_subscription_id`, `status`, `current_period_end`
- `/billing` page: current plan + manage button → Stripe Customer Portal
- `POST /billing/checkout` → Stripe Checkout Session
- `POST /webhooks/stripe` → handles `customer.subscription.{created,updated,deleted}`; updates `users.subscription_tier`
- Stripe CLI for local webhook testing

### Non-goals
- Coach payouts, per-class fees, tournament fees (Slices 5+)
- Custom invoicing, tax compliance beyond Stripe defaults
- Refund UI

---

## Slice 5 — Live Classes MVP (planning deferred)

### Scope (high-level — re-plan when Slice 4 lands)
Coach creates a class link, students join, shared board + LiveKit video. Existing Phase 2 from `ROADMAP.md` §5, with auth integrated from day one.

### Dependencies
- Slice 1 — `coach` role for class creation
- Slice 4 — *if* paid classes ship in MVP (open decision)

### Key decisions to make at planning time
1. Free vs paid classes on launch (recommend: free first, paid in 5.5)
2. Recording: opt-in per session — storage cost is the unknown
3. Max class size (recommend: 20 for MVP)
4. Drill bank: how many of the 7 drill types to ship in v1

### Architectural anchors (already decided in `ROADMAP.md` §5.1)
- LiveKit Cloud free tier → self-hosted LiveKit Server on scale
- Go server issues JWTs + calls RoomService API; browser ↔ LiveKit direct for media

---

## Slice 6 — Learning Modes (planning deferred)

### Scope (high-level)
Tactics puzzles, GM game database with annotations, opening theory tree. Per `ROADMAP.md` §4.

### Dependencies
- Slice 1 — puzzle rating per user, spaced-repetition state
- Slice 4 — Plus gate on unlimited puzzles / full GM database

### Key decisions to make at planning time
1. Puzzle DB source: Lichess open puzzle DB import vs manual curation vs both
2. Spaced repetition: SM-2 vs FSRS algorithm
3. Opening tree authoring workflow: PGN import vs in-app editor

---

## Slice 7 — Growth (TBD)

Tournaments (Swiss/round-robin/knockout), coach marketplace, achievement/streak system, parent dashboard, certificates, white-label, daily puzzle. Per `ROADMAP.md` §6–8 and "Additional Revenue-Generating Features". Plan when Slices 5–6 are landing.

---

## Open Cross-Cutting Items
- **Litestream** — continuous SQLite replication to S3-compatible storage; carry-over from `TODO.md` §4. Add when Slice 4 ships (data becomes commercially valuable).
- **Email sender** — needed for transactional email (login alerts, receipts, class reminders). Defer until needed; Resend/Postmark free tier when picked.
- **Domain + production deploy** — needed before Slice 1 OAuth review with Google. Hetzner CX22 + Cloudflare per `ROADMAP.md` deployment plan.
