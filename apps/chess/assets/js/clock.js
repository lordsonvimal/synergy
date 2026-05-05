export class ClientClock {
  constructor(el) {
    this.el = el
    this.remainingNs = 0
    this.baseMonoMs = 0   // performance.now() at last sync
    this.running = false
    this.correctionNs = 0
    this.correctionSteps = 0
  }

  // sync is called each time a clock_tick SSE event arrives.
  // snapshot: { remaining_ns, server_ts_ns, running }
  // clockOffsetNs: NTP-measured offset so that clientNow + offset ≈ serverNow
  sync(snapshot, clockOffsetNs) {
    const clientNowNs = Date.now() * 1e6
    const estimatedServerNowNs = clientNowNs + clockOffsetNs
    const transitNs = Math.max(0, estimatedServerNowNs - snapshot.server_ts_ns)
    const trueRemaining = Math.max(0, snapshot.remaining_ns - transitNs)

    const firstSync = this.baseMonoMs === 0
    const wasRunning = this.running
    this.running = snapshot.running

    if (firstSync || !wasRunning) {
      this.remainingNs = trueRemaining
      this.baseMonoMs = performance.now()
      this.correctionNs = 0
      this.correctionSteps = 0
      this.render(trueRemaining)
      return
    }

    const elapsedNs = (performance.now() - this.baseMonoMs) * 1e6
    const clientEstimate = Math.max(0, this.remainingNs - elapsedNs)
    const diff = trueRemaining - clientEstimate

    if (Math.abs(diff) < 500e6) {
      this.correctionNs += diff
      this.correctionSteps = 21
    } else {
      // Large divergence — hard resync.
      this.remainingNs = trueRemaining
      this.baseMonoMs = performance.now()
      this.correctionNs = 0
      this.correctionSteps = 0
    }
  }

  tick() {
    if (!this.el) return

    if (!this.running) {
      // Render the frozen value. Because we always re-base below, this
      // reflects the last computed value — 0 if the clock expired.
      this.render(this.remainingNs)
      return
    }

    const elapsedNs = (performance.now() - this.baseMonoMs) * 1e6
    let left = Math.max(0, this.remainingNs - elapsedNs)

    if (this.correctionSteps > 0) {
      const step = this.correctionNs / this.correctionSteps
      left = Math.max(0, left + step)
      this.correctionNs -= step
      this.correctionSteps--
    }

    // Always re-base so that when stop() is called, this.remainingNs
    // reflects the last rendered value (0 if expired, not the last sync value).
    this.remainingNs = left
    this.baseMonoMs = performance.now()

    this.render(left)
  }

  // setRemaining forces a specific value, used by game_over to pin the
  // flagged clock to exactly 0 regardless of client estimate.
  setRemaining(ns) {
    this.remainingNs = ns
  }

  render(ns) {
    if (!this.el) return
    const totalSec = Math.ceil(ns / 1e9)
    const m = Math.floor(totalSec / 60)
    const s = totalSec % 60
    this.el.textContent = `${m}:${String(s).padStart(2, "0")}`
  }

  stop() {
    this.running = false
  }
}
