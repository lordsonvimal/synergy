// Synthesize wood-knock chess sound effects as WAV files.
//
// All sounds are generated from scratch (damped sine sums + filtered noise
// transient) — no third-party samples, no licensing strings attached. Outputs
// to assets/sounds/*.wav. Run via `node scripts/gen-sounds.mjs` whenever you
// want to tweak the character.
//
// Wood-knock model:
//   1. A short noise burst (~3 ms) gives the "click" of the strike.
//   2. A damped sine at the wood's resonant pitch gives the "thunk".
//   3. A second sine one octave up at lower amplitude adds body.
//   Total length ~80–150 ms with exponential amplitude decay.

import { writeFile, mkdir } from "node:fs/promises";
import { join } from "node:path";

const SAMPLE_RATE = 44100;
const OUT_DIR = "assets/sounds";

// --- WAV encoder (mono, 16-bit PCM) ---------------------------------------

function encodeWav(samples) {
  const numSamples = samples.length;
  const byteLen = 44 + numSamples * 2;
  const buf = Buffer.alloc(byteLen);
  buf.write("RIFF", 0);
  buf.writeUInt32LE(byteLen - 8, 4);
  buf.write("WAVE", 8);
  buf.write("fmt ", 12);
  buf.writeUInt32LE(16, 16);              // PCM chunk size
  buf.writeUInt16LE(1, 20);               // PCM format
  buf.writeUInt16LE(1, 22);               // mono
  buf.writeUInt32LE(SAMPLE_RATE, 24);
  buf.writeUInt32LE(SAMPLE_RATE * 2, 28); // byte rate
  buf.writeUInt16LE(2, 32);               // block align
  buf.writeUInt16LE(16, 34);              // bits per sample
  buf.write("data", 36);
  buf.writeUInt32LE(numSamples * 2, 40);
  for (let i = 0; i < numSamples; i++) {
    const v = Math.max(-1, Math.min(1, samples[i]));
    buf.writeInt16LE(Math.round(v * 32760), 44 + i * 2);
  }
  return buf;
}

// --- synthesis primitives -------------------------------------------------

function silence(durationS) {
  return new Float32Array(Math.floor(SAMPLE_RATE * durationS));
}

// Damped sine: amp * sin(2πft) * exp(-decay * t). decay in 1/s.
function dampedSine({ freq, durationS, amp = 1, decay = 30, phase = 0 }) {
  const n = Math.floor(SAMPLE_RATE * durationS);
  const out = new Float32Array(n);
  for (let i = 0; i < n; i++) {
    const t = i / SAMPLE_RATE;
    out[i] = amp * Math.sin(2 * Math.PI * freq * t + phase) * Math.exp(-decay * t);
  }
  return out;
}

// Filtered noise burst with sharp attack and short decay — the "click" of
// stick-on-wood contact. One-pole lowpass to soften the high end.
function noiseTransient({ durationS, amp = 1, decay = 200, lpAlpha = 0.4 }) {
  const n = Math.floor(SAMPLE_RATE * durationS);
  const out = new Float32Array(n);
  let prev = 0;
  for (let i = 0; i < n; i++) {
    const t = i / SAMPLE_RATE;
    const raw = (Math.random() * 2 - 1) * Math.exp(-decay * t);
    prev = prev + lpAlpha * (raw - prev);
    out[i] = amp * prev;
  }
  return out;
}

function mix(...layers) {
  let n = 0;
  for (const l of layers) if (l.length > n) n = l.length;
  const out = new Float32Array(n);
  for (const l of layers) {
    for (let i = 0; i < l.length; i++) out[i] += l[i];
  }
  return out;
}

function concat(...parts) {
  let n = 0;
  for (const p of parts) n += p.length;
  const out = new Float32Array(n);
  let off = 0;
  for (const p of parts) { out.set(p, off); off += p.length; }
  return out;
}

function gain(samples, g) {
  const out = new Float32Array(samples.length);
  for (let i = 0; i < samples.length; i++) out[i] = samples[i] * g;
  return out;
}

// Normalize peak to target (default 0.85) to prevent clipping while keeping
// the relative dynamics within a single sound.
function normalize(samples, target = 0.85) {
  let peak = 0;
  for (let i = 0; i < samples.length; i++) {
    const a = Math.abs(samples[i]);
    if (a > peak) peak = a;
  }
  if (peak < 1e-6) return samples;
  return gain(samples, target / peak);
}

// A "wood knock" at a given pitch — building block for most events.
function woodKnock({ pitch = 260, durationS = 0.14, body = 1.0 }) {
  const click = noiseTransient({ durationS: 0.012, amp: 0.7, decay: 350, lpAlpha: 0.5 });
  const fund  = dampedSine({ freq: pitch,        durationS, amp: 0.9 * body, decay: 28 });
  const harm  = dampedSine({ freq: pitch * 2.02, durationS, amp: 0.35 * body, decay: 50 });
  const sub   = dampedSine({ freq: pitch * 0.5,  durationS, amp: 0.25 * body, decay: 18 });
  return normalize(mix(click, fund, harm, sub));
}

// --- per-event sound recipes ----------------------------------------------

// Both sides share the same wood-knock recipe — the 240 Hz variant we tried
// earlier had a slow-decaying upper harmonic that read as a bird chirp.
function sndMove() {
  return woodKnock({ pitch: 200, durationS: 0.14 });
}

function sndOpponentMove() {
  return woodKnock({ pitch: 200, durationS: 0.14 });
}

function sndCapture() {
  // Two near-overlapping hits — piece being lifted off + new piece set down.
  const hit1 = woodKnock({ pitch: 180, durationS: 0.10, body: 0.9 });
  const gap  = silence(0.025);
  const hit2 = woodKnock({ pitch: 260, durationS: 0.16, body: 1.0 });
  return normalize(concat(hit1, gap, hit2));
}

function sndCastle() {
  // King move + rook move — two evenly spaced knocks.
  const hit1 = woodKnock({ pitch: 220, durationS: 0.11 });
  const gap  = silence(0.07);
  const hit2 = woodKnock({ pitch: 220, durationS: 0.13 });
  return normalize(concat(hit1, gap, hit2));
}

function sndPromote() {
  // Wood knock followed by a soft ascending chime to mark the upgrade.
  const knock = woodKnock({ pitch: 260, durationS: 0.10 });
  const chime1 = dampedSine({ freq: 660,  durationS: 0.30, amp: 0.45, decay: 8 });
  const chime2 = dampedSine({ freq: 880,  durationS: 0.30, amp: 0.40, decay: 8 });
  const chime3 = dampedSine({ freq: 1320, durationS: 0.40, amp: 0.30, decay: 7 });
  const tail = mix(chime1,
    concat(silence(0.06), chime2),
    concat(silence(0.12), chime3));
  return normalize(concat(knock, silence(0.02), tail));
}

function sndCheck() {
  // Sharp high tap with a slight metallic ring — louder than a normal move
  // so it cuts through.
  const click = noiseTransient({ durationS: 0.008, amp: 0.9, decay: 500, lpAlpha: 0.7 });
  const ring1 = dampedSine({ freq: 740,  durationS: 0.25, amp: 0.75, decay: 14 });
  const ring2 = dampedSine({ freq: 1480, durationS: 0.20, amp: 0.40, decay: 22 });
  return normalize(mix(click, ring1, ring2));
}

// Helper for end-game arpeggios: a soft mallet note.
function mallet({ freq, durationS = 0.45, amp = 0.6, decay = 6 }) {
  const click = noiseTransient({ durationS: 0.004, amp: 0.25, decay: 600, lpAlpha: 0.8 });
  const fund  = dampedSine({ freq,         durationS, amp,         decay });
  const harm  = dampedSine({ freq: freq*2, durationS, amp: amp*0.3, decay: decay * 1.6 });
  return mix(click, fund, harm);
}

function sndWin() {
  // Major triad ascending: C5–E5–G5–C6.
  const dt = 0.14; // note onset spacing (s)
  const notes = [523.25, 659.25, 783.99, 1046.5];
  const layers = notes.map((f, i) => concat(
    silence(i * dt),
    mallet({ freq: f, durationS: 0.55 + i * 0.05, amp: 0.55, decay: 6 - i * 0.5 }),
  ));
  return normalize(mix(...layers), 0.9);
}

function sndLose() {
  // Descending minor: G4 → Eb4 → C4 with longer decays.
  const dt = 0.22;
  const notes = [392.00, 311.13, 261.63];
  const layers = notes.map((f, i) => concat(
    silence(i * dt),
    mallet({ freq: f, durationS: 0.7 + i * 0.1, amp: 0.55, decay: 4.5 - i * 0.3 }),
  ));
  return normalize(mix(...layers), 0.85);
}

function sndDraw() {
  // Two equal-pitch notes — neutral, settled.
  const dt = 0.18;
  const f = 392.00; // G4
  const layers = [0, 1].map((i) => concat(
    silence(i * dt),
    mallet({ freq: f, durationS: 0.7, amp: 0.5, decay: 4.5 }),
  ));
  return normalize(mix(...layers), 0.85);
}

// --- output ---------------------------------------------------------------

const TABLE = {
  "move.wav":          sndMove,
  "move-opponent.wav": sndOpponentMove,
  "capture.wav":       sndCapture,
  "castle.wav":        sndCastle,
  "promote.wav":       sndPromote,
  "check.wav":         sndCheck,
  "game-win.wav":      sndWin,
  "game-lose.wav":     sndLose,
  "game-end.wav":      sndDraw,
};

await mkdir(OUT_DIR, { recursive: true });
for (const [name, fn] of Object.entries(TABLE)) {
  const samples = fn();
  const buf = encodeWav(samples);
  const path = join(OUT_DIR, name);
  await writeFile(path, buf);
  console.log(`[gen-sounds] ${name} ${(buf.length / 1024).toFixed(1)} KB`);
}
