import { ClientClock } from "./clock.js"

export const clocks = {
  white: new ClientClock(document.getElementById("white-clock")),
  black: new ClientClock(document.getElementById("black-clock")),
}

// Measured once at game start via 3 pings; used in every clock_tick sync.
let clockOffsetNs = 0

// measureOffset calls GET /ping three times and takes the median offset.
// clockOffset = server_ns - (t1 + t3) / 2  (NTP formula)
// Positive offset means server clock is ahead of client.
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

// connectEventStream opens the persistent SSE connection for the given gameID.
export function connectEventStream(gameID, onGameOver) {
  const es = new EventSource(`/game/${gameID}/events`)

  es.onmessage = (e) => {
    let msg
    try {
      msg = JSON.parse(e.data)
    } catch {
      return
    }

    if (msg.type === "clock_tick") {
      clocks.white.sync(
        { remaining_ns: msg.white_remaining_ns, server_ts_ns: msg.server_ts_ns, running: msg.white_running },
        clockOffsetNs,
      )
      clocks.black.sync(
        { remaining_ns: msg.black_remaining_ns, server_ts_ns: msg.server_ts_ns, running: msg.black_running },
        clockOffsetNs,
      )
    } else if (msg.type === "game_over") {
      // GameClockFlagged = 3; pin the flagged clock to exactly 0.
      if (msg.state === 3) {
        if (msg.winner === 1) clocks.white.setRemaining(0) // Black won — White flagged
        else if (msg.winner === 0) clocks.black.setRemaining(0) // White won — Black flagged
      }
      clocks.white.stop()
      clocks.black.stop()
      es.close()
      if (onGameOver) onGameOver(msg)
    }
  }

  return es
}

export async function initClockSync(gameID, onGameOver) {
  await measureOffset()
  return connectEventStream(gameID, onGameOver)
}
