# Chess sound assets

All `.wav` files in this directory are **synthesized from scratch** by
`scripts/gen-sounds.mjs` (pure Node, no third-party samples). They carry
no license obligations — own them, modify them, ship them with a closed-
source build.

## Regenerate

```sh
node scripts/gen-sounds.mjs
```

Tweak the recipes in that script (pitch, decay, overlap) to taste; rerun
and the new files land here. The build pipeline copies this directory
verbatim to `dist/sounds/`, served at `/static/sounds/<name>.wav`.

## File map

| File | Plays on |
|---|---|
| `move.wav` | local player's normal move |
| `move-opponent.wav` | opponent's normal move (received via SSE) |
| `capture.wav` | any capture (including en-passant) |
| `castle.wav` | castling |
| `promote.wav` | pawn promotion |
| `check.wav` | any move that gives check (overrides the others) |
| `game-win.wav` | game ended and this client won |
| `game-lose.wav` | game ended and this client lost |
| `game-end.wav` | draw / spectator / neutral game end |
