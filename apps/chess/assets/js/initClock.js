import { clocks, measureClockOffset, getClockOffset, startPeriodicClockResync, stopPeriodicClockResync } from "./sync.js"
import { effect, getPath, mergePatch } from "datastar"

// All server→client communication on game pages goes through ONE datastar SSE
// connection (opened by the data-init div in the page template). Every event
// arrives as a datastar signal patch; this module reacts via effect() rather
// than running its own EventSource. Nothing in here opens a connection.

const REMATCH_TIMEOUT_S = 30
let rematchSecondsLeft = 0
let rematchTimerInterval = null

const OFFER_TIMEOUT_S = 30
let drawSecondsLeft = 0
let drawTimerInterval = null
let takebackSecondsLeft = 0
let takebackTimerInterval = null

// Final clock values for ended games. Plain JS vars: simpler than signals
// and there's no cross-tab observer that needs to react to them.
let gameEndedWhiteRemNs = null
let gameEndedBlackRemNs = null

const playerRole = (() => {
  try {
    return JSON.parse(document.querySelector('[data-signals]').getAttribute('data-signals')).role ?? ''
  } catch { return '' }
})()

const isTimed = (() => {
  try {
    const sig = JSON.parse(document.querySelector('[data-signals]').getAttribute('data-signals'))
    return sig.timed !== false
  } catch { return true }
})()

const parts = location.pathname.split("/")
const gameID = parts[2]

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

function clearDrawTimer() {
  if (drawTimerInterval !== null) {
    clearInterval(drawTimerInterval)
    drawTimerInterval = null
  }
}

function startDrawTimer() {
  clearDrawTimer()
  drawSecondsLeft = OFFER_TIMEOUT_S
  mergePatch({ drawSecondsLeft })
  drawTimerInterval = setInterval(() => {
    drawSecondsLeft = Math.max(0, drawSecondsLeft - 1)
    mergePatch({ drawSecondsLeft })
    if (drawSecondsLeft <= 0) clearDrawTimer()
  }, 1000)
}

function clearTakebackTimer() {
  if (takebackTimerInterval !== null) {
    clearInterval(takebackTimerInterval)
    takebackTimerInterval = null
  }
}

function startTakebackTimer() {
  clearTakebackTimer()
  takebackSecondsLeft = OFFER_TIMEOUT_S
  mergePatch({ takebackSecondsLeft })
  takebackTimerInterval = setInterval(() => {
    takebackSecondsLeft = Math.max(0, takebackSecondsLeft - 1)
    mergePatch({ takebackSecondsLeft })
    if (takebackSecondsLeft <= 0) clearTakebackTimer()
  }, 1000)
}

// ── Clock display setup ──────────────────────────────────────────────────────

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
    measureClockOffset() // background NTP sync; non-blocking
    startPeriodicClockResync() // re-measure every 60s to catch route/RTT changes
    window.addEventListener("beforeunload", stopPeriodicClockResync)
  }
}

// ── Effects ──────────────────────────────────────────────────────────────────
//
// Effects are set up only after datastar has populated the page's signals.
// If we wired them synchronously, the first run would see every signal as
// undefined, subscribe to nothing, and never re-fire when patches arrive.

function setupEffects() {
  if (getPath("gameState") === undefined) {
    setTimeout(setupEffects, 16)
    return
  }

  // Pre-initialise every server-pushed signal that an effect reads. Datastar's
  // reactive tracker silently skips undefined reads, so an effect's first run
  // would not subscribe to a signal that has not arrived yet — and a later SSE
  // patch carrying the real value would never wake the effect. We seed each
  // key with a sentinel only when it does not already exist, so we never
  // overwrite a value that the SSE happened to deliver before this point.
  const initIfMissing = (key, value) => {
    if (getPath(key) === undefined) mergePatch({ [key]: value })
  }
  initIfMissing("clkW", 0)
  initIfMissing("clkB", 0)
  initIfMissing("clkWRun", false)
  initIfMissing("clkBRun", false)
  initIfMissing("clkTs", 0)
  initIfMissing("firstMoveDeadlineNs", 0)
  initIfMissing("firstMoveSecondsLeft", 0)
  initIfMissing("rematchProposedBy", "")
  initIfMissing("rematchAcceptedUrl", "")
  initIfMissing("drawOfferedBy", "")
  initIfMissing("takebackOfferedBy", "")
  initIfMissing("seq", 0)
  initIfMissing("lastDrawSeqWhite", -1)
  initIfMissing("lastDrawSeqBlack", -1)
  initIfMissing("lastTakebackSeqWhite", -1)
  initIfMissing("lastTakebackSeqBlack", -1)
  initIfMissing("keepaliveTs", 0)
  initIfMissing("connectionDown", false)

  // Reconnect watchdog: server sends keepaliveTs every 10s. If the gap
  // exceeds DOWN_MS we show the "Reconnecting" banner and freeze the local
  // clock display (Datastar is already auto-retrying the SSE). If the gap
  // exceeds RELOAD_MS we assume Datastar's retry isn't recovering and
  // reload as a last resort.
  const DOWN_MS = 12000
  const RELOAD_MS = 30000
  let lastKeepAliveMs = Date.now()
  let bannerShown = false
  effect(() => {
    getPath("keepaliveTs") // register dep
    lastKeepAliveMs = Date.now()
    if (bannerShown) {
      bannerShown = false
      mergePatch({ connectionDown: false })
      // Clock running-state is restored by the next clkTs patch, which
      // arrives moments after a fresh SSE connection finishes its initial
      // sync. Nothing to do here.
    }
  })
  const reconnectTimer = setInterval(() => {
    const gap = Date.now() - lastKeepAliveMs
    if (gap > RELOAD_MS) {
      window.location.reload()
    } else if (gap > DOWN_MS && !bannerShown) {
      bannerShown = true
      mergePatch({ connectionDown: true })
      // Freeze the local clock display — no fresh ticks are arriving and
      // we don't want to keep counting down against a player who is in
      // effect not playing. The next clkTs patch (post-reconnect) calls
      // sync() which restores .running from the server's authoritative
      // snapshot.
      if (isTimed) {
        clocks.white.stop()
        clocks.black.stop()
      }
    }
  }, 2000)
  window.addEventListener("beforeunload", () => clearInterval(reconnectTimer))

  // First-move countdown: while both players are connected but white hasn't
  // moved yet, the server pushes firstMoveDeadlineNs (unix ns). Tick a local
  // 1Hz timer to update firstMoveSecondsLeft so the banner shows seconds
  // remaining. The server-side watchdog is authoritative for abandonment;
  // this is display-only.
  let firstMoveInterval = null
  const clearFirstMoveTimer = () => {
    if (firstMoveInterval !== null) {
      clearInterval(firstMoveInterval)
      firstMoveInterval = null
    }
  }
  const updateFirstMoveSecondsLeft = () => {
    const deadlineNs = Number(getPath("firstMoveDeadlineNs") ?? 0)
    if (!deadlineNs) {
      mergePatch({ firstMoveSecondsLeft: 0 })
      return 0
    }
    const offsetNs = getClockOffset() // server - client, in ns
    const nowNs = Date.now() * 1e6 + offsetNs
    const remNs = Math.max(0, deadlineNs - nowNs)
    const remS = Math.ceil(remNs / 1e9)
    mergePatch({ firstMoveSecondsLeft: remS })
    return remS
  }
  effect(() => {
    const deadlineNs = Number(getPath("firstMoveDeadlineNs") ?? 0)
    clearFirstMoveTimer()
    if (!deadlineNs) {
      mergePatch({ firstMoveSecondsLeft: 0 })
      return
    }
    if (updateFirstMoveSecondsLeft() <= 0) return
    firstMoveInterval = setInterval(() => {
      if (updateFirstMoveSecondsLeft() <= 0) clearFirstMoveTimer()
    }, 250)
  })

  // Clock-tick effect: server pushes {clkW, clkB, clkWRun, clkBRun, clkTs}
  // every second (or on every move). For live games we sync into ClientClock
  // so the RAF loop continues counting down with drift correction. For ended
  // games we cache the final value and update the display directly when not
  // browsing history.
  if (isTimed) {
    let lastClkTs = 0
    effect(() => {
      const ts = getPath("clkTs")
      if (ts == null || ts === lastClkTs) return
      lastClkTs = ts
      const wRem = getPath("clkW")
      const bRem = getPath("clkB")
      const wRun = getPath("clkWRun")
      const bRun = getPath("clkBRun")
      const isEnded = (getPath("gameState") ?? 0) !== 0
      if (!isEnded) {
        const offset = getClockOffset()
        clocks.white.sync({ remaining_ns: wRem, server_ts_ns: ts, running: wRun }, offset)
        clocks.black.sync({ remaining_ns: bRem, server_ts_ns: ts, running: bRun }, offset)
      } else {
        gameEndedWhiteRemNs = wRem
        gameEndedBlackRemNs = bRem
        if (!(getPath("viewingHistory") ?? false)) {
          clocks.white.setRemaining(wRem)
          clocks.black.setRemaining(bRem)
        }
      }
    })
  }

  // Game over → close any open mid-game offer prompts/pending banners so the
  // game-over UI isn't covered by a stale takeback/draw modal.
  effect(() => {
    const gState = getPath("gameState") ?? 0
    if (gState !== 0) {
      clearDrawTimer()
      clearTakebackTimer()
      mergePatch({
        drawOffered: false, drawPending: false, drawSecondsLeft: 0,
        takebackOffered: false, takebackPending: false, takebackSecondsLeft: 0,
      })
    }
  })

  // Game-over edge: gameState transitions from 0 (ongoing) to non-zero.
  // Stop both clocks and freeze the final values. Flag-state (3) pins the
  // losing side to exactly 0.
  let prevGameState = getPath("gameState") ?? 0
  effect(() => {
    const gState = getPath("gameState") ?? 0
    if (gState !== 0 && prevGameState === 0) {
      if (gState === 3) {
        const winner = getPath("winner")
        if (winner === 1) clocks.white.setRemaining(0)
        else if (winner === 0) clocks.black.setRemaining(0)
      }
      clocks.white.stop()
      clocks.black.stop()
      gameEndedWhiteRemNs = clocks.white.remainingNs
      gameEndedBlackRemNs = clocks.black.remainingNs
    }
    prevGameState = gState
  })

  // Rematch proposer: empty string (or null) means no rematch is pending.
  // A non-empty value identifies who proposed; the recipient sees the prompt,
  // the proposer sees the waiting state. Transitioning back to empty means
  // declined or expired (one collapsed handler — the UI is the same).
  let prevProposer = ""
  effect(() => {
    const proposer = getPath("rematchProposedBy") ?? ""
    if (proposer && proposer !== prevProposer) {
      startRematchTimer()
      if (proposer === playerRole) {
        mergePatch({ rematchPending: true })
      } else {
        mergePatch({ rematchProposed: true })
      }
    } else if (!proposer && prevProposer) {
      clearRematchTimer()
      mergePatch({ rematchProposed: false, rematchPending: false, rematchSecondsLeft: 0 })
    }
    prevProposer = proposer
  })

  // Draw offer: server pushes drawOfferedBy (role string or empty). Mirror the
  // rematch pattern — recipient sees the prompt, proposer sees the pending
  // state, transition to empty clears both.
  let prevDrawBy = ""
  effect(() => {
    const offerer = getPath("drawOfferedBy") ?? ""
    if (offerer && offerer !== prevDrawBy) {
      startDrawTimer()
      if (offerer === playerRole) {
        mergePatch({ drawPending: true, drawOffered: false })
      } else {
        mergePatch({ drawOffered: true, drawPending: false })
      }
    } else if (!offerer && prevDrawBy) {
      clearDrawTimer()
      mergePatch({ drawOffered: false, drawPending: false, drawSecondsLeft: 0 })
    }
    prevDrawBy = offerer
  })

  // Takeback offer: same shape as draw.
  let prevTakebackBy = ""
  effect(() => {
    const proposer = getPath("takebackOfferedBy") ?? ""
    if (proposer && proposer !== prevTakebackBy) {
      startTakebackTimer()
      if (proposer === playerRole) {
        mergePatch({ takebackPending: true, takebackOffered: false })
      } else {
        mergePatch({ takebackOffered: true, takebackPending: false })
      }
    } else if (!proposer && prevTakebackBy) {
      clearTakebackTimer()
      mergePatch({ takebackOffered: false, takebackPending: false, takebackSecondsLeft: 0 })
    }
    prevTakebackBy = proposer
  })

  // Rematch accepted: server sets rematchAcceptedUrl for the proposer to
  // navigate to (the accepter follows their own POST response redirect).
  effect(() => {
    const url = getPath("rematchAcceptedUrl") ?? ""
    if (url) {
      clearRematchTimer()
      location.href = url
    }
  })

  // History clock: when viewing a past move on an ended game, show the clock
  // value captured at that move. Reverts to the final clock when leaving
  // history view.
  if (isTimed) {
    effect(() => {
      const gState = getPath("gameState") ?? 0
      if (gState === 0) return
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
}

if (gameID) setupEffects()

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

