// Optimistic chessboard controller.
//
// Selection (highlighting source square + possible-move targets) runs entirely
// client-side via chessops — no server round-trip. Only the actual move POSTs
// to the server. On the server response the morphed board HTML reconciles any
// disagreement (which should be vanishingly rare since chessops and the Go
// engine implement the same FIDE rules).
import { Chess } from "chessops/chess";
import { parseFen, makeFen } from "chessops/fen";
import { effect, getPath, mergePatch } from "datastar";

// Mirror of engine.Piece (iota from Pawn=0 .. King=5) so we can plumb
// promotion choices through HTTP/URL params without a per-direction enum.
const ROLE_FROM_PIECE = { 0: "pawn", 1: "knight", 2: "bishop", 3: "rook", 4: "queen", 5: "king" };
const ROLE_TO_PIECE   = { pawn: 0, knight: 1, bishop: 2, rook: 3, queen: 4, king: 5 };
// engine.NoSquare is 64 (one past h8). Must match server/engine — using the
// wrong sentinel here silently breaks every "do I have a selection?" check
// because getPath("selectedSquare") returns 64 in the unselected state.
const NO_SQUARE = 64;
// engine.NoPiece is 255. Pawn is 0, so a `!getPath('promotionPiece')` check
// would both miss the unset state (255 is truthy) and wrongly treat a Pawn
// (0) as unset. Always compare against this sentinel explicitly.
const NO_PIECE = 255;

let chess = null;
let lastFen = null;
let gameId = null;
let routePrefix = null;

function setFen(fen) {
  if (!fen) return;
  // Only rebuild the chess instance when the FEN actually changed — parseFen
  // and Chess.fromSetup aren't cheap. The derived signals below ALWAYS
  // republish: applyOptimistic updates lastFen locally before the server's
  // echo arrives, so this function gets called with fen === lastFen on
  // every move's server-side confirmation. If we early-returned there, king
  // squares and check status would never publish for moves the local player
  // made (incl. the mating move).
  if (fen !== lastFen || !chess) {
    const parsed = parseFen(fen);
    if (parsed.isErr) {
      // Loud: a malformed FEN from the server leaves the client unable to
      // validate any move locally. Past offender was the engine emitting "a9"
      // for no-en-passant; surface anything similar fast.
      console.error("[chessleap] setFen: parseFen failed for", fen, parsed.error);
      return;
    }
    const setup = parsed.unwrap();
    const built = Chess.fromSetup(setup);
    if (built.isErr) {
      console.error("[chessleap] setFen: Chess.fromSetup failed for", fen, built.error);
      return;
    }
    chess = built.unwrap();
    lastFen = fen;
  }
  publishBoardDerivedSignals();
}

// publishBoardDerivedSignals refreshes king-square and check-square signals
// after each FEN change. The chesssquare data-class bindings read these to
// paint check + game-end king overlays without needing per-square server data.
function publishBoardDerivedSignals() {
  if (!chess) return;
  const whiteKingSq = firstSquareOf("king", "white");
  const blackKingSq = firstSquareOf("king", "black");
  // checkSquare points at the king currently in check (only meaningful while
  // the game is ongoing — the game-end branch in chesssquare ignores it).
  let checkSquare = NO_SQUARE;
  if (chess.isCheck()) {
    checkSquare = chess.turn === "white" ? whiteKingSq : blackKingSq;
  }
  mergePatch({ whiteKingSq, blackKingSq, checkSquare });
}

function firstSquareOf(role, color) {
  // chessops board iteration: scan 0..63 and return the first match.
  // Boards always have exactly one king per side, so this is O(64) once
  // per FEN change — cheaper than maintaining a cached lookup.
  for (let sq = 0; sq < 64; sq++) {
    const p = chess.board.get(sq);
    if (p && p.role === role && p.color === color) return sq;
  }
  return NO_SQUARE;
}

function mySideColor() {
  const role = getPath("role");
  if (role === "white") return "white";
  if (role === "black") return "black";
  return null; // spectator / solo
}

function isOurTurn() {
  const me = mySideColor();
  if (!me) return true; // solo: always our turn
  const turn = chess?.turn || "white";
  return me === turn;
}

function legalDestSquaresFrom(sq) {
  if (!chess) return [];
  const dests = chess.dests(sq);
  const piece = chess.board.get(sq);
  const out = [];
  for (const d of dests) {
    // chessops encodes castling as king-takes-friendly-rook (Chess960 style),
    // so dests(e1) for a kingside castle contains h1. Our UI (and Go engine)
    // use FIDE notation where the king's destination is g1/c1. Translate
    // here so the highlighted square is where the king actually lands.
    if (piece && piece.role === "king" && isOwnRookAt(d, piece.color)) {
      out.push(castledKingSquare(sq, d));
    } else {
      out.push(d);
    }
  }
  return out;
}

function isOwnRookAt(sq, color) {
  const p = chess.board.get(sq);
  return !!(p && p.role === "rook" && p.color === color);
}

// Given king-home `kingSq` and the rook square chessops picked as the dest,
// return the king's final landing square (kingside → file g, queenside → c).
function castledKingSquare(kingSq, rookSq) {
  const rank = kingSq >> 3;
  const kingside = (rookSq & 7) > (kingSq & 7);
  return rank * 8 + (kingside ? 6 : 2);
}

// Inverse of castledKingSquare: when the user clicks the king's 2-square
// landing spot, find the friendly rook chessops needs as the move target.
function castlingRookSquareFor(kingSq, kingDestSq) {
  const rank = kingSq >> 3;
  const kingside = (kingDestSq & 7) > (kingSq & 7);
  return rank * 8 + (kingside ? 7 : 0);
}

function isPromotion(fromSq, toSq) {
  const piece = chess.board.get(fromSq);
  if (!piece || piece.role !== "pawn") return false;
  const toRank = toSq >> 3;
  return (piece.color === "white" && toRank === 7) || (piece.color === "black" && toRank === 0);
}

// Move the piece DOM element from one square to another, removing any captured
// piece. Castling moves both king and rook; en-passant removes the captured
// pawn from its actual square (not the move's destination).
//
// `move.to` here is always FIDE notation (king lands on g1/c1 for castling).
// Caller is responsible for translating chessops's rook-square encoding.
function applyDom(fromSq, toSq, move) {
  const fromEl = document.querySelector(`#square-${fromSq}`);
  const toEl   = document.querySelector(`#square-${toSq}`);
  if (!fromEl || !toEl) return;

  const pieceSpan = fromEl.querySelector('span[role="img"]');
  const movingPiece = chess.board.get(fromSq);

  // En-passant: captured pawn sits on (toFile, fromRank).
  const fromRank = fromSq >> 3, toFile = toSq & 7;
  const isEnPassant =
    movingPiece && movingPiece.role === "pawn" &&
    (fromSq & 7) !== toFile && !chess.board.get(toSq);
  if (isEnPassant) {
    const epCapSq = fromRank * 8 + toFile;
    const epEl = document.querySelector(`#square-${epCapSq}`);
    const cap = epEl?.querySelector('span[role="img"]');
    cap?.remove();
  }

  // Standard capture: remove whatever is on the destination square first.
  // Skip for castling — the "destination" is empty by definition.
  if (!move.isCastle) {
    const captured = toEl.querySelector('span[role="img"]');
    captured?.remove();
  }

  // Move the piece span. For promotion, swap in the SVG of the chosen piece
  // (cloned from the promotion overlay button the user just clicked). We can't
  // rely on a server morph filling this in — in solo mode the move POST's
  // response body isn't parsed by datastar, so the dest square would stay
  // empty until the next full board re-render.
  if (pieceSpan) {
    if (move.promotion) {
      const btnPiece = document.querySelector(
        `[data-testid="promotion-button-${move.promotion}"] span[role="img"]`
      );
      if (btnPiece) {
        pieceSpan.innerHTML = btnPiece.innerHTML;
        pieceSpan.className = btnPiece.className;
        const label = btnPiece.getAttribute("aria-label");
        if (label) pieceSpan.setAttribute("aria-label", label);
      }
    }
    toEl.appendChild(pieceSpan);
  }

  // Castling: move the rook too.
  if (move.isCastle) {
    const rank = fromSq >> 3;
    const kingside = (toSq & 7) === 6;
    const rookFromSq = rank * 8 + (kingside ? 7 : 0);
    const rookToSq   = rank * 8 + (kingside ? 5 : 3);
    const rookFromEl = document.querySelector(`#square-${rookFromSq}`);
    const rookToEl   = document.querySelector(`#square-${rookToSq}`);
    const rookSpan   = rookFromEl?.querySelector('span[role="img"]');
    if (rookSpan && rookToEl) rookToEl.appendChild(rookSpan);
  }
}

function clearSelection() {
  mergePatch({ selectedSquare: NO_SQUARE, possibleMoves: [] });
}

function selectPiece(sq) {
  mergePatch({ selectedSquare: sq, possibleMoves: legalDestSquaresFrom(sq) });
}

function postMove(fromSq, toSq, promotionRole) {
  const params = new URLSearchParams();
  params.set("clientTsNs", String(Date.now() * 1_000_000));
  if (promotionRole) params.set("promo", String(ROLE_TO_PIECE[promotionRole]));
  fetch(`${routePrefix}/${gameId}/move/${fromSq}/${toSq}?${params.toString()}`, {
    method: "POST",
    headers: {
      "Accept": "text/event-stream",
      "Datastar-Request": "true",
    },
  }).then(async (res) => {
    if (!res.body) return;
    // Datastar's @get listener processes SSE frames automatically. Our fetch
    // is plain — we drain the response so the server can flush, but the
    // hub broadcast (which the /events SSE handles) is what actually patches
    // the board for both players. The POST response body is short and may be
    // empty; we just consume it.
    const reader = res.body.getReader();
    while (true) {
      const { done } = await reader.read();
      if (done) break;
    }
  }).catch(() => {
    // Hard fallback: if the network failed, the server will eventually push
    // the authoritative board via SSE on reconnect. Until then the optimistic
    // DOM stays put.
  });
}

function onSquareClick(sq) {
  if (!chess) return;
  if (!isOurTurn()) return;
  if (getPath("promotion")) return; // Promotion modal is open — let its buttons drive.

  const selected = getPath("selectedSquare");
  const piece = chess.board.get(sq);
  const ourColor = chess.turn;

  // No current selection: try to select if this square holds our piece.
  if (selected === undefined || selected === NO_SQUARE) {
    if (piece && piece.color === ourColor) selectPiece(sq);
    return;
  }

  // Clicking the same square toggles selection off.
  if (selected === sq) { clearSelection(); return; }

  // Clicking another of our pieces switches selection.
  if (piece && piece.color === ourColor) { selectPiece(sq); return; }

  // Otherwise this is a move attempt. Validate via chessops.
  const targets = legalDestSquaresFrom(selected);
  if (!targets.includes(sq)) { clearSelection(); return; }

  const fromSq = selected;
  const toSq = sq;

  if (isPromotion(fromSq, toSq) && getPath("promotionPiece") === NO_PIECE) {
    // Open the overlay; deciding the piece routes back through resumeAfterPromotion.
    mergePatch({ promotion: true, promotedSquare: toSq });
    return;
  }

  const promo = isPromotion(fromSq, toSq)
    ? ROLE_FROM_PIECE[getPath("promotionPiece")]
    : null;
  applyOptimistic(fromSq, toSq, promo);
}

function applyOptimistic(fromSq, toSq, promotionRole) {
  // `toSq` arrives in our UI/FIDE notation (king destination g1/c1 for
  // castling). chessops needs the rook-square form for play(); the server
  // and DOM both want the FIDE form.
  const piece = chess.board.get(fromSq);
  const isCastle = piece && piece.role === "king" &&
                   Math.abs((toSq & 7) - (fromSq & 7)) === 2;
  const chessopsTo = isCastle ? castlingRookSquareFor(fromSq, toSq) : toSq;

  applyDom(fromSq, toSq, { from: fromSq, to: toSq, promotion: promotionRole, isCastle });

  const move = { from: fromSq, to: chessopsTo };
  if (promotionRole) move.promotion = promotionRole;
  chess.play(move);
  lastFen = makeFen(chess.toSetup());
  // Local move can change king position (castling) or put the opponent in
  // check — both must reflect in the highlight signals immediately, since
  // the server's fen echo will short-circuit setFen (fen === lastFen).
  publishBoardDerivedSignals();

  clearSelection();
  postMove(fromSq, toSq, promotionRole);
}

// Called by the promotion overlay buttons after the user picks a piece.
function resumeAfterPromotion(pieceCode) {
  const role = ROLE_FROM_PIECE[pieceCode];
  const toSq = getPath("promotedSquare");
  const selectedSq = getPath("selectedSquare");
  const fromSq = (selectedSq !== undefined && selectedSq !== NO_SQUARE)
    ? selectedSq
    : findPromotionFrom(toSq);
  mergePatch({ promotion: false, promotedSquare: NO_SQUARE, promotionPiece: NO_PIECE });
  if (fromSq === undefined || fromSq === NO_SQUARE) return;
  applyOptimistic(fromSq, toSq, role);
}

// If the click handler cleared selectedSquare before the overlay appeared,
// recover the pawn's origin square from chess: it must be the pawn one rank
// behind the promotion square (or two files away for a capture).
function findPromotionFrom(toSq) {
  const toRank = toSq >> 3, toFile = toSq & 7;
  const dir = toRank === 7 ? -1 : +1;
  const candidates = [toFile, toFile - 1, toFile + 1]
    .filter((f) => f >= 0 && f < 8)
    .map((f) => (toRank + dir) * 8 + f);
  for (const sq of candidates) {
    const p = chess.board.get(sq);
    if (p && p.role === "pawn" && p.color === chess.turn) return sq;
  }
  return NO_SQUARE;
}

// Pull the initial FEN out of the nearest ancestor's data-signals JSON.
// Used as a synchronous seed so chessops is ready before datastar's effects
// finish wiring up.
function readInitialFen(root) {
  let el = root;
  while (el) {
    const raw = el.getAttribute && el.getAttribute("data-signals");
    if (raw) {
      try {
        const sig = JSON.parse(raw);
        if (typeof sig.fen === "string" && sig.fen.length > 0) return sig.fen;
      } catch { /* ignore — keep walking up */ }
    }
    el = el.parentElement;
  }
  return null;
}

function init() {
  const root = document.querySelector("#chessboard");
  if (!root) return; // not a game page — nothing to wire
  const cfg = root.dataset || {};
  gameId      = cfg.gameId || "";
  routePrefix = cfg.routePrefix || "";

  // Seed chess from the page-level data-signals attribute directly — it is
  // present synchronously on first paint, well before datastar has finished
  // wiring up its reactive store. Falling back to polling getPath('fen')
  // raced with the SSE connection on slower machines and left chess null.
  const initialFen = readInitialFen(root);
  if (initialFen) setFen(initialFen);
  else console.warn("[chessleap] init: no fen in page data-signals");

  // Re-anchor whenever the server pushes a new fen. Datastar effects only
  // subscribe to signals they actually read on first run, so we poll for
  // $fen to appear (matching initClock.js's setupEffects pattern) before
  // registering the effect.
  const armEffect = () => {
    if (getPath("fen") === undefined) { setTimeout(armEffect, 16); return; }
    effect(() => {
      const fen = getPath("fen");
      if (typeof fen === "string" && fen.length > 0) setFen(fen);
    });
  };
  armEffect();

  window.__chessleap = {
    onSquareClick,
    resumeAfterPromotion,
    // Debug-only handles. Use in DevTools to probe signal state from a
    // running page, e.g. window.__chessleap._debug.getPath('selectedSquare').
    _debug: {
      getPath,
      mergePatch,
      chess: () => chess,
      lastFen: () => lastFen,
    },
  };
}

if (document.readyState === "loading") {
  document.addEventListener("DOMContentLoaded", init, { once: true });
} else {
  init();
}
