package engine

type PerftTTEntry struct {
	Depth int
	Nodes uint64
}

type PerftTT map[uint64]PerftTTEntry

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
		b.UnapplyMove()
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
		b.UnapplyMove()

		results[m.ToUCI()] = count
	}

	return results
}

func (b *Board) PerftTT(depth int, tt PerftTT) uint64 {
	if depth == 0 {
		return 1
	}

	// TT lookup
	if entry, ok := tt[b.Hash]; ok && entry.Depth == depth {
		return entry.Nodes
	}

	var nodes uint64
	moves := b.GeneratePseudoLegalMoves()

	for _, m := range moves {
		if !b.MakeMove(m) {
			continue
		}

		nodes += b.PerftTT(depth-1, tt)
		b.UnapplyMove()
	}

	// Store in TT
	tt[b.Hash] = PerftTTEntry{
		Depth: depth,
		Nodes: nodes,
	}

	return nodes
}

func (b *Board) PerftDivideTT(depth int) map[string]uint64 {
	results := make(map[string]uint64)
	tt := make(PerftTT)

	moves := b.GeneratePseudoLegalMoves()

	for _, m := range moves {
		if !b.MakeMove(m) {
			continue
		}

		nodes := b.PerftTT(depth-1, tt)
		b.UnapplyMove()

		results[m.ToUCI()] = nodes
	}

	return results
}
