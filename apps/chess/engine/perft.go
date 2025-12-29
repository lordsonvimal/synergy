package engine

// Perft counts legal leaf nodes at given depth
func (b *Board) Perft(depth int) uint64 {
	if depth == 0 {
		return 1
	}

	var nodes uint64
	moves := b.GeneratePseudoLegalMoves()

	for _, m := range moves {
		if !b.MakeMove(m) {
			continue // illegal move (king in check)
		}

		nodes += b.Perft(depth - 1)
		b.unapplyMove()
	}

	return nodes
}

func (b *Board) PerftDivide(depth int) map[string]uint64 {
	results := make(map[string]uint64)
	moves := b.GeneratePseudoLegalMoves()

	for _, m := range moves {
		if !b.MakeMove(m) {
			continue
		}

		count := b.Perft(depth - 1)
		b.unapplyMove()

		results[m.String()] = count
	}

	return results
}
