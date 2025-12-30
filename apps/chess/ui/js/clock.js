export class ClientClock {
  constructor(el) {
    this.el = el
    this.remaining = 0
    this.baseMono = 0
    this.running = false
  }

  sync(snapshot) {
    this.remaining = snapshot.remaining_ns
    this.baseMono = performance.now()
    this.running = snapshot.running
  }

  tick() {
    if (!this.running) return

    const elapsedNs =
      (performance.now() - this.baseMono) * 1e6

    const left = Math.max(0, this.remaining - elapsedNs)
    this.render(left)
  }

  render(ns) {
    const sec = ns / 1e9
    this.el.textContent = sec.toFixed(2)
  }
}
