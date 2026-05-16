import { ClientClock } from "./clock.js"

export const clocks = {
  white: new ClientClock(document.getElementById("white-clock")),
  black: new ClientClock(document.getElementById("black-clock")),
}

let clockOffsetNs = 0

export function getClockOffset() {
  return clockOffsetNs
}

// measureOffset calls GET /ping three times and takes the median offset.
// clockOffset = server_ns - (t1 + t3) / 2  (NTP formula)
// Runs in the background — clock accuracy is not blocking.
export async function measureClockOffset() {
  const samples = []
  for (let i = 0; i < 3; i++) {
    const t1 = Date.now() * 1e6
    const res = await fetch("/ping")
    const t3 = Date.now() * 1e6
    const { server_ns } = await res.json()
    samples.push(server_ns - (t1 + t3) / 2)
  }
  samples.sort((a, b) => a - b)
  clockOffsetNs = samples[1] // median — discard outlier caused by congestion
}

// Re-measure periodically so the offset doesn't drift if RTT changes mid-game
// (route flaps, proxy changes, network handoff). The next clock_tick after a
// re-measurement folds the new offset in smoothly via ClientClock.sync's
// correctionNs ramp — small drifts (< 500 ms) are blended over ~20 ticks
// rather than hard-jumped.
const RESYNC_INTERVAL_MS = 60_000
let resyncInterval = null

export function startPeriodicClockResync() {
  if (resyncInterval !== null) return
  resyncInterval = setInterval(() => {
    measureClockOffset().catch(() => {}) // transient ping failures are fine; next tick retries
  }, RESYNC_INTERVAL_MS)
}

export function stopPeriodicClockResync() {
  if (resyncInterval !== null) {
    clearInterval(resyncInterval)
    resyncInterval = null
  }
}
