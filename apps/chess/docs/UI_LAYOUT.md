# ChessLeap — UI/UX Layout Specification

## Design Principles

- **Board is always the hero.** The chess board is the primary content. Every layout decision defends its space.
- **Bottom strip is the spine.** A full-width bottom strip is the sole persistent chrome below the board. It is identical in structure on desktop and mobile — panels open upward from it.
- **Panels are independent.** The video panel and chat panel open and close independently. The board fills whatever space remains.
- **Role-rendered, not role-hidden.** Coach tools (mode tabs, board tools, recording controls, participant actions) are never sent to the student's browser. Students receive a structurally different board area.
- **Global vs contextual.** The top bar contains only application-level navigation. All session-specific context lives inside the three panels.

---

## Page Structure

```
┌──────────────────────────────────────────────────────────────────────┐
│ GLOBAL TOP BAR  (application navigation only)                        │
├─────────────┬──────────────────────────────────────┬─────────────────┤
│             │                                      │                 │
│   VIDEO     │           BOARD AREA                 │   CHAT PANEL    │
│   PANEL     │        (always present,              │                 │
│  (optional) │         expands to fill)             │   (optional)    │
│             │                                      │                 │
├─────────────┼──────────────────────────────────────┼─────────────────┤
│ [👥 ctrl]   │   🎤   📹   ✋   [Leave / End]        │  [💬 ctrl]      │
└─────────────┴──────────────────────────────────────┴─────────────────┘
     ~240px               flexible                        ~280px
```

---

## Global Top Bar

Contains only application-level elements. Identical on every page and for every role.

```
┌──────────────────────────────────────────────────────────────────────┐
│ ◆ ChessLeap                                          [⚙]      [👤]  │
└──────────────────────────────────────────────────────────────────────┘
```

| Element | Purpose |
|---------|---------|
| ◆ ChessLeap | App logo + home link |
| [⚙] | Settings menu |
| [👤] | User / profile menu |

No class title, no recording indicator, no participant count, no mode name — none of these belong in global navigation.

---

## Bottom Strip

A full-width persistent bar at the bottom of the viewport. Three sections mirror the three panels above them. Section widths stay aligned with their corresponding panel.

```
├─────────────┼──────────────────────────────────────┼─────────────────┤
│  [👥] ctrl  │   🎤   📹   ✋   [Leave / End]        │  [💬] ctrl      │
└─────────────┴──────────────────────────────────────┴─────────────────┘
  aligns with       aligns with board area              aligns with
  video panel                                           chat panel
```

### Strip Controls

| Control | Coach | Student |
|---------|-------|---------|
| [👥] | Toggle video panel open/close; shows participant count | Same |
| 🎤 | Mute / unmute own microphone | Same |
| 📹 | Enable / disable own camera | Same |
| ✋ | View and manage raise-hand queue | Raise / lower own hand |
| [Leave / End] | "End Class" with confirmation | "Leave" with confirmation |
| [💬] | Toggle chat panel open/close; shows unread badge | Same |

**Active state:** Filled icon (●) when the panel is open. Dimmed icon when closed.

### Strip States

```
Both panels open
├─────────────┼──────────────────────────────────────┼─────────────────┤
│  [👥●] 12   │     🎤   📹   ✋   [End Class]        │   [💬●] +3      │
└─────────────┴──────────────────────────────────────┴─────────────────┘

Video only
├─────────────┬──────────────────────────────────────────────────────  ┤
│  [👥●] 12   │     🎤   📹   ✋   [End Class]              [💬] +3     │
└─────────────┴─────────────────────────────────────────────────────  ┘

Chat only
├──────────────────────────────────────────────────┬──────────────────  ┤
│  [👥] 12    🎤   📹   ✋   [End Class]            │   [💬●] +3        │
└──────────────────────────────────────────────────┴──────────────────  ┘

Both closed (maximum board)
├──────────────────────────────────────────────────────────────────────┤
│  [👥] 12    🎤   📹   ✋   [End Class]                     [💬] +3   │
└──────────────────────────────────────────────────────────────────────┘
```

---

## Video Panel (Left, Optional)

Slides in from the left. The board area shrinks to accommodate it. Width ~240px.

```
┌─────────────┐
│ 👥 12 [⊞▼] │  ← participant count + video layout toggle
│ ─────────── │
│ ┌─────────┐ │
│ │  Coach  │ │
│ │ [video] │ │
│ └─────────┘ │
│ ┌────┐┌───┐ │
│ │ S1 ││S2 │ │  ← tiles scroll vertically
│ └────┘└───┘ │
│ ┌────┐┌───┐ │
│ │ S3 ││+8 │ │  ← "+N more" tile opens full grid
│ └────┘└───┘ │
└─────────────┘
```

### Video Tile States
- Speaking: animated border ring
- Muted: mic icon shown on tile
- Camera off: avatar / initials fallback
- Poor connection: quality dot (green / yellow / red) in tile corner
- Hand raised: ✋ icon overlaid on tile

### Tile Interactions by Role

| Action | Coach | Student |
|--------|-------|---------|
| Hover | Mute · Spotlight · Remove · Grant board control | Pin |
| Click | Expand to larger overlay | Pin |

### Video Layout Toggle [⊞▼]
Accessible from the video panel header. Options:

| Layout | Behaviour |
|--------|-----------|
| Strip (default) | Vertical tile list in the left panel |
| Speaker View | Active speaker large (70% panel height), others as thumbnails below |
| Grid View | Equal-sized tiles, up to 20, reflow on join/leave |
| Broadcast | Only coach video shown; students' streams transmitted but not rendered |

---

## Chat Panel (Right, Optional)

Slides in from the right. The board area shrinks to accommodate it. Width ~280px.

```
┌─────────────────┐
│ Chat            │
│ ─────────────── │
│ Coach  10:41    │
│ "Watch Nf6→"    │
│                 │
│ Arun  10:42     │
│ "Got it!"       │
│                 │
│ ─────────────── │
│ ✋ Arun · Priya │  ← raise hand queue (coach sees dismiss action)
│                 │
│ 👍 😕 👏 😮    │  ← emoji reactions
│ ┌─────────────┐ │
│ │ message…    │ │
│ └─────────────┘ │
└─────────────────┘
```

### Chat Behaviour by Role

| Feature | Coach | Student |
|---------|-------|---------|
| Send messages | ✅ | ✅ |
| Clear all chat | ✅ | — |
| Mute a student's chat | ✅ | — |
| Raise hand queue | View + dismiss per student | — |
| Raise hand button | — | ✅ (also in strip) |
| Emoji reactions | ✅ | ✅ |

Reactions appear as a floating animation over the sender's video tile, fade after 3 seconds, and do not interrupt audio.

---

## Board Area

Always present. Fills all horizontal space not occupied by open panels. Contains all class-session context that is not global navigation.

### Board Area — Coach View

```
┌──────────────────────────────────────────────────────────────────────┐
│ Sicilian · Ruy Lopez, Morphy Defence                        [●REC]  │  ← class info row
│ ─────────────────────────────────────────────────────────────────── │
│ Teaching │ Training Game │ Position Drill │ Sandbox                  │  ← mode tabs (coach only)
│ ─────────────────────────────────────────────────────────────────── │
│                                                                      │
│                        CHESS BOARD                                   │
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │ 1.e4 e5  2.Nf3 Nc6  3.Bb5 a6  4.Ba4 Nf6  5.O-O Be7 …       │    │
│  └──────────────────────────────────────────────── [◀][▶][⊙]  ┘    │
│  [Flip] [FEN] [↩ Takeback] [● Colour▼] [Engine▼] [✕ Clear]         │  ← coach tools
└──────────────────────────────────────────────────────────────────────┘
```

### Board Area — Student View

Structurally different HTML. Mode tabs and all board tools are absent.

```
┌──────────────────────────────────────────────────────────────────────┐
│ Sicilian · Ruy Lopez, Morphy Defence                        [●REC]  │  ← same info, REC read-only
│ ─────────────────────────────────────────────────────────────────── │
│                                                                      │
│                        CHESS BOARD                                   │
│                       (read-only)                                    │
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │ 1.e4 e5  2.Nf3 Nc6  3.Bb5 a6 …                              │    │
│  └──────────────────────────────────────────────── [◀][▶][⊙]  ┘    │
│  [◀ Prev move]  [▶ Next move]  [⊙ Live]  [Flip]                     │  ← navigation only
└──────────────────────────────────────────────────────────────────────┘
```

### Class Info Row

| Element | Coach | Student |
|---------|-------|---------|
| Class title + opening name | ✅ | ✅ |
| ●REC indicator | ✅ as control (start / pause / stop) | ✅ as status only |

---

## Board Modes (Coach Only)

The mode tab strip is rendered only for the coach. Switching modes broadcasts the transition to all participants via SSE.

### Teaching Mode (default)

Standard analysis board. Full tool set.

```
│ ●Teaching │ Training Game │ Position Drill │ Sandbox │
│ ───────────────────────────────────────────────────  │
│                   CHESS BOARD                        │
│ [Flip][FEN][↩][● Colour▼][Engine▼][✕ Clear]         │
```

**Coach tools:**
- Free piece movement (no turn or rules enforcement)
- Takeback / multi-step undo (`Ctrl+Z` or button)
- FEN input + piece placement editor modal
- Coloured arrows (drag) + circle highlights (right-click); colour picker: Green / Red / Yellow / Blue
- Engine evaluation: eval bar, best-move arrow (purple), top 3 moves; coach-only by default; "Show engine to students" toggle
- Student spotlight: grant one student board control at a time; coach can revoke

**Student view:** Read-only board, notation, and navigation controls only. Arrows and engine output shown if coach has toggled "Show engine to students".

---

### Training Game Mode

Two participants play a rules-enforced timed game. All others spectate.

```
│ Teaching │ ●Training Game │ Position Drill │ Sandbox │
│ ─────────────────────────────────────────────────── │
│ ┌──────────────────────┐  ┌──────────────────────┐  │
│ │ ○ Arun (White) ⏱04:32│  │ ● Priya (Black)⏱03:51│  │
│ └──────────────────────┘  └──────────────────────┘  │
│                   CHESS BOARD                        │
│  Notation                          [◀][▶][⊙]        │
│  [⏸ Pause]  [↩ Force Takeback]  [✕ Abort Game]     │
```

**Coach controls (hidden from all others):**
- Pause / Resume: freezes both clocks
- Force Takeback: reverts last move; 10-second rejection window for players; coach can override
- Abort: ends game without result

**On game end:** Result banner shown; board automatically returns to Teaching Mode with the final position loaded.

**Student view:** Player cards and clocks visible. Board read-only (spectator). No coach controls rendered.

---

### Drill Mode

The coach selects one of seven drill types and runs an interactive exercise. All types share the same coach setup panel, submission status view, private preview flow, and reveal grid. What varies is the student input mechanism and what is shown on the reveal tiles.

---

#### Drill Type Selector (coach setup panel)

Coach enters Drill Mode and sees the type selector before configuring anything else.

```
│ Teaching │ Training Game │ ●Drill │ Sandbox         │
│ ─────────────────────────────────────────────────── │
│                                                     │
│  Select drill type                                  │
│                                                     │
│  ○ Find the Move         Best move for a position   │
│  ○ Find the Mate         Mate-in-N sequence         │
│  ○ Arrange the Board     Drag pieces to target pos  │
│  ○ Name This Piece       Identify piece on square   │
│  ○ Write the Notation    Type the SAN move          │
│  ○ Spot the Illegal Move Identify the illegal move  │
│  ○ Play the Opening      Play N book moves          │
│                                                     │
│  ⏱ Timer  ○ None  ○ 30s  ○ 60s  ○ 90s              │
│                                                     │
│  [← Back]                         [Next: Setup →]  │
```

After selecting a type, coach proceeds to type-specific configuration, then clicks "Start Drill".

---

#### Type-specific student experiences

**Find the Move — student view**

```
│ ─────────────────────────────────────────────────── │
│ ⏱ 00:42   "Find the best move for White"            │
│                                                     │
│                   CHESS BOARD                       │
│                (your private copy)                  │
│                                                     │
│  ▓▓▓▓▓▓▓░░░░░  7 / 12 submitted                    │
│  Click a destination square to submit.              │
│  Submission is final.                               │
```

**Find the Mate — student view**

```
│ ─────────────────────────────────────────────────── │
│ ⏱ 01:12   "White to play — Mate in 3"              │
│                                                     │
│                   CHESS BOARD                       │
│                (your private copy)                  │
│                                                     │
│  Play moves to deliver checkmate.                   │
│  Move 1 of 3                                        │
│  ▓▓▓▓░░░░░░░░░  4 / 12 submitted       [Submit]    │
```

Students play moves freely until checkmate is reached; the board auto-submits on checkmate, or they can click Submit.

**Arrange the Board — student view**

```
│ ─────────────────────────────────────────────────── │
│ ⏱ 00:58   "Recreate the starting position"         │
│                                                     │
│         BLANK BOARD (drag to place pieces)          │
│                                                     │
│  Piece palette:  ♔ ♕ ♖ ♗ ♘ ♟ (White & Black)       │
│                                                     │
│  ▓▓▓▓▓░░░░░░░░  5 / 12 submitted      [Submit]     │
```

Pieces in the palette are dragged to squares. Clicking a placed piece removes it.

**Write the Notation — student view**

```
│ ─────────────────────────────────────────────────── │
│ ⏱ 00:31   "Write the SAN for the highlighted move" │
│                                                     │
│                   CHESS BOARD                       │
│              (arrow shows the move)                 │
│                                                     │
│  ┌──────────────────────────────┐                   │
│  │ e.g. Nf3, O-O, exd5+         │  ← text input    │
│  └──────────────────────────────┘                   │
│  ▓▓▓▓▓▓░░░░░░░  6 / 12 submitted  [Submit / Enter] │
```

**Play the Opening — student view**

```
│ ─────────────────────────────────────────────────── │
│ "Play the first 5 moves of the Ruy Lopez as White"  │
│                                                     │
│                   CHESS BOARD                       │
│                (your private copy)                  │
│                                                     │
│  Move 3 of 5                                        │
│  1.e4 ✓  e5 ✓  2.Nf3 ✓  …                          │
│  ▓▓▓▓▓▓░░░░░░░  6 / 12 submitted                   │
```

Each correct book move turns green in the move list. First deviation is flagged inline: "Off-book — expected Bb5".

---

#### Shared states (all drill types)

**Active state — coach participants panel**

Submission status visible per student. Answer content never shown until the coach previews privately.

```
│ Participants                                        │
│ ─────────────────────────────────────────────────  │
│ Arun      ● Submitted   [👁 Preview]               │
│ Priya     ● Submitted   [👁 Preview]               │
│ Siva      ● Submitted   [👁 Preview]               │
│ Raj       ◌ Thinking…                              │
│ Tanvi     ● Submitted   [👁 Preview]               │
│ Meera     ◌ Thinking…                              │
│ ─────────────────────────────────────────────────  │
│ 5 / 12 submitted                                   │
│ [Reveal All]                                       │
```

**Private preview state — coach only**

Coach clicks [👁 Preview] on any submitted student. Students see no change.

```
│ ┌─ Private preview — not visible to students ────┐ │
│ │ Priya's answer                                  │ │
│ └─────────────────────────────────────────────── ┘ │
│                                                     │
│                   CHESS BOARD                       │
│         (Priya's answer highlighted in amber;       │
│          for multi-move types: full sequence shown) │
│                                                     │
│  [← Prev answer]  [Next answer →]   [Skip]         │
│  [Show to Class]                                   │
```

- [← Prev] / [Next →] cycles through all submitted answers without leaving preview
- [Skip] advances without revealing the current answer to the class
- [Show to Class] adds that student's tile to the class reveal grid

**Partial reveal state (after some answers revealed)**

```
│ ─────────────────────────────────────────────────────────────────── │
│  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐   │
│  │ Arun   → Nf6 ✓   │  │ Priya  → Bg4 ✗   │  │ Siva   [hidden]  │   │
│  └──────────────────┘  └──────────────────┘  └──────────────────┘   │
│  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐   │
│  │ Meera  [hidden]  │  │ Raj    no answer │  │ Tanvi  [hidden]  │   │
│  └──────────────────┘  └──────────────────┘  └──────────────────┘   │
│  ┌────────────────────────────────────────────────────────────────┐  │
│  │ Main board — click any revealed tile to load for discussion    │  │
│  │ [Reveal All]                               [End Drill]         │  │
│  └────────────────────────────────────────────────────────────────┘  │
```

Revealed tiles show the student's answer + auto-validation result (✓ / ✗ / ≈). Unrevealed tiles show the student's name in a greyed state — students know a tile exists but cannot see the answer.

**Full reveal state (Reveal All)**

```
│ ─────────────────────────────────────────────────────────────────── │
│  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐   │
│  │ Arun   → Nf6 ✓   │  │ Priya  → Bg4 ✗   │  │ Siva   → Nf6 ✓   │   │
│  └──────────────────┘  └──────────────────┘  └──────────────────┘   │
│  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐   │
│  │ Meera  → Nf6 ✓   │  │ Raj    no answer │  │ Tanvi  → Bg4 ✗   │   │
│  └──────────────────┘  └──────────────────┘  └──────────────────┘   │
│  ┌────────────────────────────────────────────────────────────────┐  │
│  │ Main board — click any tile to load for discussion             │  │
│  │                                            [End Drill]         │  │
│  └────────────────────────────────────────────────────────────────┘  │
```

Both panels auto-collapse when the reveal grid is active to give the grid maximum canvas. User can reopen panels at any time via the strip.

Tile validation symbols apply across all drill types:
- **✓** correct (exact match or checkmate achieved in N)
- **✗** incorrect
- **≈** partial (e.g., checkmate in more moves than specified; board arrangement with 1–2 pieces misplaced)

---

### Student Sandbox Mode

Each student gets an independent copy of the current position to explore freely.

**Coach view (both panels auto-collapse):**

```
│ ─────────────────────────────────────────────────────────────────── │
│  ┌───────────────────┐  ┌───────────────────┐  ┌───────────────────┐│
│  │ Arun    [Pull ▲]  │  │ Priya   [Watch]   │  │ Siva    [Pull ▲]  ││
│  │ [board thumbnail] │  │ [board thumbnail] │  │ [board thumbnail] ││
│  └───────────────────┘  └───────────────────┘  └───────────────────┘│
│  ┌───────────────────┐  ┌───────────────────┐                        │
│  │ Meera   [Watch]   │  │  + 7 more (scroll) │                        │
│  └───────────────────┘  └───────────────────┘                        │
│  [Close Sandbox → all boards reset, Teaching Mode resumes]           │
```

- [Watch]: enlarges that student's board for the coach to observe (student is not notified)
- [Pull ▲]: copies that student's position to the main board; all participants return to Teaching Mode

**Student view:** Full-size interactive board (no rules enforcement), same layout as Teaching Mode read-only but pieces are moveable.

---

## Reference: Full Desktop — Both Panels Open, Teaching Mode (Coach)

```
┌──────────────────────────────────────────────────────────────────────┐
│ ◆ ChessLeap                                          [⚙]      [👤]  │
├─────────────┬──────────────────────────────────────┬─────────────────┤
│ 👥 12 [⊞▼] │ Sicilian · Ruy Lopez         [●REC]  │ Chat            │
│ ─────────── │ ─────────────────────────────────── │ ─────────────── │
│ ┌─────────┐ │ Teaching│Train│Drill│Sandbox         │ Coach  10:41    │
│ │  Coach  │ │ ─────────────────────────────────── │ "Watch Nf6→"    │
│ │ [video] │ │                                     │                 │
│ └─────────┘ │          CHESS BOARD                │ Arun  10:42     │
│ ┌────┐┌───┐ │                                     │ "Got it!"       │
│ │ S1 ││S2 │ │                                     │                 │
│ └────┘└───┘ │ ┌───────────────────────────────┐   │ ─────────────── │
│ ┌────┐┌───┐ │ │ 1.e4 e5  2.Nf3 Nc6  3.Bb5 a6  │   │ ✋ Arun · Priya │
│ │ S3 ││+8 │ │ └──────────────────── [◀][▶][⊙] ┘   │ 👍 😕 👏 😮    │
│ └────┘└───┘ │ [Flip][FEN][↩][●▼][Engine▼][✕]     │ ┌─────────────┐ │
│             │                                     │ │ message…    │ │
├─────────────┼──────────────────────────────────────┼─────────────────┤
│  [👥●] 12   │     🎤   📹   ✋   [End Class]        │   [💬●] +3      │
└─────────────┴──────────────────────────────────────┴─────────────────┘
     ~240px               ~760px                         ~280px
```

---

## Reference: Full Desktop — Board Only (Both Panels Closed)

```
┌──────────────────────────────────────────────────────────────────────┐
│ ◆ ChessLeap                                          [⚙]      [👤]  │
├──────────────────────────────────────────────────────────────────────┤
│ Sicilian · Ruy Lopez, Morphy Defence                        [●REC]  │
│ ─────────────────────────────────────────────────────────────────── │
│ Teaching │ Training Game │ Position Drill │ Sandbox                  │
│ ─────────────────────────────────────────────────────────────────── │
│                                                                      │
│                          CHESS BOARD                                 │
│                        (full canvas)                                 │
│                                                                      │
│   ┌──────────────────────────────────────────────────────────────┐   │
│   │ 1.e4 e5  2.Nf3 Nc6  3.Bb5 a6  4.Ba4 Nf6  5.O-O Be7 …        │   │
│   └─────────────────────────────────────────── [◀][▶][⊙]       ┘   │
│   [Flip][FEN][↩][●▼][Engine▼][✕ Clear]                              │
├──────────────────────────────────────────────────────────────────────┤
│  [👥] 12        🎤   📹   ✋   [End Class]                  [💬] +3  │
└──────────────────────────────────────────────────────────────────────┘
                              ~1280px
```

---

## Mobile Layout

The strip remains at the bottom. Video and chat open as full-width bottom sheets that slide upward. The board compresses vertically when a sheet is open.

```
Default (board dominant)           Video sheet open
┌─────────────────────┐            ┌─────────────────────┐
│ ◆ ChessLeap  [⚙][👤]│            │ ◆ ChessLeap  [⚙][👤]│
│ Sicilian  [●REC]    │            │ Sicilian  [●REC]    │
│ ─────────────────── │            ├─────────────────────┤
│                     │            │   BOARD (compressed)│
│                     │            │   [◀]  [▶]  [⊙]    │
│    CHESS BOARD      │            ├─────────────────────┤
│    (full height)    │            │ ▔▔▔▔ drag ▔▔▔▔▔▔▔  │
│                     │            │ 👥 12       [⊞▼]   │
│                     │            │ ┌──────┐  ┌──────┐  │
│  1.e4 e5  2.Nf3 … │            │ │Coach │  │  S1  │  │
│  [◀]  [▶]  [⊙]    │            │ └──────┘  └──────┘  │
├─────────────────────┤            │ ┌──────┐  ┌──────┐  │
│[👥]12 🎤 📹 ✋ [↩] [💬]│           │ │  S2  │  │  S3  │  │
└─────────────────────┘            │ └──────┘  └──────┘  │
                                   ├─────────────────────┤
                                   │[👥●]12 🎤 📹 ✋[↩][💬]│
                                   └─────────────────────┘
```

**Mobile coach mode tabs:** Rendered as a horizontally scrollable pill row above the board.

**Mobile board tools:** Accessible via a `[🔧]` button in the strip that opens a bottom sheet — keeps the board uncluttered.

```
Mode tabs (coach, mobile):
│ ‹ Teaching › Training Game  Position Drill  Sandbox │
  ← horizontally scrollable, active mode bold/underlined

Tools sheet (coach, mobile):
┌─────────────────────┐
│ ▔▔▔▔ drag ▔▔▔▔▔▔▔  │
│ [Flip]  [FEN]  [↩]  │
│ [● Colour]  [Engine] │
│ [✕ Clear]           │
│ ─────────────────── │
│ [End Class]         │
└─────────────────────┘
```

---

## Responsive Breakpoints

| Viewport | Layout |
|----------|--------|
| ≥1280px | Three-column: video (240px) + board (flex) + chat (280px) |
| 1024–1279px | Video panel narrower (200px); chat panel narrower (240px) |
| 768–1023px | One panel open at a time maximum; board always ≥ 400px |
| <768px | Single column; panels as bottom sheets; strip at bottom |

---

## Panel Auto-Collapse Rules

| Trigger | Panels collapsed |
|---------|-----------------|
| Position Drill Reveal | Both — student grid needs full canvas |
| Student Sandbox view (coach) | Both — thumbnail grid needs full canvas |
| Screen share active (board area) | Video panel — board takes priority |
| Viewport width drops below 768px | Both — mobile sheet model takes over |

Auto-collapse is animated (200ms slide). User can reopen at any time via strip icons.

---

## Screen Sharing

**Coach screen share:**
- Shared stream replaces the board area by default, or appears as a large tile in the video panel (coach chooses at share time)
- When sharing in the board area: board moves to a minimised panel; "Return to board" button restores it
- Annotation overlay canvas available: coach draws arrows and highlights over the shared screen

**Student screen share:**
- Student clicks "Request Screen Share" in the strip (`⋯` menu on mobile)
- Coach approves or denies via a notification
- One student at a time; shown in the board area for the coach; coach can broadcast to all
- Ends automatically if coach revokes or student leaves

---

## Waiting Room

Students who join before the coach or before "Start Class" land in a waiting room page — a separate page, not the class layout. On session start the page transitions to the full class layout.

```
┌──────────────────────────────────────────────────────────────────────┐
│ ◆ ChessLeap                                          [⚙]      [👤]  │
├──────────────────────────────────────────────────────────────────────┤
│                                                                      │
│                  Intro to Sicilian                                   │
│                  Coach: Lordson V.                                   │
│                  Starting soon…  ⏱ 00:03:22                         │
│                                                                      │
│            ┌────────────────────────────────┐                        │
│            │   Your camera preview          │                        │
│            │                                │                        │
│            └────────────────────────────────┘                        │
│                 🎤 Mic level  ▓▓▓▓░░░                                │
│                                                                      │
│              Waiting for coach to admit you…                         │
│                                                                      │
└──────────────────────────────────────────────────────────────────────┘
```

---

## Online Play (`/play`) Layout

The Online Play layout is simpler — no live class features. Two-column: board on the left, game info + chat on the right.

```
┌──────────────────────────────────────────────────────────────────────┐
│ ◆ ChessLeap                                          [⚙]      [👤]  │
├──────────────────────────────────────────────────┬───────────────────┤
│  ┌────────────────────────────────────────────┐  │ ● Priya (Black)   │
│  │ ● Priya (Black)  ⏱ 03:51                  │  │ ⏱ 03:51           │
│  └────────────────────────────────────────────┘  │                   │
│                                                  │ ─────────────── │
│                  CHESS BOARD                     │ [White] Good luck!│
│                                                  │ [Black] You too   │
│  ┌────────────────────────────────────────────┐  │                   │
│  │ ○ You (White)    ⏱ 04:32                  │  │ ─────────────── │
│  └────────────────────────────────────────────┘  │ [Offer Draw]      │
│  ┌──────────────────────────────────────────┐    │ [Resign]          │
│  │ 1.e4 e5  2.Nf3 Nc6 …      [◀][▶][⊙][Flip│    │                   │
│  └──────────────────────────────────────────┘    │ ┌───────────────┐ │
│                                                  │ │ message…      │ │
└──────────────────────────────────────────────────┴───────────────────┘
```

Player clock cards are displayed above and below the board (standard chess app convention). Game actions (draw offer, resign) and chat are in the right panel.
