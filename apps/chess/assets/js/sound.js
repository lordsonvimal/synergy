// Chess sound effects — move/capture/check/castle/promote + game-end.
//
// Usage from board.js:
//   import { playMove, playGameEnd, isSoundEnabled } from "./sound.js";
//   playMove({ capture, castle, promotion, check, isOpponent });
//   playGameEnd({ outcome: "win" | "lose" | "draw" });
//
// On/off is stored in localStorage under "chess-sound" ("on" | "off").
// Default is "on". The soundtoggle component reads/writes the same key.
//
// Audio files are expected at /static/sounds/<name>.ogg (or .mp3 — we try
// .ogg first, falling back to .mp3). Missing files fail silently so a
// half-installed asset set never breaks the move handler.

const STORAGE_KEY = "chess-sound";

// Filenames (without extension) keyed by logical event.
const FILES = {
  move:     "move",
  capture:  "capture",
  castle:   "castle",
  check:    "check",
  promote:  "promote",
  opponent: "move-opponent",
  win:      "game-win",
  lose:     "game-lose",
  draw:     "game-end",
};

// Pool size per sound — lets two near-simultaneous plays (e.g. capture + check)
// fire without one cutting the other off. HTMLAudioElement.play() restarts
// from currentTime, so a single element drops the previous play.
const POOL_SIZE = 2;

const pools = {};        // name -> HTMLAudioElement[]
const poolIdx = {};      // name -> next index to use
let initialized = false;

function makeAudio(name) {
  // WAV is the source of truth — files are synthesized by
  // scripts/gen-sounds.mjs and small enough that the size overhead vs.
  // compressed formats is irrelevant for sub-second clips. Universally
  // supported, no codec licensing concerns.
  const a = new Audio(`/static/sounds/${name}.wav`);
  a.preload = "auto";
  return a;
}

function ensureInit() {
  if (initialized) return;
  initialized = true;
  for (const [key, name] of Object.entries(FILES)) {
    pools[key] = Array.from({ length: POOL_SIZE }, () => makeAudio(name));
    poolIdx[key] = 0;
  }
}

export function isSoundEnabled() {
  // Default ON — only "off" disables. New users get sound without opting in.
  return localStorage.getItem(STORAGE_KEY) !== "off";
}

export function setSoundEnabled(on) {
  localStorage.setItem(STORAGE_KEY, on ? "on" : "off");
}

function play(key) {
  if (!isSoundEnabled()) return;
  ensureInit();
  const pool = pools[key];
  if (!pool) return;
  const i = poolIdx[key];
  poolIdx[key] = (i + 1) % pool.length;
  const el = pool[i];
  try {
    el.currentTime = 0;
    const p = el.play();
    // play() returns a Promise in modern browsers; reject is fine — usually
    // an autoplay-policy block before the first user gesture. Swallow it.
    if (p && typeof p.catch === "function") p.catch(() => {});
  } catch {
    // ignore
  }
}

// playMove picks the most specific sound for a single move. Priority order
// mirrors chess.com's convention: checkmate handled separately by
// playGameEnd, so here check trumps everything else; otherwise
// promote > castle > capture > move (self/opponent variant).
export function playMove({ capture = false, castle = false, promotion = false, check = false, isOpponent = false } = {}) {
  if (check) { play("check"); return; }
  if (promotion) { play("promote"); return; }
  if (castle) { play("castle"); return; }
  if (capture) { play("capture"); return; }
  play(isOpponent ? "opponent" : "move");
}

// playGameEnd: outcome is "win" | "lose" | "draw". Spectators and solo games
// should pass "draw" so they hear the neutral end sound.
export function playGameEnd({ outcome }) {
  if (outcome === "win") play("win");
  else if (outcome === "lose") play("lose");
  else play("draw");
}
