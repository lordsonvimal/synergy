import { clocks, initClockSync } from "./sync.js"
import { mergePatch } from "/assets/datastar.js"

// Extract game ID from the URL: /game/<id>
const gameID = location.pathname.split("/")[2]

if (gameID && clocks.white.el && clocks.black.el) {
  clocks.white.el.textContent = "--"
  clocks.black.el.textContent = "--"

  // Animation loop runs regardless of connection state.
  const raf = () => {
    clocks.white.tick()
    clocks.black.tick()
    requestAnimationFrame(raf)
  }
  raf()

  // Measure clock offset then open the persistent SSE stream.
  initClockSync(gameID, (gameOverMsg) => {
    // Server has declared game over (flag fall).
    // Patch DataStar signals so the UI updates immediately even if idle.
    mergePatch({
      gameState: gameOverMsg.state,
      gameStateText: gameOverMsg.state_text,
      winner: gameOverMsg.winner,
    })
  })
}
