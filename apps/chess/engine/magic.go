package engine

import "math/bits"

type Magic struct {
	Mask    uint64
	Magic   uint64
	Shift   uint
	Attacks []uint64
}

var RookMagics [64]Magic
var BishopMagics [64]Magic

var rookMagicNumbers = [64]uint64{
	0xA180022080400230, 0x0040100040022000, 0x0080088020001002, 0x0080080280841000,
	0x4200042010460008, 0x04800A0003040080, 0x0400110082041008, 0x008000A041000880,
	0x10138001A080C010, 0x0000804008200480, 0x00010011012000C0, 0x0022004128102200,
	0x000200081201200C, 0x202A001048460004, 0x0081000100420004, 0x000080008000800,
	0x0000208002904001, 0x0001000101080200, 0x0002000110008020, 0x000208000800800,
	0x0002804008200020, 0x0000010004040080, 0x0000204080800880, 0x0000880004002080,
	0x0000801004002001, 0x0020180280080080, 0x000028008000880, 0x0000002040080080,
	0x0000408004040080, 0x000000A010020080, 0x0000040080080080, 0x0000040004008001,
	0x0000020002004000, 0x0000808002004001, 0x0000080010200080, 0x0000802000400080,
	0x0000004000802000, 0x0000002000401000, 0x0000001000200800, 0x0000000800100400,
	0x0000000400080200, 0x0000000200040100, 0x0000000100020080, 0x0000000080010040,
	0x0000000040008020, 0x0000000020004010, 0x0000000010002008, 0x0000000008001004,
	0x0000000004000802, 0x0000000002000401, 0x0000000001000200, 0x0000000000800100,
	0x0000000000400080, 0x0000000000200040, 0x0000000000100020, 0x0000000000080010,
	0x0000000000040008, 0x0000000000020004, 0x0000000000010002, 0x0000000000008001,
}

var bishopMagicNumbers = [64]uint64{
	0x0040201000400040, 0x0000401000200000, 0x0000400800200000, 0x0000002010080000,
	0x00000010080A0000, 0x0000000800401200, 0x0000000400200800, 0x0000000100080400,
	0x0000201000400000, 0x0000004020000000, 0x0000004008000000, 0x0000000020100000,
	0x0000000010080000, 0x0000000008041000, 0x0000000004020800, 0x0000000001000400,
	0x0000401000200000, 0x0000002010000000, 0x0000002008000000, 0x0000000010100000,
	0x0000000008080000, 0x0000000004041000, 0x0000000002020800, 0x0000000000800400,
	0x0000001000800000, 0x0000000020000000, 0x0000000004000000, 0x0000000000082000,
	0x0000000000041000, 0x0000000000020800, 0x0000000000010400, 0x0000000000000200,
	0x0000200800400000, 0x0000001008000000, 0x0000000804000000, 0x0000000008100000,
	0x0000000004080000, 0x0000000002041000, 0x0000000001020800, 0x0000000000800400,
	0x0000000800200000, 0x0000000010000000, 0x0000000002000000, 0x0000000000204000,
	0x0000000000102000, 0x0000000000081000, 0x0000000000040800, 0x0000000000020400,
	0x0000000400100000, 0x0000000008000000, 0x0000000000400000, 0x0000000000040800,
	0x0000000000020400, 0x0000000000010200, 0x0000000000008100, 0x0000000000000080,
	0x0000000200080000, 0x0000000004000000, 0x0000000001000000, 0x0000000000102000,
	0x0000000000081000, 0x0000000000040800, 0x0000000000020400, 0x0000000000010200,
}

func init() {
	initRookMagics()
	initBishopMagics()
}

func rookMask(sq int) uint64 {
	var mask uint64
	rank, file := sq/8, sq%8

	for r := rank + 1; r <= 6; r++ {
		mask |= 1 << (r*8 + file)
	}
	for r := rank - 1; r >= 1; r-- {
		mask |= 1 << (r*8 + file)
	}
	for f := file + 1; f <= 6; f++ {
		mask |= 1 << (rank*8 + f)
	}
	for f := file - 1; f >= 1; f-- {
		mask |= 1 << (rank*8 + f)
	}
	return mask
}

func bishopMask(sq int) uint64 {
	var mask uint64
	rank, file := sq/8, sq%8

	// NE (north-east)
	for r, f := rank+1, file+1; r <= 6 && f <= 6; r, f = r+1, f+1 {
		mask |= 1 << (r*8 + f)
	}
	// NW (north-west)
	for r, f := rank+1, file-1; r <= 6 && f >= 1; r, f = r+1, f-1 {
		mask |= 1 << (r*8 + f)
	}
	// SE (south-east)
	for r, f := rank-1, file+1; r >= 1 && f <= 6; r, f = r-1, f+1 {
		mask |= 1 << (r*8 + f)
	}
	// SW (south-west)
	for r, f := rank-1, file-1; r >= 1 && f >= 1; r, f = r-1, f-1 {
		mask |= 1 << (r*8 + f)
	}

	return mask
}

func initRookMagics() {
	for sq := 0; sq < 64; sq++ {
		mask := rookMask(sq)
		bits := bits.OnesCount64(mask)
		size := 1 << bits

		magic := rookMagicNumbers[sq]
		attacks := make([]uint64, size)

		for i := 0; i < size; i++ {
			occ := indexToOccupancy(i, mask)
			index := (occ * magic) >> (64 - bits)
			attacks[index] = rookAttacksOnTheFly(sq, occ)
		}

		RookMagics[sq] = Magic{
			Mask:    mask,
			Magic:   magic,
			Shift:   uint(64 - bits),
			Attacks: attacks,
		}
	}
}

func initBishopMagics() {
	for sq := 0; sq < 64; sq++ {
		mask := bishopMask(sq) // you need to write bishopMask similar to rookMask
		numBits := bits.OnesCount64(mask)
		size := 1 << numBits
		attacks := make([]uint64, size)
		magic := bishopMagicNumbers[sq]

		for i := 0; i < size; i++ {
			occ := indexToOccupancy(i, mask)
			index := (occ * magic) >> (64 - numBits)
			attacks[index] = bishopAttacksOnTheFly(sq, occ)
		}

		BishopMagics[sq] = Magic{
			Mask:    mask,
			Magic:   magic,
			Shift:   uint(64 - numBits),
			Attacks: attacks,
		}
	}
}

func indexToOccupancy(index int, mask uint64) uint64 {
	var occ uint64
	numBits := bits.OnesCount64(mask) // renamed from bits

	for i := 0; i < numBits; i++ {
		sq := bits.TrailingZeros64(mask) // now this uses the package
		mask &^= 1 << sq

		if (index & (1 << i)) != 0 {
			occ |= 1 << sq
		}
	}

	return occ
}

func rookAttacksOnTheFly(sq int, occ uint64) uint64 {
	var attacks uint64

	rank := sq / 8
	file := sq % 8

	// North
	for r := rank + 1; r < 8; r++ {
		s := r*8 + file
		attacks |= 1 << s
		if occ&(1<<s) != 0 {
			break
		}
	}

	// South
	for r := rank - 1; r >= 0; r-- {
		s := r*8 + file
		attacks |= 1 << s
		if occ&(1<<s) != 0 {
			break
		}
	}

	// East
	for f := file + 1; f < 8; f++ {
		s := rank*8 + f
		attacks |= 1 << s
		if occ&(1<<s) != 0 {
			break
		}
	}

	// West
	for f := file - 1; f >= 0; f-- {
		s := rank*8 + f
		attacks |= 1 << s
		if occ&(1<<s) != 0 {
			break
		}
	}

	return attacks
}

func bishopAttacksOnTheFly(sq int, occ uint64) uint64 {
	var attacks uint64

	rank := sq / 8
	file := sq % 8

	// NE
	for r, f := rank+1, file+1; r < 8 && f < 8; r, f = r+1, f+1 {
		s := r*8 + f
		attacks |= 1 << s
		if occ&(1<<s) != 0 {
			break
		}
	}

	// NW
	for r, f := rank+1, file-1; r < 8 && f >= 0; r, f = r+1, f-1 {
		s := r*8 + f
		attacks |= 1 << s
		if occ&(1<<s) != 0 {
			break
		}
	}

	// SE
	for r, f := rank-1, file+1; r >= 0 && f < 8; r, f = r-1, f+1 {
		s := r*8 + f
		attacks |= 1 << s
		if occ&(1<<s) != 0 {
			break
		}
	}

	// SW
	for r, f := rank-1, file-1; r >= 0 && f >= 0; r, f = r-1, f-1 {
		s := r*8 + f
		attacks |= 1 << s
		if occ&(1<<s) != 0 {
			break
		}
	}

	return attacks
}
