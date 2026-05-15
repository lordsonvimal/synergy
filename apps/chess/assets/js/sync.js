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
