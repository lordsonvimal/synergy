package engine

import "fmt"

// --------------------------
// Bit helpers
// --------------------------

// Bitboard type (alias for clarity)
type Bitboard uint64

// Return bitboard with only bit `sq` set
func SqBB(sq uint64) uint64 {
	return 1 << sq
}

// Pop least significant bit and return index
func PopLSB(bb *uint64) uint8 {
	lsb := *bb & -(*bb)
	index := uint8(BitScanForward(*bb))
	*bb &^= lsb
	return index
}

// Count set bits
func PopCount(bb uint64) int {
	count := 0
	b := uint64(bb)
	for b != 0 {
		b &= b - 1
		count++
	}
	return count
}

// Scan forward (LSB index)
func BitScanForward(bb uint64) int {
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
var KnightAttacks [64]uint64

func init() {
	for sq := 0; sq < 64; sq++ {
		attacks := uint64(0)
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
var KingAttacks [64]uint64

func init() {
	for sq := 0; sq < 64; sq++ {
		attacks := uint64(0)
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
// Rank, File, Diagonal masks
// --------------------------
var RankMask [8]uint64
var FileMask [8]uint64

func init() {
	for r := 0; r < 8; r++ {
		var mask uint64
		for f := 0; f < 8; f++ {
			mask |= SqBB(uint64(r*8 + f))
		}
		RankMask[r] = mask
	}

	for f := 0; f < 8; f++ {
		var mask uint64
		for r := 0; r < 8; r++ {
			mask |= SqBB(uint64(r*8 + f))
		}
		FileMask[f] = mask
	}
}

func PrintBB(bb uint64) {
	fmt.Println("+---+---+---+---+---+---+---+---+")
	// Start from rank 7 (top) down to rank 0 (bottom)
	for r := 7; r >= 0; r-- {
		fmt.Print("|")
		for f := 0; f < 8; f++ {
			sq := r*8 + f
			if (bb & (1 << sq)) != 0 {
				fmt.Print(" X |")
			} else {
				fmt.Print(" . |")
			}
		}
		fmt.Printf(" %d\n", r+1) // Rank number
		fmt.Println("+---+---+---+---+---+---+---+---+")
	}
	fmt.Println("  a   b   c   d   e   f   g   h")
	fmt.Printf("Bitboard: 0x%016X\n\n", bb)
}
