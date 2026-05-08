package engine

import "math/bits"

// SAN returns the Standard Algebraic Notation string for m.
// b must be in the position BEFORE m is applied.
// m.Flags need not be set; all special-move types are derived from board state.
func (b *Board) SAN(m Move) string {
	color := b.SideToMove
	opp := color ^ 1
	piece := b.pieceOnSquare(m.From)

	// Derive move type from board state (m.Flags may be zero when coming from UI)
	isCastle := piece == King && (m.To == m.From+2 || m.To+2 == m.From)
	isEP := piece == Pawn && m.From%8 != m.To%8 && b.Occupancy[opp]&(1<<m.To) == 0
	isCapture := isEP || b.Occupancy[opp]&(1<<m.To) != 0
	isPromo := m.Promotion != NoPiece

	// Build a move with correct flags for the temporary application in sanCheckSuffix.
	flags := uint8(MoveNormal)
	if isCastle {
		flags |= MoveCastle
	}
	if isEP {
		flags |= MoveEP
	}
	if isCapture && !isEP {
		flags |= MoveCapture
	}
	if isPromo {
		flags |= MovePromo
	}
	mf := Move{From: m.From, To: m.To, Promotion: m.Promotion, Flags: flags}

	// Castling
	if isCastle {
		if m.To > m.From {
			return "O-O" + b.sanCheckSuffix(mf, opp)
		}
		return "O-O-O" + b.sanCheckSuffix(mf, opp)
	}

	var buf [10]byte
	n := 0

	// Piece letter (omitted for pawns)
	if piece != Pawn {
		buf[n] = sanPieceLetter(piece)
		n++
	}

	// Disambiguation (non-pawn, non-king only)
	if piece != Pawn && piece != King {
		needFile, needRank := b.sanDisambiguate(m, piece, color)
		if needFile {
			buf[n] = 'a' + m.From%8
			n++
		}
		if needRank {
			buf[n] = '1' + m.From/8
			n++
		}
	}

	// Capture indicator
	if isCapture {
		if piece == Pawn {
			buf[n] = 'a' + m.From%8 // pawn captures always prefix source file
			n++
		}
		buf[n] = 'x'
		n++
	}

	// Destination square
	buf[n] = 'a' + m.To%8
	n++
	buf[n] = '1' + m.To/8
	n++

	// Promotion piece
	if isPromo {
		buf[n] = '='
		n++
		buf[n] = sanPieceLetter(m.Promotion)
		n++
	}

	return string(buf[:n]) + b.sanCheckSuffix(mf, opp)
}

// sanCheckSuffix returns "+", "#", or "" by temporarily applying mf and inspecting
// the resulting position. mf must have correct Flags set.
func (b *Board) sanCheckSuffix(mf Move, opp Color) string {
	temp := *b
	temp.MoveStack = nil // detach; ApplyMove does not use the stack
	temp.ApplyMove(mf)
	temp.SideToMove = opp // flip so HasLegalMoves generates opp's moves correctly
	if !temp.IsKingInCheck(opp) {
		return ""
	}
	if !temp.HasLegalMoves(opp) {
		return "#"
	}
	return "+"
}

// sanDisambiguate returns whether the source file and/or rank must be included to
// distinguish this move from other legal moves by the same piece type.
func (b *Board) sanDisambiguate(m Move, piece Piece, color Color) (needFile, needRank bool) {
	fromFile := m.From % 8
	fromRank := m.From / 8

	var sameFilePeer, sameRankPeer, anyPeer bool

	bb := b.Pieces[color][piece] &^ (1 << uint(m.From))
	for bb != 0 {
		sq := uint8(bits.TrailingZeros64(bb))
		bb &^= 1 << uint(sq)

		for _, lm := range b.GenerateMovesForSquare(sq) {
			if lm.To == m.To {
				anyPeer = true
				if sq%8 == fromFile {
					sameFilePeer = true
				}
				if sq/8 == fromRank {
					sameRankPeer = true
				}
				break
			}
		}
	}

	if !anyPeer {
		return false, false
	}
	// Need both when another piece shares the file AND another shares the rank.
	if sameFilePeer && sameRankPeer {
		return true, true
	}
	if sameFilePeer {
		return false, true // rank alone disambiguates
	}
	return true, false // file alone disambiguates (default)
}

func sanPieceLetter(p Piece) byte {
	switch p {
	case Knight:
		return 'N'
	case Bishop:
		return 'B'
	case Rook:
		return 'R'
	case Queen:
		return 'Q'
	default: // King (Pawn is never passed here)
		return 'K'
	}
}
