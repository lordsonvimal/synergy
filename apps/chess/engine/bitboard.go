package engine

// --------------------------
// Bit helpers
// --------------------------

// Bitboard type (alias for clarity)
type Bitboard uint64

// Return bitboard with only bit `sq` set
func SqBB(sq uint64) Bitboard {
	return 1 << sq
}

// Pop least significant bit and return index
func PopLSB(bb *Bitboard) uint8 {
	lsb := *bb & -(*bb)
	index := uint8(BitScanForward(*bb))
	*bb &^= lsb
	return index
}

// Count set bits
func PopCount(bb Bitboard) int {
	count := 0
	b := uint64(bb)
	for b != 0 {
		b &= b - 1
		count++
	}
	return count
}

// Scan forward (LSB index)
func BitScanForward(bb Bitboard) int {
	if bb == 0 {
		return -1
	}
	index := 0
	for (bb & 1) == 0 {
		bb >>= 1
		index++
	}
	return index
}

// --------------------------
// Precomputed attack masks
// --------------------------

// Knight moves
var KnightAttacks [64]Bitboard

func init() {
	for sq := 0; sq < 64; sq++ {
		attacks := Bitboard(0)
		rank := sq / 8
		file := sq % 8

		dirs := [][2]int{
			{2, 1}, {1, 2}, {-1, 2}, {-2, 1},
			{-2, -1}, {-1, -2}, {1, -2}, {2, -1},
		}

		for _, d := range dirs {
			r, f := rank+d[0], file+d[1]
			if r >= 0 && r < 8 && f >= 0 && f < 8 {
				attacks |= SqBB(uint64(r*8 + f))
			}
		}
		KnightAttacks[sq] = attacks
	}
}

// King moves
var KingAttacks [64]Bitboard

func init() {
	for sq := 0; sq < 64; sq++ {
		attacks := Bitboard(0)
		rank := sq / 8
		file := sq % 8

		dirs := [][2]int{
			{1, 0}, {1, 1}, {0, 1}, {-1, 1},
			{-1, 0}, {-1, -1}, {0, -1}, {1, -1},
		}

		for _, d := range dirs {
			r, f := rank+d[0], file+d[1]
			if r >= 0 && r < 8 && f >= 0 && f < 8 {
				attacks |= SqBB(uint64(r*8 + f))
			}
		}
		KingAttacks[sq] = attacks
	}
}

// --------------------------
// Pawn attacks
// --------------------------
func PawnAttacks(color Color, sq uint8) Bitboard {
	rank := sq / 8
	file := sq % 8
	var attacks Bitboard

	if color == White {
		if file > 0 && rank < 7 {
			attacks |= SqBB(uint64((rank+1)*8 + (file - 1)))
		}
		if file < 7 && rank < 7 {
			attacks |= SqBB(uint64((rank+1)*8 + (file + 1)))
		}
	} else {
		if file > 0 && rank > 0 {
			attacks |= SqBB(uint64((rank-1)*8 + (file - 1)))
		}
		if file < 7 && rank > 0 {
			attacks |= SqBB(uint64((rank-1)*8 + (file + 1)))
		}
	}

	return attacks
}

// --------------------------
// Rank, File, Diagonal masks
// --------------------------
var RankMask [8]Bitboard
var FileMask [8]Bitboard

func init() {
	for r := 0; r < 8; r++ {
		var mask Bitboard
		for f := 0; f < 8; f++ {
			mask |= SqBB(uint64(r*8 + f))
		}
		RankMask[r] = mask
	}

	for f := 0; f < 8; f++ {
		var mask Bitboard
		for r := 0; r < 8; r++ {
			mask |= SqBB(uint64(r*8 + f))
		}
		FileMask[f] = mask
	}
}
