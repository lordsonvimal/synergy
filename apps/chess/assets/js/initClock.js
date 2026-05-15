import { clocks, initClockSync, getClockOffset } from "./sync.js"
import { effect, getPath, mergePatch } from "/assets/datastar.js"

const REMATCH_TIMEOUT_S = 30
let rematchSecondsLeft = 0
let rematchTimerInterval = null

// Final clock values for ended games. Stored as plain JS variables — datastar's
// reactive system silently fails to track signals that start as null/undefined,
// so we cannot rely on signals to carry these values across async boundaries.
let gameEndedWhiteRemNs = null
let gameEndedBlackRemNs = null

// Read the player's role from the initial data-signals JSON (set once at page load).
const playerRole = (() => {
  try {
    return JSON.parse(document.querySelector('[data-signals]').getAttribute('data-signals')).role ?? ''
  } catch { return '' }
})()

function clearRematchTimer() {
  if (rematchTimerInterval !== null) {
    clearInterval(rematchTimerInterval)
    rematchTimerInterval = null
  }
}

function startRematchTimer() {
  clearRematchTimer()
  rematchSecondsLeft = REMATCH_TIMEOUT_S
  mergePatch({ rematchSecondsLeft })
  rematchTimerInterval = setInterval(() => {
    rematchSecondsLeft = Math.max(0, rematchSecondsLeft - 1)
    mergePatch({ rematchSecondsLeft })
    if (rematchSecondsLeft <= 0) clearRematchTimer()
  }, 1000)
}

// Works for both /game/<id> and /play/<id> routes.
const parts = location.pathname.split("/")
const routePrefix = parts[1]  // "solo" | "play"
const gameID = parts[2]

const isTimed = (() => {
  try {
    const sig = JSON.parse(document.querySelector('[data-signals]').getAttribute('data-signals'))
    return sig.timed !== false
  } catch { return true }
})()

if (gameID && clocks.white.el && clocks.black.el) {
  if (!isTimed) {
    for (const el of [clocks.white.el, clocks.black.el]) {
      el.textContent = "∞"
      el.style.fontSize = "1.25rem"
      el.style.fontFamily = "inherit"
    }
  } else {
    clocks.white.el.textContent = "--"
    clocks.black.el.textContent = "--"

    const raf = () => {
      clocks.white.tick()
      clocks.black.tick()
      requestAnimationFrame(raf)
    }
    raf()
  }
}

if (gameID) {
  initClockSync(`/${routePrefix}/${gameID}/events`, (msg, es) => {
    switch (msg.type) {
      case "clock_tick": {
        if (!isTimed) break
        const isEnded = (getPath("gameState") ?? 0) !== 0
        if (!isEnded) {
          // Live game: sync with server time for drift correction.
          const offset = getClockOffset()
          clocks.white.sync(
            { remaining_ns: msg.white_remaining_ns, server_ts_ns: msg.server_ts_ns, running: msg.white_running },
            offset,
          )
          clocks.black.sync(
            { remaining_ns: msg.black_remaining_ns, server_ts_ns: msg.server_ts_ns, running: msg.black_running },
            offset,
          )
        } else {
          // Ended game: store the final values, then update the display directly
          // unless the user is currently browsing history. This is the only
          // place the page-load case is handled — the effect below cannot be
          // relied on because it is set up before datastar populates signals
          // from the page's data-signals attribute, so its first run subscribes
          // to nothing.
          gameEndedWhiteRemNs = msg.white_remaining_ns
          gameEndedBlackRemNs = msg.black_remaining_ns
          if (!(getPath("viewingHistory") ?? false)) {
            clocks.white.setRemaining(gameEndedWhiteRemNs)
            clocks.black.setRemaining(gameEndedBlackRemNs)
          }
        }
        break
      }

      case "game_over":
        // Pin the flagged clock to exactly 0 on flag, but only on the first
        // game_over. On SSE reconnect game_over replays, and if the user is
        // viewing history at that moment, setRemaining(0) would clobber the
        // history display. The same null-guard protects the JS-var capture.
        if (gameEndedWhiteRemNs === null) {
          if (msg.state === 3) {
            if (msg.winner === 1) clocks.white.setRemaining(0)
            else if (msg.winner === 0) clocks.black.setRemaining(0)
          }
          clocks.white.stop()
          clocks.black.stop()
          gameEndedWhiteRemNs = clocks.white.remainingNs
          gameEndedBlackRemNs = clocks.black.remainingNs
        } else {
          clocks.white.stop()
          clocks.black.stop()
        }
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
        startRematchTimer()
        // Only the recipient sees the accept/decline prompt; proposer sees waiting state.
        if (msg.proposed_by === playerRole) {
          mergePatch({ rematchPending: true })
        } else {
          mergePatch({ rematchProposed: true })
        }
        break

      case "rematch_accepted":
        clearRematchTimer()
        es.close()
        // Proposer navigates to their token URL — server can only set the
        // accepter's cookie via HTTP response, not over SSE.
        if (msg.proposer_redirect_url) {
          location.href = msg.proposer_redirect_url
        }
        break

      case "rematch_declined":
      case "rematch_expired":
        clearRematchTimer()
        mergePatch({ rematchProposed: false, rematchPending: false, rematchSecondsLeft: 0 })
        break
    }
  })
}

// ── History clock sync (ended games only) ────────────────────────────────────
// When viewing move history in a finished game, show the clock at each move.
// For live games, history nav does not affect the clock display.
//
// CRITICAL: this module loads before datastar scans the DOM and populates
// signals from data-signals attributes, so calling effect() synchronously
// here would have its first run see all signals as undefined and subscribe
// to nothing. We poll until gameState is populated (signalling datastar
// initialisation has happened for this page's signals) before setting up
// the effect, so it tracks every signal correctly on its first run.
function setupEndedGameEffect() {
  if (getPath("gameState") === undefined) {
    setTimeout(setupEndedGameEffect, 16)
    return
  }
  effect(() => {
    const gState = getPath("gameState") ?? 0
    if (gState === 0) return // live game — clock controlled by sync() in clock_tick

    const viewingHistory = getPath("viewingHistory") ?? false
    getPath("historyIdx") // tracked dep
    const historyWRem = getPath("historyWhiteRemNs")
    const historyBRem = getPath("historyBlackRemNs")

    if (viewingHistory) {
      if (historyWRem != null) clocks.white.setRemaining(historyWRem)
      if (historyBRem != null) clocks.black.setRemaining(historyBRem)
    } else if (gameEndedWhiteRemNs != null) {
      clocks.white.setRemaining(gameEndedWhiteRemNs)
      clocks.black.setRemaining(gameEndedBlackRemNs)
    }
  })
}

if (isTimed && gameID) setupEndedGameEffect()

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
