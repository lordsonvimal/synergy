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
	0xa8002c862a04105, 0x4210810052201, 0x4008130061341, 0x188109010120004,
	0x100440b0120114, 0x120481003024, 0x80a4010004a10, 0x880104031220300,
	0x80010c12020302, 0x8124812030900, 0x440a02a01104, 0x40a020110006040,
	0x1108a8165140100, 0x201108069110040, 0x100444a00401100, 0x820800302020001,
	0x2028a0040812200, 0x410822028420040, 0x4000820800a0050, 0x401020c004108,
	0x400a02012001, 0x4004080120a0100, 0x408800001220040, 0x2080088201100,
	0x202a00201048020, 0x410081010020004, 0x4010020102a00, 0x40210820c0041,
	0x411082a040001, 0x40110408100040, 0x401000820802, 0x2044011001100,
	0x40040801010c020, 0x400440801020401, 0x42021080040101, 0x2084101306802,
	0x40840a0201008, 0x10008c01001080, 0x480d800061000, 0x210a4000411,
	0x201c00a0280102, 0x2001000420004, 0x410082008020001, 0x40a030004008,
	0x208008824008, 0x804010a0110001, 0x4220080201200, 0x2010802002401,
	0x44100008020802, 0x2080600100042, 0x2201000408104, 0x420880104006005,
	0x20820a004000, 0x400408100500a, 0x4004080120008, 0x1100082052000,
	0x204000504812, 0x100044010011, 0x209800061020001, 0x210c08070100,
	0x41005080602, 0x10028408000410, 0x40800204010040, 0x1008021020010,
}

var bishopMagicNumbers = [64]uint64{
	0x40040822862081, 0x40810a4108000, 0x200800840042001, 0x401600c80210200,
	0x4060820c800a00, 0x480204028100402, 0x100808202020010, 0x24160080200080,
	0x18044840008010, 0x108080840450010, 0x108002012020010, 0x80220201408140,
	0x803888004452a0, 0x202c0084161004, 0x10010041004002, 0x20400021008010,
	0x80402040080810, 0x201000820101100, 0x20200802401010, 0x10010082020420,
	0x20010b00100104, 0x4021082080020, 0x41008210400a00, 0x40008082004400,
	0x810102100020, 0x2104448080100, 0x40400801200802, 0x220800c010c080,
	0x8100408010, 0x4010008008041, 0x40801062000, 0x8200400101100,
	0x82a08120401, 0x80040080800201, 0x2010806102000, 0x210081008221,
	0x800108220200, 0x80010081000, 0x204081000402, 0x101000208200,
	0x810a0300102001, 0x40028c200600, 0x20a0003040080, 0x841200080080,
	0x4040001011002, 0x21000010081, 0x4002100111008, 0x80004040008010,
	0x20001101000, 0x21001400802, 0x80a00108000, 0x4008081011002,
	0x80040080800201, 0x1012a0120004, 0x410024000810, 0x40011000210200,
	0x40820000044100, 0x11001028000, 0x401000820080, 0x1000010820044,
	0x4002000408221, 0x4004010140c020, 0x20120020200, 0x8001020200102,
}

// Use these sizes for the Attack arrays to prevent out-of-bounds
// Rook: max 12 bits (4096), Bishop: max 9 bits (512)
var RookTable [64][4096]uint64
var BishopTable [64][512]uint64

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

	// Directions: NE, NW, SE, SW
	dr := []int{1, 1, -1, -1}
	df := []int{1, -1, 1, -1}

	for i := 0; i < 4; i++ {
		r, f := rank+dr[i], file+df[i]
		// We stop BEFORE the edge (r=0, r=7, f=0, f=7)
		// because blockers on the edge don't change the reachable squares
		for r > 0 && r < 7 && f > 0 && f < 7 {
			mask |= 1 << (r*8 + f)
			r += dr[i]
			f += df[i]
		}
	}
	return mask
}

func initRookMagics() {
	for sq := 0; sq < 64; sq++ {
		mask := rookMask(sq)
		numBits := bits.OnesCount64(mask)
		size := 1 << numBits
		magic := rookMagicNumbers[sq]

		for i := 0; i < size; i++ {
			occ := indexToOccupancy(i, mask)
			index := (occ * magic) >> (64 - numBits)
			// FIX: Store in global RookTable, not a local slice
			RookTable[sq][index] = rookAttacksOnTheFly(sq, occ)
		}

		RookMagics[sq] = Magic{
			Mask:  mask,
			Magic: magic,
			Shift: uint(64 - numBits),
		}
	}
}

func initBishopMagics() {
	for sq := 0; sq < 64; sq++ {
		mask := bishopMask(sq)
		numBits := bits.OnesCount64(mask)
		size := 1 << numBits
		magic := bishopMagicNumbers[sq]

		for i := 0; i < size; i++ {
			occ := indexToOccupancy(i, mask)
			// The shift must be exactly 64 - numBits
			index := (occ * magic) >> (64 - numBits)
			BishopTable[sq][index] = bishopAttacksOnTheFly(sq, occ)
		}

		BishopMagics[sq] = Magic{
			Mask:  mask,
			Magic: magic,
			Shift: uint(64 - numBits),
		}
	}
}

func indexToOccupancy(index int, mask uint64) uint64 {
	occ := uint64(0)
	numBits := bits.OnesCount64(mask)
	for i := 0; i < numBits; i++ {
		sq := bits.TrailingZeros64(mask)
		mask &= ^(uint64(1) << sq)
		if (index & (1 << i)) != 0 {
			occ |= (uint64(1) << sq)
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
