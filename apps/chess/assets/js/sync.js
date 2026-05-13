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
async function measureOffset() {
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

// connectEventStream opens a persistent SSE connection to eventURL.
// onMessage(msg, es) is called for every parsed JSON event.
// Callers are responsible for handling all event types and closing es when done.
export function connectEventStream(eventURL, onMessage) {
  const es = new EventSource(eventURL)

  es.onmessage = (e) => {
    let msg
    try {
      msg = JSON.parse(e.data)
    } catch {
      return
    }
    onMessage(msg, es)
  }

  return es
}

export async function initClockSync(eventURL, onMessage) {
  await measureOffset()
  return connectEventStream(eventURL, onMessage)
}
