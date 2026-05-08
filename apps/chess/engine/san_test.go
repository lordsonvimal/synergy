package engine_test

import (
	"testing"

	"github.com/lordsonvimal/synergy/apps/chess/engine"
)

// applyMoves applies a sequence of UCI move strings to the board, panicking on
// any rejected move so test failures are obvious.
func applyMoves(t *testing.T, b *engine.Board, ucis ...string) {
	t.Helper()
	for _, uci := range ucis {
		if !b.MakeMove(engine.MoveFromUCI(uci)) {
			t.Fatalf("MakeMove(%q) rejected unexpectedly", uci)
		}
	}
}

// ---- Pawn moves ----------------------------------------------------------------

func TestSANPawnPush(t *testing.T) {
	b := engine.NewBoard()
	got := b.SAN(engine.MoveFromUCI("e2e4"))
	if got != "e4" {
		t.Errorf("want 'e4', got %q", got)
	}
}

func TestSANPawnCapture(t *testing.T) {
	b := engine.NewBoard()
	applyMoves(t, b, "e2e4", "d7d5")
	got := b.SAN(engine.MoveFromUCI("e4d5"))
	if got != "exd5" {
		t.Errorf("want 'exd5', got %q", got)
	}
}

func TestSANEnPassantCapture(t *testing.T) {
	b := engine.NewBoard()
	// 1.e4 e5 2.e5 d5 (advances White pawn to e5 then Black pawn to d5)
	// 3.exd6 is the en-passant capture
	applyMoves(t, b, "e2e4", "e7e5", "e4e5", "d7d5")
	got := b.SAN(engine.MoveFromUCI("e5d6"))
	if got != "exd6" {
		t.Errorf("want 'exd6', got %q", got)
	}
}

func TestSANPawnPromotion(t *testing.T) {
	b := &engine.Board{}
	// White: Ke1, pawn on e7; Black: Ka1 — not on e-file, 8th rank, or e8 diagonal.
	b.Pieces[engine.White][engine.King] = 1 << 4  // Ke1
	b.Pieces[engine.White][engine.Pawn] = 1 << 52 // e7
	b.Pieces[engine.Black][engine.King] = 1 << 0  // Ka1
	b.Occupancy[engine.White] = b.Pieces[engine.White][engine.King] | b.Pieces[engine.White][engine.Pawn]
	b.Occupancy[engine.Black] = b.Pieces[engine.Black][engine.King]
	b.All = b.Occupancy[engine.White] | b.Occupancy[engine.Black]
	b.SideToMove = engine.White
	b.EnPassant = engine.NoSquare

	got := b.SAN(engine.Move{From: 52, To: 60, Promotion: engine.Queen})
	if got != "e8=Q" {
		t.Errorf("want 'e8=Q', got %q", got)
	}
}

// ---- Piece moves ----------------------------------------------------------------

func TestSANKnightMove(t *testing.T) {
	b := engine.NewBoard()
	got := b.SAN(engine.MoveFromUCI("g1f3"))
	if got != "Nf3" {
		t.Errorf("want 'Nf3', got %q", got)
	}
}

func TestSANBishopMove(t *testing.T) {
	b := engine.NewBoard()
	applyMoves(t, b, "e2e4", "e7e5")
	got := b.SAN(engine.MoveFromUCI("f1c4"))
	if got != "Bc4" {
		t.Errorf("want 'Bc4', got %q", got)
	}
}

// ---- Disambiguation ----------------------------------------------------------------

func TestSANDisambiguationByFile(t *testing.T) {
	// Nb1 and Nd1 can both move to c3 — disambiguate by source file.
	b := &engine.Board{}
	b.Pieces[engine.White][engine.King] = 1 << 4          // Ke1
	b.Pieces[engine.White][engine.Knight] = (1 << 1) | (1 << 3) // Nb1, Nd1
	b.Pieces[engine.Black][engine.King] = 1 << 60         // Ke8
	b.Occupancy[engine.White] = b.Pieces[engine.White][engine.King] | b.Pieces[engine.White][engine.Knight]
	b.Occupancy[engine.Black] = b.Pieces[engine.Black][engine.King]
	b.All = b.Occupancy[engine.White] | b.Occupancy[engine.Black]
	b.SideToMove = engine.White
	b.Castling = 0
	b.EnPassant = engine.NoSquare

	// Nb1→c3 needs file 'b' to distinguish from Nd1→c3
	got := b.SAN(engine.MoveFromUCI("b1c3"))
	if got != "Nbc3" {
		t.Errorf("Nb1→c3: want 'Nbc3', got %q", got)
	}
}

func TestSANDisambiguationByRank(t *testing.T) {
	// Ra1 and Ra5 can both move to a3 — disambiguate by source rank.
	b := &engine.Board{}
	b.Pieces[engine.White][engine.King] = 1 << 4          // Ke1
	b.Pieces[engine.White][engine.Rook] = (1 << 0) | (1 << 32) // Ra1, Ra5
	b.Pieces[engine.Black][engine.King] = 1 << 60         // Ke8
	b.Occupancy[engine.White] = b.Pieces[engine.White][engine.King] | b.Pieces[engine.White][engine.Rook]
	b.Occupancy[engine.Black] = b.Pieces[engine.Black][engine.King]
	b.All = b.Occupancy[engine.White] | b.Occupancy[engine.Black]
	b.SideToMove = engine.White
	b.Castling = 0
	b.EnPassant = engine.NoSquare

	// Ra1→a3 needs rank '1' to distinguish from Ra5→a3
	// MoveFromUCI("a1a3") = from sq 0, to sq 16
	got := b.SAN(engine.MoveFromUCI("a1a3"))
	if got != "R1a3" {
		t.Errorf("Ra1→a3: want 'R1a3', got %q", got)
	}
}

// ---- Castling ----------------------------------------------------------------

func TestSANCastlingKingside(t *testing.T) {
	b := &engine.Board{}
	b.Pieces[engine.White][engine.King] = 1 << 4 // Ke1
	b.Pieces[engine.White][engine.Rook] = 1 << 7 // Rh1
	b.Pieces[engine.Black][engine.King] = 1 << 60 // Ke8
	b.Occupancy[engine.White] = b.Pieces[engine.White][engine.King] | b.Pieces[engine.White][engine.Rook]
	b.Occupancy[engine.Black] = b.Pieces[engine.Black][engine.King]
	b.All = b.Occupancy[engine.White] | b.Occupancy[engine.Black]
	b.SideToMove = engine.White
	b.Castling = 0b0001 // White kingside
	b.EnPassant = engine.NoSquare

	got := b.SAN(engine.MoveFromUCI("e1g1"))
	if got != "O-O" {
		t.Errorf("want 'O-O', got %q", got)
	}
}

func TestSANCastlingQueenside(t *testing.T) {
	b := &engine.Board{}
	b.Pieces[engine.White][engine.King] = 1 << 4 // Ke1
	b.Pieces[engine.White][engine.Rook] = 1 << 0 // Ra1
	b.Pieces[engine.Black][engine.King] = 1 << 60 // Ke8
	b.Occupancy[engine.White] = b.Pieces[engine.White][engine.King] | b.Pieces[engine.White][engine.Rook]
	b.Occupancy[engine.Black] = b.Pieces[engine.Black][engine.King]
	b.All = b.Occupancy[engine.White] | b.Occupancy[engine.Black]
	b.SideToMove = engine.White
	b.Castling = 0b0010 // White queenside
	b.EnPassant = engine.NoSquare

	got := b.SAN(engine.MoveFromUCI("e1c1"))
	if got != "O-O-O" {
		t.Errorf("want 'O-O-O', got %q", got)
	}
}

// ---- Check and checkmate -------------------------------------------------------

func TestSANCheck(t *testing.T) {
	// White Ke1, Qd1, Black Ke8 — Qd8 gives check.
	b := &engine.Board{}
	b.Pieces[engine.White][engine.King] = 1 << 4  // Ke1
	b.Pieces[engine.White][engine.Queen] = 1 << 3 // Qd1
	b.Pieces[engine.Black][engine.King] = 1 << 60 // Ke8
	b.Occupancy[engine.White] = b.Pieces[engine.White][engine.King] | b.Pieces[engine.White][engine.Queen]
	b.Occupancy[engine.Black] = b.Pieces[engine.Black][engine.King]
	b.All = b.Occupancy[engine.White] | b.Occupancy[engine.Black]
	b.SideToMove = engine.White
	b.Castling = 0
	b.EnPassant = engine.NoSquare

	// Qd1→d8: queen moves along d-file, checks Ke8 diagonally from d8 to e8? No —
	// Ke8 (e8 = sq 60) and Qd8 (d8 = sq 59): same rank, adjacent file → in check.
	got := b.SAN(engine.MoveFromUCI("d1d8"))
	if got != "Qd8+" {
		t.Errorf("want 'Qd8+', got %q", got)
	}
}

func TestSANScholarsMateCheckmate(t *testing.T) {
	b := engine.NewBoard()
	applyMoves(t, b, "e2e4", "e7e5", "f1c4", "b8c6", "d1h5", "g8f6")
	got := b.SAN(engine.MoveFromUCI("h5f7"))
	if got != "Qxf7#" {
		t.Errorf("want 'Qxf7#', got %q", got)
	}
}
