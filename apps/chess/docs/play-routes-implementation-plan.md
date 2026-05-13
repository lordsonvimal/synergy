# Plan: Two-Player Online Chess (`/play/*` routes)

## Context

The chess app currently supports a single-player / coach-controlled flow under `/game/*`. This plan adds a parallel two-player online flow under `/play/*` where two people can play against each other using invite links with session-based role claiming.

---

## Design Decisions (confirmed via grill-me)

| Decision | Choice |
|---|---|
| Session cookie | Signed `{game_id, role}` (HMAC-SHA256), stateless |
| Token URL format | Query param: `/play/:gameID?token=abc` |
| Post-creation redirect | `POST /play` → `GET /play/:gameID` (creator's cookie set) |
| Clock unlock | White's first move, after both players have active SSE connections |
| First-move timeout | 30s after both online → abandon |
| Disconnect behavior | Clock keeps running; after 60s, opponent sees "Claim Victory" button |
| Spectator orientation | White's perspective + flip button |
| Claim victory | Manual — opponent clicks button (server shows after 60s of disconnect) |

---

## New Routes

```
GET  /play                          → CreatePlayForm
POST /play                          → CreatePlay
GET  /play/:gameID                  → ShowPlayGame
GET  /play/:gameID/events           → PlayEventsHandler (SSE)
POST /play/:gameID/select/:square   → PlaySelectSquare
POST /play/:gameID/claim-victory    → ClaimVictory
```

---

## Implementation Steps

### Step 1: DB — Add `UpdateJoinedAt` to `ParticipantStore`

**`db/repository.go`** — extend interface:
```go
type ParticipantStore interface {
    Create(ctx context.Context, p *Participant) error
    GetByToken(ctx context.Context, token string) (*Participant, error)
    ListBySession(ctx context.Context, sessionID string) ([]*Participant, error)
    UpdateJoinedAt(ctx context.Context, id string, joinedAt int64) error  // NEW
}
```

**`db/sqlite/participant.go`** — implement:
```go
func (s *participantStore) UpdateJoinedAt(ctx context.Context, id string, joinedAt int64) error {
    _, err := s.db.ExecContext(ctx, `UPDATE participants SET joined_at = ? WHERE id = ?`, joinedAt, id)
    return err
}
```

`UpdateJoinedAt` is called when a player first claims their token. Participants are created at game creation with `joined_at = created_at` (creation placeholder). After claiming, `joined_at` is updated to the actual claim timestamp. On DB restore, a participant with `joined_at > game.created_at + 1 second` is treated as already claimed.

---

### Step 2: Session Cookie Signing

**New file: `server/session.go`**

```go
type PlaySessionClaims struct {
    GameID string `json:"game_id"`
    Role   string `json:"role"` // "white" | "black" | "spectator"
}

// Cookie value: base64url(JSON).HMAC-SHA256(base64url(JSON))
// Signing key: SESSION_SECRET env var (fallback "dev-insecure-secret")

func SignSession(claims PlaySessionClaims) (string, error) { ... }
func VerifySession(cookie string) (PlaySessionClaims, error) { ... }
func SetPlaySession(c *gin.Context, claims PlaySessionClaims) error { ... }
func GetPlaySession(r *http.Request) (PlaySessionClaims, error) { ... }
```

Cookie name: `play_session`, HttpOnly, SameSite=Lax, 7-day MaxAge.

**`config/env.go`** — add helper:
```go
func SessionSecret() []byte {
    return []byte(GetEnv("SESSION_SECRET", "dev-insecure-secret-change-me"))
}
```

---

### Step 3: Game Layer — `PlayMeta` and `NewPlayGame`

**New file: `game/play_meta.go`**

```go
type PlayMeta struct {
    mu sync.Mutex

    // Set at creation; never change
    WhiteParticipantID string
    BlackParticipantID string
    WhiteToken         string
    BlackToken         string
    SessionID          string

    // Token claiming (in-memory; resets on server restart but cookies survive)
    WhiteJoined bool
    BlackJoined bool

    // Online tracking
    whiteConns        int        // active SSE connections for white
    blackConns        int
    whiteDisconnectAt *time.Time // nil = online
    blackDisconnectAt *time.Time

    // Clock-unlock gate
    BothPlayersConnectedOnce bool
    FirstMoveDeadline        *time.Time // 30s deadline for white's first move

    // Claim victory state
    ClaimVictoryFor engine.Color // disconnected side; NoColor = nobody
}

func (m *PlayMeta) RecordSSEConnect(role string) (bothNowConnected bool) { ... }
func (m *PlayMeta) RecordSSEDisconnect(role string) { ... }
func (m *PlayMeta) OnlineStatus() (white, black bool) { ... }
func (m *PlayMeta) CheckDisconnectExpiry(now time.Time) engine.Color { ... } // returns disconnected side after 60s
```

**`game/game.go`** — add field and constructor:
```go
type Game struct {
    // ... existing fields unchanged ...
    PlayMeta *PlayMeta // nil for /game/* flow
}

// NewPlayGame creates a play-mode game. Clock is NOT started at creation.
func NewPlayGame(mode *GameMode, meta *PlayMeta) *Game {
    // Same as NewGame() but:
    // - Does NOT call gc.Start(engine.White)
    // - Sets g.PlayMeta = meta
}
```

`NewGame()` and `NewRestoredGame()` remain unchanged.

---

### Step 4: New Event Types

**`game/events.go`** — add (existing types unchanged):

```go
type OnlineStatusEvent struct {
    Type        string `json:"type"`         // "online_status"
    WhiteOnline bool   `json:"white_online"`
    BlackOnline bool   `json:"black_online"`
}

type ClaimVictoryEvent struct {
    Type            string `json:"type"`             // "claim_victory_available"
    DisconnectedFor string `json:"disconnected_for"` // "white" | "black"
}

type ClockUnlockedEvent struct {
    Type string `json:"type"` // "clock_unlocked"
}
```

---

### Step 5: Watchdog — Play-Mode Gate

**`game/watchdog.go`** — additive change inside the watchdog goroutine, after the `g.mu.Lock()` at the top of the loop:

```go
// Inserted after "if g.State != GameOngoing { ... return }"
if g.PlayMeta != nil {
    if !g.PlayMeta.BothPlayersConnectedOnce {
        g.mu.Unlock()
        continue // clock not yet unlocked; skip tick
    }

    // First-move timeout (30s)
    if g.PlayMeta.FirstMoveDeadline != nil &&
        now.After(*g.PlayMeta.FirstMoveDeadline) &&
        len(g.History) == 0 {
        g.State = GameAbandoned
        g.signalGameOver()
        g.mu.Unlock()
        return
    }

    // Disconnect expiry (60s) → enable claim-victory button
    disconnectedSide := g.PlayMeta.CheckDisconnectExpiry(now)
    if disconnectedSide != engine.NoColor && g.PlayMeta.ClaimVictoryFor == engine.NoColor {
        g.PlayMeta.ClaimVictoryFor = disconnectedSide
        evt := ClaimVictoryEvent{Type: "claim_victory_available", DisconnectedFor: colorStr(disconnectedSide)}
        go g.broadcastJSON(evt)
    }
}
// ... existing flag-fall check and 1Hz tick ...
```

No changes to the existing logic paths; all modifications are additive inside the `if g.PlayMeta != nil` guard.

---

### Step 6: Play Handlers

**New file: `server/play_handlers.go`**

#### `ShowCreatePlay` — `GET /play`
Renders the creation form with `game.ListGameModes()` + color preference radio (White / Black / Random).

#### `CreatePlay` — `POST /play`
1. Parse `mode` and `color` form fields
2. Resolve creator's color (Random → `rand.Intn(2)`)
3. Generate `whiteToken`, `blackToken` via `crypto/rand`
4. Create DB rows (session, 2 participants, game, `"game_created"` event) via `persistPlayGame`
5. Call `game.NewPlayGame(...)` + `repo.Add(g)`
6. Set `play_session` cookie for creator (their color)
7. `c.Redirect(303, "/play/"+g.ID)`

**`persistPlayGame`** private function (similar to `persistGame`):
- Creates `db.Session{SessionType: "play", Status: "pending"}`
- Creates 2 `db.Participant` rows with `role: "white"/"black"`, `Token: whiteToken/blackToken`, `JoinedAt: now`
- Creates `db.Game` with `WhiteParticipantID`, `BlackParticipantID` set
- Calls `g.InitBatch(...)` reusing the exact same batch flush/end pattern from `persistGame`

#### `ShowPlayGame` — `GET /play/:gameID`
Role resolution (in order):
1. Check `play_session` cookie → if valid and `game_id` matches → use cookie role
2. Else check `?token=` query param:
   - Match against `g.PlayMeta.WhiteToken` or `BlackToken`
   - If unclaimed (`!g.PlayMeta.WhiteJoined` or `!g.PlayMeta.BlackJoined`): claim (set `Joined = true`, call `UpdateJoinedAt`, set cookie), redirect to `/play/:gameID` (strips token from URL)
   - If already claimed: role = "spectator"
3. No valid cookie, no valid token → "spectator"

Renders `pages.PlayGamePage(g, role, flipped, csrfToken)`.

#### `PlayEventsHandler` — `GET /play/:gameID/events`
Same SSE loop as `GameEventsHandler`, extended:
- On connect (if role is "white"/"black"):
  - `PlayMeta.RecordSSEConnect(role)` → if `bothNowConnected` for first time:
    - Lock `g.mu`, set `PlayMeta.BothPlayersConnectedOnce = true`, set `FirstMoveDeadline = now+30s`, call `g.Clock.Start(engine.White)`, unlock
    - Broadcast `ClockUnlockedEvent`
    - Optionally update session status to "active" in DB
  - Broadcast `OnlineStatusEvent`
- On disconnect (defer): `PlayMeta.RecordSSEDisconnect(role)`, broadcast `OnlineStatusEvent`

#### `PlaySelectSquare` — `POST /play/:gameID/select/:square`
Before the existing square-selection logic, add auth guards:
1. Verify `play_session` cookie for this game
2. Reject if it's not the cookie holder's turn (`g.Board.SideToMove != playerColor`)
3. Reject if `!g.PlayMeta.BothPlayersConnectedOnce` (game not started)
4. Then: identical logic to `SelectSquare` (extract into shared `applySquareSelection(c, g)` helper, called from both handlers to avoid duplication)

#### `ClaimVictory` — `POST /play/:gameID/claim-victory`
1. Verify cookie → must be "white" or "black"
2. Verify `g.PlayMeta.ClaimVictoryFor` is the opponent's color
3. Lock `g.mu`, set `g.Winner`, `g.State = GameAbandoned`, call `g.signalGameOver()`
4. Return `200 OK`

---

### Step 7: Route Registration

**`server/routes.go`** — add after existing routes:
```go
r.GET("/play", ShowCreatePlay)
r.POST("/play", CreatePlay)
r.GET("/play/:gameID", ShowPlayGame)
r.GET("/play/:gameID/events", PlayEventsHandler)
r.POST("/play/:gameID/select/:square", PlaySelectSquare)
r.POST("/play/:gameID/claim-victory", ClaimVictory)
```

---

### Step 8: Templates

**New files:**

- **`ui/pages/createplay.templ`** — Creation form: mode cards + color preference radio (White/Black/Random) + CSRF + submit. Reuses same card styling as `gamemodes.templ`.

- **`ui/pages/playgame.templ`** — Play game page. Wraps existing board + panel components. Adds:
  - Share links panel (visible until both joined)
  - Online status dots in player cards
  - DataStar signals include `role`, `flipped`, `whiteOnline`, `blackOnline`, `claimVictory`, `clockUnlocked`

- **`ui/components/sharelinks.templ`** — Shows white link + black link with copy buttons. Hidden after `ClockUnlockedEvent` via `data-show="!$clockUnlocked"`.

- **`ui/components/playboard.templ`** — Parameterized board where square clicks post to `/play/:gameID/select/:square`. Reuse existing square rendering logic by extracting a `postPrefix` parameter, or simply copy + adjust the POST URL. Board container gets `data-class="{'rotate-180': $flipped}"` for black's orientation.

- **`ui/components/playplayercards.templ`** — Player cards with online status dot: `data-class="{'bg-green-500': $whiteOnline, 'bg-gray-400': !$whiteOnline}"`.

- **`ui/components/playtoolspanel.templ`** — Includes:
  - Flip button (always for spectators; hidden for players)
  - Claim victory form: `data-show="$claimVictory"`, posts to `/play/:gameID/claim-victory`

**Reused without change:** `MoveNotationPanel`, `RenderPromotionOverlay`, `layouts.Layout`, all existing components.

---

### Step 9: JS Client Extension

**`assets/js/initPlayClock.js`** — extends `initClock.js` event handling with new event types:

```js
case "online_status":
    ctx.$patch({ whiteOnline: e.white_online, blackOnline: e.black_online })
    break
case "clock_unlocked":
    ctx.$patch({ clockUnlocked: true })
    break
case "claim_victory_available":
    ctx.$patch({ claimVictory: true })
    break
```

The play game page includes `initPlayClock.js` instead of `initClock.js`.

---

## DB Restore for Play Games

When loading a play game from DB (server restart scenario):
1. Load game row → `g.PlayMeta` is nil initially
2. Load session's participants via `dbRepo.Participants().ListBySession(ctx, sessionID)`
3. Reconstruct `PlayMeta` with white/black tokens from participant rows
4. Set `WhiteJoined = participant.JoinedAt > game.CreatedAt + 1e9` (1 second threshold)
5. `BothPlayersConnectedOnce = false` (both must reconnect; clock was paused anyway since game is restored)

---

## Files Modified

| File | Change |
|---|---|
| `db/repository.go` | Add `UpdateJoinedAt` to `ParticipantStore` |
| `db/sqlite/participant.go` | Implement `UpdateJoinedAt` |
| `config/env.go` | Add `SessionSecret()` |
| `game/game.go` | Add `PlayMeta *PlayMeta` field; add `NewPlayGame` |
| `game/events.go` | Add 3 new event types |
| `game/watchdog.go` | Add play-mode gate (additive only) |
| `server/routes.go` | Register 6 new `/play/*` routes |

## Files Created

| File | Purpose |
|---|---|
| `game/play_meta.go` | `PlayMeta` struct + online tracking methods |
| `server/session.go` | Cookie sign/verify (`PlaySessionClaims`) |
| `server/play_handlers.go` | All 6 `/play/*` handlers + `persistPlayGame` |
| `ui/pages/createplay.templ` | Game creation form |
| `ui/pages/playgame.templ` | Play game page |
| `ui/components/sharelinks.templ` | Share links panel |
| `ui/components/playboard.templ` | Board with play-mode POST URLs + flip support |
| `ui/components/playplayercards.templ` | Player cards with online dots |
| `ui/components/playtoolspanel.templ` | Tools panel with claim victory button |
| `assets/js/initPlayClock.js` | SSE event handler for play-mode events |

## No Migration Needed

The existing `participants` table already has all required fields. `joined_at` doubles as a claim timestamp (creation placeholder vs. actual claim time, distinguished by the 1-second threshold).

---

## Verification

1. `go build ./...` — no compile errors
2. `templ generate` — all `.templ` files compile
3. Manual flow:
   - `GET /play` → form renders with modes + color radio
   - `POST /play` → creates game, redirects to `/play/:gameID`, shows share panel
   - Open white link in browser A, black link in browser B → each claims role; cookie set
   - Both SSE connect → `clock_unlocked` event hides share panel, online dots go green
   - White makes a move within 30s → clock starts normally
   - Close browser B tab → black dot goes grey after ~2s SSE timeout
   - After 60s → "Claim Victory" button appears for white
   - White clicks → game ends as abandoned
4. Revisit `/play/:gameID` in browser A (no token in URL) → session cookie restores white role
5. Open white link in browser C (token already claimed) → assigned spectator, sees flip button
