package engine

import "math/bits"

// --------------------------
// Generate pseudo-legal moves
// --------------------------
func (b *Board) GeneratePseudoLegalMoves() []Move {
	var moves []Move

	color := b.SideToMove

	// Iterate all pieces for side to move
	for p := Piece(0); p < PieceNB; p++ {
		bb := b.Pieces[color][p]
		for bb != 0 {
			sq := PopLSB((*Bitboard)(&bb))

			switch p {
			case Pawn:
				moves = append(moves, b.pawnMoves(sq, color)...)
			case Knight:
				moves = append(moves, b.knightMoves(sq, color)...)
			case Bishop:
				moves = append(moves, b.bishopMoves(sq, color)...)
			case Rook:
				moves = append(moves, b.rookMoves(sq, color)...)
			case Queen:
				moves = append(moves, b.queenMoves(sq, color)...)
			case King:
				moves = append(moves, b.kingMoves(sq, color)...)
			}
		}
	}

	return moves
}

// --------------------------
// Move generation for pieces
// --------------------------

func (b *Board) pawnMoves(sq uint8, color Color) []Move {
	var moves []Move
	attacks := PawnAttacks(color, sq)

	// Capture moves
	for target := uint8(0); target < 64; target++ {
		if attacks&Bitboard(1<<target) != 0 {
			if b.Occupancy[color^1]&Bitboard(1<<target) != 0 {
				moves = append(moves, Move{From: sq, To: target, Promotion: NoPiece})
				// Promotions
				rank := target / 8
				if rank == 0 || rank == 7 {
					for _, promo := range []Piece{Queen, Rook, Bishop, Knight} {
						moves = append(moves, Move{From: sq, To: target, Promotion: promo})
					}
				}
			}
		}
	}

	// Forward moves
	var forward uint8
	if color == White {
		forward = sq + 8
		if forward < 64 && (b.All&Bitboard(1<<forward)) == 0 {
			moves = append(moves, Move{From: sq, To: forward, Promotion: NoPiece})
			if sq/8 == 1 { // double move
				double := sq + 16
				if (b.All & Bitboard(1<<double)) == 0 {
					moves = append(moves, Move{From: sq, To: double, Promotion: NoPiece})
				}
			}
		}
	} else {
		forward = sq - 8
		if forward < 64 && (b.All&Bitboard(1<<forward)) == 0 {
			moves = append(moves, Move{From: sq, To: forward, Promotion: NoPiece})
			if sq/8 == 6 { // double move
				double := sq - 16
				if (b.All & Bitboard(1<<double)) == 0 {
					moves = append(moves, Move{From: sq, To: double, Promotion: NoPiece})
				}
			}
		}
	}

	return moves
}

func (b *Board) knightMoves(sq uint8, color Color) []Move {
	var moves []Move
	attacks := KnightAttacks[sq] & ^b.Occupancy[color]
	for bb := attacks; bb != 0; {
		to := PopLSB((*Bitboard)(&bb))
		moves = append(moves, Move{From: sq, To: to, Promotion: NoPiece})
	}
	return moves
}

func (b *Board) kingMoves(sq uint8, color Color) []Move {
	var moves []Move
	attacks := KingAttacks[sq] & ^b.Occupancy[color]
	for bb := attacks; bb != 0; {
		to := PopLSB((*Bitboard)(&bb))
		moves = append(moves, Move{From: sq, To: to, Promotion: NoPiece})
	}
	// Castling placeholder (implement later)
	return moves
}

func (b *Board) rookMoves(sq uint8, color Color) []Move {
	// TODO: sliding attack with occupancy (rook)
	return []Move{}
}

func (b *Board) bishopMoves(sq uint8, color Color) []Move {
	// TODO: sliding attack with occupancy (bishop)
	return []Move{}
}

func (b *Board) queenMoves(sq uint8, color Color) []Move {
	// Queen = rook + bishop
	moves := b.rookMoves(sq, color)
	moves = append(moves, b.bishopMoves(sq, color)...)
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
