import { clocks, initClockSync } from "./sync.js"
import { mergePatch } from "/assets/datastar.js"

// Extract game ID from the URL: /game/<id>
const gameID = location.pathname.split("/")[2]

if (gameID && clocks.white.el && clocks.black.el) {
  clocks.white.el.textContent = "--"
  clocks.black.el.textContent = "--"

  const raf = () => {
    clocks.white.tick()
    clocks.black.tick()
    requestAnimationFrame(raf)
  }
  raf()

  initClockSync(gameID, (gameOverMsg) => {
    mergePatch({
      gameState: gameOverMsg.state,
      gameStateText: gameOverMsg.state_text,
      winner: gameOverMsg.winner,
    })
  })
}

// ── Keyboard navigation ───────────────────────────────────────────────────────
//
// ArrowLeft/ArrowRight click the prev/next notation buttons, which trigger
// DataStar @post requests handled server-side.

document.addEventListener("keydown", (e) => {
  const tag = (document.activeElement?.tagName ?? "").toUpperCase()
  if (tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT") return
  if (e.key === "ArrowLeft") {
    e.preventDefault()
    document.getElementById("notation-prev")?.click()
  } else if (e.key === "ArrowRight") {
    e.preventDefault()
    document.getElementById("notation-next")?.click()
  }
})

// ── Auto-scroll move list to the end when new moves arrive ───────────────────
//
// When a new move is appended (DataStar morphs #move-notation-panel), scroll
// the move list to the bottom so the latest move is always visible — but only
// when the user is watching live (not browsing history).

const observeList = () => {
  const panel = document.getElementById("move-notation-panel")
  if (!panel) return

  new MutationObserver(() => {
    // Only auto-scroll when not viewing history (historyIdx === -1).
    // We check the DOM directly rather than importing getPath to avoid
    // pulling in the full datastar bundle just for this one signal read.
    const list = document.getElementById("move-list")
    if (list && !document.querySelector("[data-show='$viewingHistory'][style='']")) {
      list.scrollTop = list.scrollHeight
    }
  }).observe(panel, { childList: true, subtree: true })
}

observeList()
