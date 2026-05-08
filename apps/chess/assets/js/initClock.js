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

// ── Move notation navigation ──────────────────────────────────────────────────
// Architecture: DataStar morphs the hidden #move-notation-panel data store after
// each move. This code watches that store and renders the current pair into the
// static #notation-* elements (which DataStar never touches).

const store    = document.getElementById("move-notation-panel")
const emptyEl  = document.getElementById("notation-empty")
const moveEl   = document.getElementById("notation-move")
const numEl    = document.getElementById("notation-num")
const whiteEl  = document.getElementById("notation-white")
const blackEl  = document.getElementById("notation-black")
const prevBtn  = document.getElementById("notation-prev")
const nextBtn  = document.getElementById("notation-next")

if (store && moveEl && prevBtn && nextBtn) {
  let idx = 0

  const render = (targetIdx) => {
    const rows = store.querySelectorAll("[data-move-row]")
    const total = rows.length

    if (total === 0) {
      emptyEl.style.display = ""
      moveEl.style.display  = "none"
      prevBtn.classList.add("opacity-40", "pointer-events-none")
      nextBtn.classList.add("opacity-40", "pointer-events-none")
      idx = 0
      return
    }

    idx = Math.max(0, Math.min(total - 1, targetIdx))
    const row = rows[idx]

    emptyEl.style.display = "none"
    moveEl.style.display  = "flex"
    numEl.textContent     = row.dataset.num + "."
    whiteEl.textContent   = row.dataset.white
    blackEl.textContent   = row.dataset.black || ""
    blackEl.style.display = row.dataset.black ? "" : "none"

    prevBtn.classList.toggle("opacity-40",        idx <= 0)
    prevBtn.classList.toggle("pointer-events-none", idx <= 0)
    nextBtn.classList.toggle("opacity-40",        idx >= total - 1)
    nextBtn.classList.toggle("pointer-events-none", idx >= total - 1)
  }

  prevBtn.addEventListener("click", () => render(idx - 1))
  nextBtn.addEventListener("click", () => render(idx + 1))

  // Advance to latest after each SSE morph.
  // childList catches new pairs; attributeFilter catches the black move being
  // added to an existing pair (idiomorph updates data-black on the existing row).
  new MutationObserver(() => {
    const total = store.querySelectorAll("[data-move-row]").length
    render(total - 1)
  }).observe(store, {
    childList: true,
    subtree: true,
    attributes: true,
    attributeFilter: ["data-black"],
  })

  // Initial render (handles page reload mid-game).
  const initialTotal = store.querySelectorAll("[data-move-row]").length
  render(initialTotal > 0 ? initialTotal - 1 : 0)
}
