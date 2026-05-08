package engine_test

import (
	"testing"

	"github.com/lordsonvimal/synergy/apps/chess/engine"
)

// sq converts file (0=a … 7=h) and rank (0=1st … 7=8th) to a board square index.
func sq(file, rank int) uint8 {
	return uint8(rank*8 + file)
}

type pieceEntry struct {
	color  engine.Color
	kind   engine.Piece
	square uint8
}

func buildBoard(side engine.Color, entries ...pieceEntry) *engine.Board {
	b := &engine.Board{}
	for _, e := range entries {
		b.Pieces[e.color][e.kind] |= 1 << e.square
	}
	for c := engine.Color(0); c < 2; c++ {
		for p := engine.Piece(0); p < 6; p++ {
			b.Occupancy[c] |= b.Pieces[c][p]
		}
	}
	b.All = b.Occupancy[0] | b.Occupancy[1]
	b.SideToMove = side
	b.EnPassant = engine.NoSquare
	b.FullMoveNumber = 1
	return b
}

// playMoves applies UCI moves to b, failing the test on the first illegal move.
func playMoves(t *testing.T, b *engine.Board, ucis ...string) {
	t.Helper()
	for _, uci := range ucis {
		m := engine.MoveFromUCI(uci)
		if !b.MakeMove(m) {
			t.Fatalf("move %s was rejected as illegal", uci)
		}
	}
}

// ---- Legal move generation ------------------------------------------------

func TestInitialPositionMoveCount(t *testing.T) {
	b := engine.NewBoard()
	// Perft(1) counts all legal leaf nodes at depth 1 = number of legal moves.
	// From the starting position there are exactly 20 (16 pawn + 4 knight moves).
	got := b.Perft(1)
	if got != 20 {
		t.Errorf("starting position: got %d legal moves, want 20", got)
	}
}

func TestPerftDepth2(t *testing.T) {
	b := engine.NewBoard()
	got := b.Perft(2)
	if got != 400 {
		t.Errorf("perft(2): got %d, want 400", got)
	}
}

// ---- Check detection ------------------------------------------------------

func TestKingInCheck(t *testing.T) {
	// White rook on e8 gives check to the black king also on e8 — impossible
	// physically, so use: black king on e8, white queen on e5, nothing in between.
	// Queen on e5 attacks along the e-file → e6, e7, e8. King is in check.
	b := buildBoard(engine.Black,
		pieceEntry{engine.White, engine.King, sq(0, 0)},  // Ka1
		pieceEntry{engine.White, engine.Queen, sq(4, 4)}, // Qe5
		pieceEntry{engine.Black, engine.King, sq(4, 7)},  // Ke8
	)
	if !b.IsKingInCheck(engine.Black) {
		t.Error("black king on e8 should be in check from white queen on e5 (e-file)")
	}
}

func TestKingNotInCheck(t *testing.T) {
	// Starting position: no king is in check.
	b := engine.NewBoard()
	if b.IsKingInCheck(engine.White) {
		t.Error("white king should not be in check in the starting position")
	}
	if b.IsKingInCheck(engine.Black) {
		t.Error("black king should not be in check in the starting position")
	}
}

// ---- Checkmate detection --------------------------------------------------

// scholarsMate applies the classic four-move Scholar's Mate.
// 1.e4 e5  2.Bc4 Nc6  3.Qh5 Nf6??  4.Qxf7#
func scholarsMate(t *testing.T) *engine.Board {
	t.Helper()
	b := engine.NewBoard()
	playMoves(t, b,
		"e2e4", "e7e5",
		"f1c4", "b8c6",
		"d1h5", "g8f6",
		"h5f7",
	)
	return b
}

func TestCheckmateDetection(t *testing.T) {
	b := scholarsMate(t)
	// After Qxf7# it is Black's turn; Black is in check and has no legal moves.
	if !b.IsKingInCheck(engine.Black) {
		t.Error("black king must be in check after Scholar's mate")
	}
	if b.HasLegalMoves(engine.Black) {
		t.Error("black must have no legal moves after Scholar's mate")
	}
}

func TestNotCheckmateWhenHasMoves(t *testing.T) {
	b := engine.NewBoard()
	// White is to move in the starting position and has 20 legal moves — not checkmate.
	// HasLegalMoves must be called with the board's SideToMove colour.
	if !b.HasLegalMoves(engine.White) {
		t.Error("white should have legal moves in the starting position")
	}
}

// ---- Stalemate detection --------------------------------------------------

func TestStalemateDetection(t *testing.T) {
	// Classic two-piece stalemate:
	//   White Kb6 + Qe5  vs  Black Ka8   (Black to move)
	//
	// Black king on a8 can only go to a7, b7, b8:
	//   a7 (48) — attacked by white king on b6 (adjacent diagonally)
	//   b7 (49) — attacked by white king on b6 (adjacent vertically)
	//   b8 (57) — attacked by white queen on e5 (e5–d6–c7–b8 diagonal)
	// Black king is NOT in check (e5 queen does not attack a8).
	b := buildBoard(engine.Black,
		pieceEntry{engine.White, engine.King, sq(1, 5)},  // Kb6
		pieceEntry{engine.White, engine.Queen, sq(4, 4)}, // Qe5
		pieceEntry{engine.Black, engine.King, sq(0, 7)},  // Ka8
	)
	if b.IsKingInCheck(engine.Black) {
		t.Fatal("black king on a8 must NOT be in check (precondition for stalemate)")
	}
	if b.HasLegalMoves(engine.Black) {
		t.Error("black should have no legal moves (stalemate position)")
	}
}

func TestStalemateIsNotCheck(t *testing.T) {
	b := buildBoard(engine.Black,
		pieceEntry{engine.White, engine.King, sq(1, 5)},
		pieceEntry{engine.White, engine.Queen, sq(4, 4)},
		pieceEntry{engine.Black, engine.King, sq(0, 7)},
	)
	// Stalemate requires the king NOT to be in check.
	if b.IsKingInCheck(engine.Black) {
		t.Error("stalemate position must not have the king in check")
	}
}

// ---- Castling -------------------------------------------------------------

func TestCastlingKingside(t *testing.T) {
	// Clear white's kingside pieces by playing normal opening moves:
	// 1.e4  2.Nf3  3.Bc4  then castle O-O (e1g1).
	b := engine.NewBoard()
	playMoves(t, b, "e2e4", "e7e5", "g1f3", "b8c6", "f1c4", "a7a6")
	// e1g1 is kingside castling
	m := engine.MoveFromUCI("e1g1")
	if !b.MakeMove(m) {
		t.Fatal("white kingside castling (O-O) should be legal after clearing f1 and g1")
	}
	_, _, kingOnG1 := b.PieceAt(sq(6, 0)) // g1
	if !kingOnG1 {
		t.Error("white king should be on g1 after castling kingside")
	}
	_, _, rookOnF1 := b.PieceAt(sq(5, 0)) // f1
	if !rookOnF1 {
		t.Error("white rook should be on f1 after castling kingside")
	}
}

func TestCastlingQueenside(t *testing.T) {
	// Clear queenside: move d-pawn, queen, bishop, and knight.
	b := engine.NewBoard()
	playMoves(t, b,
		"d2d4", "d7d5",
		"d1d3", "c8g4",   // queen and dark-squared bishop clear
		"c1e3", "b8c6",   // light-squared bishop clears
		"b1c3", "a7a6",   // knight clears
	)
	// Now b1, c1, d1 are all empty — e1a1 (queenside castle)
	m := engine.MoveFromUCI("e1c1")
	if !b.MakeMove(m) {
		t.Fatal("white queenside castling (O-O-O) should be legal after clearing b1/c1/d1")
	}
	_, _, kingOnC1 := b.PieceAt(sq(2, 0)) // c1
	if !kingOnC1 {
		t.Error("white king should be on c1 after castling queenside")
	}
	_, _, rookOnD1 := b.PieceAt(sq(3, 0)) // d1
	if !rookOnD1 {
		t.Error("white rook should be on d1 after castling queenside")
	}
}

func TestCastlingRightsRevokedAfterKingMove(t *testing.T) {
	// Move the white king off e1 and back; castling rights should be gone.
	b := engine.NewBoard()
	// Clear e2 pawn and f1/g1 pieces so king has room to manoeuvre.
	playMoves(t, b, "e2e4", "e7e5", "g1f3", "b8c6", "f1e2", "a7a6")
	// Move king: e1→f1 (not castling, just a king move)
	if !b.MakeMove(engine.MoveFromUCI("e1f1")) {
		t.Fatal("king move e1→f1 should be legal")
	}
	// Black placeholder
	playMoves(t, b, "a6a5")
	// King back to e1
	if !b.MakeMove(engine.MoveFromUCI("f1e1")) {
		t.Fatal("king move f1→e1 should be legal")
	}
	playMoves(t, b, "a5a4")

	// Kingside castling bit (0b0001) should be cleared after king moved.
	if b.Castling&0b0001 != 0 {
		t.Error("white kingside castling right should be revoked after the king moved")
	}
	if b.Castling&0b0010 != 0 {
		t.Error("white queenside castling right should be revoked after the king moved")
	}
}

// ---- En passant -----------------------------------------------------------

func TestEnPassantCapture(t *testing.T) {
	// 1.e4 a6  2.e5 d5  → en-passant target is d6.
	// White plays e5d6 (en-passant capture).
	b := engine.NewBoard()
	playMoves(t, b, "e2e4", "a7a6", "e4e5", "d7d5")

	epTarget := sq(3, 5) // d6
	if b.EnPassant != epTarget {
		t.Fatalf("en-passant square should be d6 (%d) after d7d5, got %d", epTarget, b.EnPassant)
	}

	// Execute en passant: e5→d6
	if !b.MakeMove(engine.MoveFromUCI("e5d6")) {
		t.Fatal("en-passant capture e5d6 should be legal")
	}

	// White pawn must now be on d6
	_, _, pawnOnD6 := b.PieceAt(epTarget)
	if !pawnOnD6 {
		t.Error("white pawn should be on d6 after en-passant capture")
	}

	// Captured black pawn on d5 must be gone
	_, _, pieceOnD5 := b.PieceAt(sq(3, 4))
	if pieceOnD5 {
		t.Error("black pawn on d5 should have been captured by en-passant")
	}
}

// ---- Promotion ------------------------------------------------------------

func TestPromotionToQueen(t *testing.T) {
	// White pawn on a7, kings out of the way.
	b := buildBoard(engine.White,
		pieceEntry{engine.White, engine.King, sq(4, 0)},  // Ke1
		pieceEntry{engine.White, engine.Pawn, sq(0, 6)},  // Pa7
		pieceEntry{engine.Black, engine.King, sq(7, 7)},  // Kh8
	)
	b.Castling = 0

	// Promote pawn: a7a8q
	if !b.MakeMove(engine.MoveFromUCI("a7a8q")) {
		t.Fatal("pawn promotion a7a8=Q should be legal")
	}

	c, p, ok := b.PieceAt(sq(0, 7)) // a8
	if !ok || c != engine.White || p != engine.Queen {
		t.Errorf("white queen expected on a8 after promotion, got color=%v piece=%v ok=%v", c, p, ok)
	}
	// Original pawn square must be empty
	_, _, stillPawn := b.PieceAt(sq(0, 6))
	if stillPawn {
		t.Error("pawn should be gone from a7 after promotion")
	}
}

func TestPromotionToKnight(t *testing.T) {
	b := buildBoard(engine.White,
		pieceEntry{engine.White, engine.King, sq(4, 0)},
		pieceEntry{engine.White, engine.Pawn, sq(0, 6)},
		pieceEntry{engine.Black, engine.King, sq(7, 7)},
	)
	b.Castling = 0

	if !b.MakeMove(engine.MoveFromUCI("a7a8n")) {
		t.Fatal("pawn promotion a7a8=N should be legal")
	}

	c, p, ok := b.PieceAt(sq(0, 7))
	if !ok || c != engine.White || p != engine.Knight {
		t.Errorf("white knight expected on a8 after promotion, got color=%v piece=%v ok=%v", c, p, ok)
	}
}

// ---- Move undo ------------------------------------------------------------

func TestMoveUndo(t *testing.T) {
	b := engine.NewBoard()
	fenBefore := b.FEN()

	// Apply a move then undo it — board should return to the exact same FEN.
	playMoves(t, b, "e2e4")
	b.UnapplyMove()

	fenAfter := b.FEN()
	if fenBefore != fenAfter {
		t.Errorf("FEN mismatch after undo:\n  before: %s\n  after:  %s", fenBefore, fenAfter)
	}
}

func TestMultipleMoveUndo(t *testing.T) {
	b := engine.NewBoard()
	fenStart := b.FEN()

	playMoves(t, b, "e2e4", "e7e5", "g1f3")
	// Undo all three moves
	b.UnapplyMove()
	b.UnapplyMove()
	b.UnapplyMove()

	if b.FEN() != fenStart {
		t.Errorf("FEN after undoing 3 moves does not match starting position")
	}
}
