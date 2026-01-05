package engine

import (
	"math/bits"
)

const NoSquare uint8 = 64

func (b *Board) squareAttacked(sq uint8, by Color) bool {
	// Pawn
	if PawnAttacks(by^1, sq)&b.Pieces[by][Pawn] != 0 {
		return true
	}

	// Knight
	if KnightAttacks[sq]&b.Pieces[by][Knight] != 0 {
		return true
	}

	// King
	if KingAttacks[sq]&b.Pieces[by][King] != 0 {
		return true
	}

	// Sliding
	if RookAttacks(sq, b.All)&(b.Pieces[by][Rook]|b.Pieces[by][Queen]) != 0 {
		return true
	}
	if BishopAttacks(sq, b.All)&(b.Pieces[by][Bishop]|b.Pieces[by][Queen]) != 0 {
		return true
	}

	return false
}

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
		flag := MoveNormal
		if uint8(to/8) == promoRank {
			for _, promo := range []Piece{Queen, Rook, Bishop, Knight} {
				moves = append(moves, Move{From: sq, To: uint8(to), Promotion: promo, Flags: MovePromo})
			}
		} else {
			moves = append(moves, Move{From: sq, To: uint8(to), Flags: uint8(flag)})
		}

		// Double move
		if sq/8 == startRank {
			to2 := int(sq) + int(forward*2)
			if to2 >= 0 && to2 < 64 && (b.All&(1<<to2)) == 0 {
				moves = append(moves, Move{From: sq, To: uint8(to2), Flags: MoveNormal})
			}
		}
	}

	// Captures
	caps := PawnAttacks(color, sq) & b.Occupancy[opp]
	for bb := caps; bb != 0; {
		to := PopLSB(&bb)
		flag := MoveCapture
		if uint8(to/8) == promoRank {
			for _, promo := range []Piece{Queen, Rook, Bishop, Knight} {
				moves = append(moves, Move{From: sq, To: to, Promotion: promo, Flags: MoveCapture | MovePromo})
			}
		} else {
			moves = append(moves, Move{From: sq, To: to, Flags: uint8(flag)})
		}
	}

	// En-passant
	if b.EnPassant != NoSquare {
		epSq := b.EnPassant
		if PawnAttacks(color, sq)&(1<<epSq) != 0 {
			moves = append(moves, Move{From: sq, To: epSq, Flags: MoveEP})
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

		flag := MoveNormal
		if (b.Occupancy[color^1] & (1 << to)) != 0 {
			flag |= MoveCapture
		}

		moves = append(moves, Move{
			From:  sq,
			To:    to,
			Flags: uint8(flag),
		})
	}

	return moves
}

// --------------------------
// Bishop moves
// --------------------------
func (b *Board) generateBishopMoves(sq uint8, color Color) []Move {
	var moves []Move
	occ := b.All &^ (uint64(1) << sq)
	attacks := BishopAttacks(sq, occ) &^ b.Occupancy[color]
	PrintBB(attacks)
	for bb := attacks; bb != 0; {
		to := PopLSB(&bb)

		flag := MoveNormal
		if (b.Occupancy[color^1] & (1 << to)) != 0 {
			flag |= MoveCapture
		}

		moves = append(moves, Move{
			From:  sq,
			To:    to,
			Flags: uint8(flag),
		})
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

		flag := MoveNormal
		if (b.Occupancy[color^1] & (1 << to)) != 0 {
			flag |= MoveCapture
		}

		moves = append(moves, Move{
			From:  sq,
			To:    to,
			Flags: uint8(flag),
		})
	}

	return moves
}

// --------------------------
// Queen moves
// --------------------------
func (b *Board) generateQueenMoves(sq uint8, color Color) []Move {
	rMoves := b.generateRookMoves(sq, color)
	bMoves := b.generateBishopMoves(sq, color)

	moves := make([]Move, 0, len(rMoves)+len(bMoves))
	moves = append(moves, rMoves...)
	moves = append(moves, bMoves...)
	return moves
}

// --------------------------
// King moves (including castling)
// --------------------------
func (b *Board) generateKingMoves(sq uint8, color Color) []Move {
	var moves []Move
	opp := color ^ 1

	// -----------------
	// Normal king moves
	// -----------------
	attacks := KingAttacks[sq] &^ b.Occupancy[color]
	for bb := attacks; bb != 0; {
		to := PopLSB(&bb)

		flag := MoveNormal
		if b.Occupancy[opp]&(1<<to) != 0 {
			flag |= MoveCapture
		}

		// Skip squares under attack
		if !b.squareAttacked(to, opp) {
			moves = append(moves, Move{
				From:  sq,
				To:    to,
				Flags: uint8(flag),
			})
		}
	}

	// -----------------
	// Castling
	// -----------------
	if color == White {
		// White King is on sq 4 (e1)
		// King side: e1 -> g1 (4 -> 6)
		if b.Castling&0b0001 != 0 &&
			b.All&(bit(5)|bit(6)) == 0 &&
			!b.squareAttacked(4, Black) &&
			!b.squareAttacked(5, Black) &&
			!b.squareAttacked(6, Black) {
			moves = append(moves, Move{From: 4, To: 6, Flags: MoveCastle})
		}

		// Queen side: e1 -> c1 (4 -> 2)
		if b.Castling&0b0010 != 0 &&
			b.All&(bit(1)|bit(2)|bit(3)) == 0 &&
			!b.squareAttacked(4, Black) &&
			!b.squareAttacked(3, Black) &&
			!b.squareAttacked(2, Black) {
			moves = append(moves, Move{From: 4, To: 2, Flags: MoveCastle})
		}
	} else {
		// Black King is on sq 60 (e8)
		// King side: e8 -> g8 (60 -> 62)
		if b.Castling&0b0100 != 0 &&
			b.All&(bit(61)|bit(62)) == 0 &&
			!b.squareAttacked(60, White) &&
			!b.squareAttacked(61, White) &&
			!b.squareAttacked(62, White) {
			moves = append(moves, Move{From: 60, To: 62, Flags: MoveCastle})
		}

		// Queen side: e8 -> c8 (60 -> 58)
		if b.Castling&0b1000 != 0 &&
			b.All&(bit(57)|bit(58)|bit(59)) == 0 &&
			!b.squareAttacked(60, White) &&
			!b.squareAttacked(59, White) &&
			!b.squareAttacked(58, White) {
			moves = append(moves, Move{From: 60, To: 58, Flags: MoveCastle})
		}
	}

	return moves
}

// --------------------------
// Check detection
// --------------------------
func (b *Board) IsKingInCheck(color Color) bool {
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
// Generate all legal moves for a given square
// --------------------------
func (b *Board) GenerateMovesForSquare(sq uint8) []Move {
	color, piece, ok := b.PieceAt(sq)
	if !ok || color != b.SideToMove {
		return nil // no piece or not this side's turn
	}

	var moves []Move

	switch piece {
	case Pawn:
		opp := color ^ 1
		moves = b.generatePawnMoves(sq, color, opp)
	case Knight:
		moves = b.generateKnightMoves(sq, color)
	case Bishop:
		moves = b.generateBishopMoves(sq, color)
	case Rook:
		moves = b.generateRookMoves(sq, color)
	case Queen:
		moves = b.generateQueenMoves(sq, color)
	case King:
		moves = b.generateKingMoves(sq, color)
	}

	// Filter moves to only legal ones
	legalMoves := []Move{}
	for _, m := range moves {
		if b.TryMove(m) {
			legalMoves = append(legalMoves, m)
		}
	}

	return legalMoves
}

func (b *Board) HasLegalMoves(color Color) bool {
	for sq := uint8(0); sq < 64; sq++ {
		c, p, ok := b.PieceAt(sq)
		if !ok || c != color {
			continue
		}
		// generate pseudo-legal moves for this piece
		var moves []Move
		switch p {
		case Pawn:
			moves = b.generatePawnMoves(sq, color, color^1)
		case Knight:
			moves = b.generateKnightMoves(sq, color)
		case Bishop:
			moves = b.generateBishopMoves(sq, color)
		case Rook:
			moves = b.generateRookMoves(sq, color)
		case Queen:
			moves = b.generateQueenMoves(sq, color)
		case King:
			moves = b.generateKingMoves(sq, color)
		}

		for _, m := range moves {
			if b.TryMove(m) {
				return true // found at least one legal move
			}
		}
	}
	return false
}

func (b *Board) GenerateCaptures() []Move {
	var moves []Move

	color := b.SideToMove
	opp := color ^ 1
	occOpp := b.Occupancy[opp]

	// --------------------------
	// Pawns
	// --------------------------
	pawns := b.Pieces[color][Pawn]
	for pawns != 0 {
		sq := PopLSB(&pawns)

		attacks := PawnAttacks(color, sq) & occOpp
		for bb := attacks; bb != 0; {
			to := PopLSB(&bb)

			// Promotion captures
			promoRank := uint8(7)
			if color == Black {
				promoRank = 0
			}

			if to/8 == promoRank {
				for _, promo := range []Piece{Queen, Rook, Bishop, Knight} {
					moves = append(moves, Move{
						From:      sq,
						To:        to,
						Promotion: promo,
						Flags:     MoveCapture | MovePromo,
					})
				}
			} else {
				moves = append(moves, Move{
					From:  sq,
					To:    to,
					Flags: MoveCapture,
				})
			}
		}

		// En-passant capture
		if b.EnPassant != NoSquare {
			ep := b.EnPassant
			if PawnAttacks(color, sq)&(1<<ep) != 0 {
				moves = append(moves, Move{
					From:  sq,
					To:    ep,
					Flags: MoveEP | MoveCapture,
				})
			}
		}
	}

	// --------------------------
	// Knights
	// --------------------------
	knights := b.Pieces[color][Knight]
	for knights != 0 {
		sq := PopLSB(&knights)
		attacks := KnightAttacks[sq] & occOpp

		for bb := attacks; bb != 0; {
			to := PopLSB(&bb)
			moves = append(moves, Move{
				From:  sq,
				To:    to,
				Flags: MoveCapture,
			})
		}
	}

	// --------------------------
	// Bishops
	// --------------------------
	bishops := b.Pieces[color][Bishop]
	for bishops != 0 {
		sq := PopLSB(&bishops)
		attacks := BishopAttacks(sq, b.All) & occOpp

		for bb := attacks; bb != 0; {
			to := PopLSB(&bb)
			moves = append(moves, Move{
				From:  sq,
				To:    to,
				Flags: MoveCapture,
			})
		}
	}

	// --------------------------
	// Rooks
	// --------------------------
	rooks := b.Pieces[color][Rook]
	for rooks != 0 {
		sq := PopLSB(&rooks)
		attacks := RookAttacks(sq, b.All) & occOpp

		for bb := attacks; bb != 0; {
			to := PopLSB(&bb)
			moves = append(moves, Move{
				From:  sq,
				To:    to,
				Flags: MoveCapture,
			})
		}
	}

	// --------------------------
	// Queens
	// --------------------------
	queens := b.Pieces[color][Queen]
	for queens != 0 {
		sq := PopLSB(&queens)
		attacks := (RookAttacks(sq, b.All) | BishopAttacks(sq, b.All)) & occOpp

		for bb := attacks; bb != 0; {
			to := PopLSB(&bb)
			moves = append(moves, Move{
				From:  sq,
				To:    to,
				Flags: MoveCapture,
			})
		}
	}

	// --------------------------
	// King (captures only, no castling)
	// --------------------------
	king := b.Pieces[color][King]
	if king != 0 {
		sq := uint8(bits.TrailingZeros64(king))
		attacks := KingAttacks[sq] & occOpp

		for bb := attacks; bb != 0; {
			to := PopLSB(&bb)

			// Skip squares under attack
			if b.squareAttacked(to, opp) {
				continue
			}

			moves = append(moves, Move{
				From:  sq,
				To:    to,
				Flags: MoveCapture,
			})
		}
	}

	return moves
}

// --------------------------
// Pseudo-legal validation for all moves
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
