package engine

import (
	"math/rand"
)

const (
	ZobristPieceCount = 12 // 6 pieces x 2 colors
	Squares           = 64
)

// Zobrist table
var Zobrist [2][6][64]uint64 // [color][piece][square]
// var ZobristSide uint64       // side to move
var (
	ZPiece  [ColorNB][PieceNB][64]uint64
	ZCastle [16]uint64
	ZEP     [8]uint64
	ZSide   uint64
)

func init() {
	r := rand.New(rand.NewSource(20250101))
	for color := 0; color < 2; color++ {
		for piece := 0; piece < 6; piece++ {
			for sq := 0; sq < 64; sq++ {
				Zobrist[color][piece][sq] = r.Uint64()
			}
		}
	}
	for i := 0; i < 16; i++ {
		ZCastle[i] = r.Uint64()
	}

	for i := 0; i < 8; i++ {
		ZEP[i] = r.Uint64()
	}

	ZSide = r.Uint64()
}

func (b *Board) BoardHash() uint64 {
	var h uint64

	for c := Color(0); c < ColorNB; c++ {
		for p := Piece(0); p < PieceNB; p++ {
			bb := b.Pieces[c][p]
			for bb != 0 {
				sq := PopLSB(&bb)
				h ^= ZPiece[c][p][sq]
			}
		}
	}

	h ^= ZCastle[b.Castling]

	if b.EnPassant != 255 {
		h ^= ZEP[b.EnPassant%8]
	}

	if b.SideToMove == Black {
		h ^= ZSide
	}

	return h
}

func (b *Board) UpdateHash(
	m Move,
	color Color,
	piece Piece,
	captured Piece,
	oldCastle uint8,
	oldEP uint8,
) {
	// Remove old state
	b.Hash ^= ZSide
	b.Hash ^= ZCastle[oldCastle]
	if oldEP != 255 {
		b.Hash ^= ZEP[oldEP%8]
	}

	// 1. Remove pawn from FROM square
	b.Hash ^= ZPiece[color][piece][m.From]

	// 2. Add promoted piece to TO square
	finalPiece := piece
	if m.Promotion != NoPiece {
		finalPiece = m.Promotion
	}
	b.Hash ^= ZPiece[color][finalPiece][m.To]

	// 3. Handle capture
	if captured != NoPiece {
		capSq := m.To
		if m.Flags&MoveEP != 0 {
			if color == White {
				capSq = m.To - 8
			} else {
				capSq = m.To + 8
			}
		}
		b.Hash ^= ZPiece[color^1][captured][capSq]
	}

	// Add new state
	b.Hash ^= ZCastle[b.Castling]
	if b.EnPassant != 255 {
		b.Hash ^= ZEP[b.EnPassant%8]
	}
}
