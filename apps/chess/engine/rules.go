package engine

import "math/bits"

const NoSquare uint8 = 64

// --------------------------
// Generate pseudo-legal moves
// --------------------------
func (b *Board) GeneratePseudoLegalMoves() []Move {
	var moves []Move
	color := b.SideToMove
	opp := color ^ 1

	for p := Piece(0); p < PieceNB; p++ {
		bb := b.Pieces[color][p]
		for bb != 0 {
			sq := PopLSB(&bb)

			switch p {
			case Pawn:
				moves = append(moves, b.generatePawnMoves(sq, color, opp)...)
			case Knight:
				moves = append(moves, b.generateKnightMoves(sq, color)...)
			case Bishop:
				moves = append(moves, b.generateBishopMoves(sq, color)...)
			case Rook:
				moves = append(moves, b.generateRookMoves(sq, color)...)
			case Queen:
				moves = append(moves, b.generateQueenMoves(sq, color)...)
			case King:
				moves = append(moves, b.generateKingMoves(sq, color)...)
			}
		}
	}

	return moves
}

// --------------------------
// Pawn moves
// --------------------------
func (b *Board) generatePawnMoves(sq uint8, color, opp Color) []Move {
	var moves []Move
	forward := int8(8)
	startRank := uint8(1)
	promoRank := uint8(7)

	if color == Black {
		forward = -8
		startRank = 6
		promoRank = 0
	}

	// Single forward
	to := int(sq) + int(forward)
	if to >= 0 && to < 64 && (b.All&(1<<to)) == 0 {
		if uint8(to/8) == promoRank {
			for _, promo := range []Piece{Queen, Rook, Bishop, Knight} {
				moves = append(moves, Move{From: sq, To: uint8(to), Promotion: promo})
			}
		} else {
			moves = append(moves, Move{From: sq, To: uint8(to)})
		}

		// Double move
		if sq/8 == startRank {
			to2 := int(sq) + int(forward*2)
			if to2 >= 0 && to2 < 64 && (b.All&(1<<to2)) == 0 {
				moves = append(moves, Move{From: sq, To: uint8(to2)})
			}
		}
	}

	// Captures
	caps := PawnAttacks(color, sq) & b.Occupancy[opp]
	for bb := caps; bb != 0; {
		to := PopLSB(&bb)
		if uint8(to/8) == promoRank {
			for _, promo := range []Piece{Queen, Rook, Bishop, Knight} {
				moves = append(moves, Move{From: sq, To: to, Promotion: promo})
			}
		} else {
			moves = append(moves, Move{From: sq, To: to})
		}
	}

	// En-passant
	if b.EnPassant != NoSquare {
		epSq := b.EnPassant
		if PawnAttacks(color, sq)&(1<<epSq) != 0 {
			moves = append(moves, Move{From: sq, To: epSq})
		}
	}

	return moves
}

// --------------------------
// Knight moves
// --------------------------
func (b *Board) generateKnightMoves(sq uint8, color Color) []Move {
	var moves []Move
	attacks := KnightAttacks[sq] &^ b.Occupancy[color]
	for bb := attacks; bb != 0; {
		to := PopLSB(&bb)
		moves = append(moves, Move{From: sq, To: to})
	}
	return moves
}

// --------------------------
// Bishop moves
// --------------------------
func (b *Board) generateBishopMoves(sq uint8, color Color) []Move {
	var moves []Move
	attacks := BishopAttacks(sq, b.All) &^ b.Occupancy[color]
	for bb := attacks; bb != 0; {
		to := PopLSB(&bb)
		moves = append(moves, Move{From: sq, To: to})
	}
	return moves
}

// --------------------------
// Rook moves
// --------------------------
func (b *Board) generateRookMoves(sq uint8, color Color) []Move {
	var moves []Move
	attacks := RookAttacks(sq, b.All) &^ b.Occupancy[color]
	for bb := attacks; bb != 0; {
		to := PopLSB(&bb)
		moves = append(moves, Move{From: sq, To: to})
	}
	return moves
}

// --------------------------
// Queen moves
// --------------------------
func (b *Board) generateQueenMoves(sq uint8, color Color) []Move {
	moves := b.generateRookMoves(sq, color)
	moves = append(moves, b.generateBishopMoves(sq, color)...)
	return moves
}

// --------------------------
// King moves (including castling)
// --------------------------
func (b *Board) generateKingMoves(sq uint8, color Color) []Move {
	var moves []Move
	attacks := KingAttacks[sq] &^ b.Occupancy[color]
	for bb := attacks; bb != 0; {
		to := PopLSB(&bb)
		moves = append(moves, Move{From: sq, To: to})
	}

	// Castling
	if color == White {
		if b.Castling&0b0001 != 0 { // White kingside
			if b.All&(1<<61|1<<62) == 0 {
				moves = append(moves, Move{From: sq, To: 62})
			}
		}
		if b.Castling&0b0010 != 0 { // White queenside
			if b.All&(1<<57|1<<58|1<<59) == 0 {
				moves = append(moves, Move{From: sq, To: 58})
			}
		}
	} else {
		if b.Castling&0b0100 != 0 { // Black kingside
			if b.All&(1<<5|1<<6) == 0 {
				moves = append(moves, Move{From: sq, To: 6})
			}
		}
		if b.Castling&0b1000 != 0 { // Black queenside
			if b.All&(1<<1|1<<2|1<<3) == 0 {
				moves = append(moves, Move{From: sq, To: 2})
			}
		}
	}

	return moves
}

// --------------------------
// Check detection
// --------------------------
func (b *Board) isKingInCheck(color Color) bool {
	opp := color ^ 1

	// Find king square as bitboard
	kingBB := b.Pieces[color][King]
	if kingBB == 0 {
		panic("king missing on the board")
	}

	kingSq := uint8(bits.TrailingZeros64(uint64(kingBB))) // position 0..63

	// Opponent pawn attacks
	if color == White {
		if kingBB&PawnAttacks(Black, kingSq) != 0 {
			return true
		}
	} else {
		if kingBB&PawnAttacks(White, kingSq) != 0 {
			return true
		}
	}

	// Knight attacks
	if kingBB&KnightAttacks[kingSq]&b.Pieces[opp][Knight] != 0 {
		return true
	}

	// King attacks (rare)
	if kingBB&KingAttacks[kingSq]&b.Pieces[opp][King] != 0 {
		return true
	}

	// Sliding pieces
	all := b.All

	// Rook + Queen orthogonal attacks
	if (RookAttacks(kingSq, all) & (b.Pieces[opp][Rook] | b.Pieces[opp][Queen])) != 0 {
		return true
	}

	// Bishop + Queen diagonal attacks
	if (BishopAttacks(kingSq, all) & (b.Pieces[opp][Bishop] | b.Pieces[opp][Queen])) != 0 {
		return true
	}

	return false
}

// --------------------------
// Pseudo-legal validation
// --------------------------
func (b *Board) isPseudoLegal(m Move) bool {
	moves := b.GeneratePseudoLegalMoves()
	for _, mv := range moves {
		if mv.From == m.From && mv.To == m.To && mv.Promotion == m.Promotion {
			return true
		}
	}
	return false
}
