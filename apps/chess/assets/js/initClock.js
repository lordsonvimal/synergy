import { clocks, initClockSync, getClockOffset } from "./sync.js"
import { mergePatch } from "/assets/datastar.js"

// Works for both /game/<id> and /play/<id> routes.
const parts = location.pathname.split("/")
const routePrefix = parts[1]  // "game" | "play"
const gameID = parts[2]

if (gameID && clocks.white.el && clocks.black.el) {
  clocks.white.el.textContent = "--"
  clocks.black.el.textContent = "--"

  const raf = () => {
    clocks.white.tick()
    clocks.black.tick()
    requestAnimationFrame(raf)
  }
  raf()
}

if (gameID) {
  initClockSync(`/${routePrefix}/${gameID}/events`, (msg, es) => {
    switch (msg.type) {
      case "clock_tick": {
        const offset = getClockOffset()
        clocks.white.sync(
          { remaining_ns: msg.white_remaining_ns, server_ts_ns: msg.server_ts_ns, running: msg.white_running },
          offset,
        )
        clocks.black.sync(
          { remaining_ns: msg.black_remaining_ns, server_ts_ns: msg.server_ts_ns, running: msg.black_running },
          offset,
        )
        break
      }

      case "game_over":
        // Pin the flagged clock to exactly 0 on flag.
        if (msg.state === 3) {
          if (msg.winner === 1) clocks.white.setRemaining(0)
          else if (msg.winner === 0) clocks.black.setRemaining(0)
        }
        clocks.white.stop()
        clocks.black.stop()
        // Keep the EventSource open — rematch events (proposed/accepted/declined)
        // arrive on the same stream after the game ends.
        mergePatch({
          gameState: msg.state,
          gameStateText: msg.state_text,
          winner: msg.winner,
          claimVictory: false,
        })
        break

      // ── Play-mode events (no-ops in game mode) ────────────────────────────

      case "online_status":
        mergePatch({ whiteOnline: msg.white_online, blackOnline: msg.black_online })
        break

      case "clock_unlocked":
        mergePatch({ clockUnlocked: true })
        break

      case "claim_victory_available":
        mergePatch({ claimVictory: true })
        break

      case "game_cancelled":
        es.close()
        mergePatch({ gameState: 5, gameStateText: "Game cancelled" })
        break

      case "rematch_proposed":
        mergePatch({ rematchProposed: true })
        break

      case "rematch_accepted":
        es.close()
        // Proposer navigates to their token URL — server can only set the
        // accepter's cookie via HTTP response, not over SSE.
        if (msg.proposer_redirect_url) {
          location.href = msg.proposer_redirect_url
        }
        break

      case "rematch_declined":
      case "rematch_expired":
        mergePatch({ rematchProposed: false })
        break
    }
  })
}

// ── Keyboard navigation ───────────────────────────────────────────────────────

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

// ── Auto-scroll move list on new moves ───────────────────────────────────────

const observeList = () => {
  const panel = document.getElementById("move-notation-panel")
  if (!panel) return

  new MutationObserver(() => {
    const list = document.getElementById("move-list")
    if (list && !document.querySelector("[data-show='$viewingHistory'][style='']")) {
      list.scrollTop = list.scrollHeight
    }
  }).observe(panel, { childList: true, subtree: true })
}

observeList()
