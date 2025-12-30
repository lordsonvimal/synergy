import { clocks } from "./sync.js"

function raf() {
  clocks.white.tick()
  clocks.black.tick()
  requestAnimationFrame(raf)
}

raf()
