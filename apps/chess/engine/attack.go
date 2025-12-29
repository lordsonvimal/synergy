package engine

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

func RookAttacks(sq uint8, occ Bitboard) Bitboard {
	var attacks Bitboard

	rank := sq / 8
	file := sq % 8

	// Up
	for r := rank + 1; r < 8; r++ {
		s := uint8(r*8) + file
		attacks |= 1 << s
		if occ&(1<<s) != 0 {
			break
		}
	}

	// Down
	for r := int(rank) - 1; r >= 0; r-- {
		s := uint8(r*8) + file
		attacks |= 1 << s
		if occ&(1<<s) != 0 {
			break
		}
	}

	// Right
	for f := file + 1; f < 8; f++ {
		s := rank*8 + f
		attacks |= 1 << s
		if occ&(1<<s) != 0 {
			break
		}
	}

	// Left
	for f := int(file) - 1; f >= 0; f-- {
		s := rank*8 + uint8(f)
		attacks |= 1 << s
		if occ&(1<<s) != 0 {
			break
		}
	}

	return attacks
}

// BishopAttacks generates all squares a bishop attacks from sq given occupancy
func BishopAttacks(sq uint8, occ Bitboard) Bitboard {
	var attacks Bitboard

	rank := sq / 8
	file := sq % 8

	// Up-right
	for r, f := int(rank)+1, int(file)+1; r < 8 && f < 8; r, f = r+1, f+1 {
		s := uint8(r*8 + f)
		attacks |= 1 << s
		if occ&(1<<s) != 0 {
			break
		}
	}

	// Up-left
	for r, f := int(rank)+1, int(file)-1; r < 8 && f >= 0; r, f = r+1, f-1 {
		s := uint8(r*8 + f)
		attacks |= 1 << s
		if occ&(1<<s) != 0 {
			break
		}
	}

	// Down-right
	for r, f := int(rank)-1, int(file)+1; r >= 0 && f < 8; r, f = r-1, f+1 {
		s := uint8(r*8 + f)
		attacks |= 1 << s
		if occ&(1<<s) != 0 {
			break
		}
	}

	// Down-left
	for r, f := int(rank)-1, int(file)-1; r >= 0 && f >= 0; r, f = r-1, f-1 {
		s := uint8(r*8 + f)
		attacks |= 1 << s
		if occ&(1<<s) != 0 {
			break
		}
	}

	return attacks
}
