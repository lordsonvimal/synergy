package engine

import (
	"math/rand"
	"time"
)

const (
	ZobristPieceCount = 12 // 6 pieces x 2 colors
	Squares           = 64
)

// Zobrist table
var Zobrist [2][6][64]uint64 // [color][piece][square]
var ZobristSide uint64       // side to move

func init() {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for color := 0; color < 2; color++ {
		for piece := 0; piece < 6; piece++ {
			for sq := 0; sq < 64; sq++ {
				Zobrist[color][piece][sq] = r.Uint64()
			}
		}
	}
	ZobristSide = r.Uint64()
}

// BoardHash computes the Zobrist hash for a board
func (b *Board) BoardHash() uint64 {
	hash := uint64(0)
	for color := 0; color < 2; color++ {
		for piece := Piece(0); piece < PieceNB; piece++ {
			bb := b.Pieces[color][piece]
			for bb != 0 {
				sq := PopLSB((*Bitboard)(&bb))
				hash ^= Zobrist[color][piece][sq]
			}
		}
	}
	if b.SideToMove == Black {
		hash ^= ZobristSide
	}
	return hash
}

// Incremental update after a move
func (b *Board) UpdateHash(m Move, color Color, captured Piece, promoted Piece) uint64 {
	hash := b.Hash // current hash stored in board

	// Remove moving piece from from-square
	hash ^= Zobrist[color][movedPiece(b, m.From)][m.From]

	// Add moving piece to to-square
	finalPiece := movedPiece(b, m.From)
	if promoted != NoPiece {
		finalPiece = promoted
	}
	hash ^= Zobrist[color][finalPiece][m.To]

	// Remove captured piece if any
	if captured != NoPiece {
		hash ^= Zobrist[color^1][captured][m.To]
	}

	// Side to move
	hash ^= ZobristSide

	return hash
}

// Helper to get piece on a square
func movedPiece(b *Board, sq uint8) Piece {
	for p := Piece(0); p < PieceNB; p++ {
		for color := Color(0); color < 2; color++ {
			if b.Pieces[color][p]&(1<<sq) != 0 {
				return p
			}
		}
	}
	return NoPiece
}
