package engine

// --------------------------
// Pawn attacks
// --------------------------
func PawnAttacks(color Color, sq uint8) uint64 {
	rank := sq / 8
	file := sq % 8
	var attacks uint64

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

// BishopAttacks generates all squares a bishop attacks from sq given occupancy
func RookAttacks(sq uint8, occ uint64) uint64 {
	m := RookMagics[sq]
	occ &= m.Mask
	index := (occ * m.Magic) >> m.Shift
	return RookTable[sq][index]
}

func BishopAttacks(sq uint8, occ uint64) uint64 {
	m := BishopMagics[sq]
	occ &= m.Mask
	index := (occ * m.Magic) >> m.Shift
	return BishopTable[sq][index]
}
