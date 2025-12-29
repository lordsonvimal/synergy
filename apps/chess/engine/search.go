package engine

import (
	"context"
	"math/bits"
	"time"
)

const (
	INF        = 1_000_000
	MATE_SCORE = 100_000
)

// --------------------
// Search result
// --------------------

type SearchResult struct {
	BestMove Move
	Score    int
	Depth    int
	Nodes    uint64
}

// --------------------
// Searcher
// --------------------

type Searcher struct {
	TT    *TranspositionTable
	Nodes uint64
	Ctx   context.Context
}

// --------------------
// Evaluation
// --------------------

var pieceValue = [PieceNB]int{
	Pawn:   100,
	Knight: 320,
	Bishop: 330,
	Rook:   500,
	Queen:  900,
	King:   0,
}

func Evaluate(b *Board) int {
	score := 0
	for color := Color(0); color < ColorNB; color++ {
		sign := 1
		if color != b.SideToMove {
			sign = -1
		}
		for p := Piece(0); p < PieceNB; p++ {
			score += sign * pieceValue[p] * bits.OnesCount64(b.Pieces[color][p])
		}
	}
	return score
}

// --------------------
// Public entry point
// --------------------

func (s *Searcher) Search(b *Board, maxDepth int, timeLimit time.Duration) SearchResult {
	s.Nodes = 0
	ctx, cancel := context.WithTimeout(context.Background(), timeLimit)
	defer cancel()
	s.Ctx = ctx

	s.TT.NewSearch()

	var bestMove Move
	bestScore := -INF
	lastDepth := 0

	for depth := 1; depth <= maxDepth; depth++ {
		score, move := s.searchRoot(b, depth)
		if ctx.Err() != nil {
			break
		}
		bestScore = score
		bestMove = move
		lastDepth = depth
	}

	return SearchResult{
		BestMove: bestMove,
		Score:    bestScore,
		Depth:    lastDepth,
		Nodes:    s.Nodes,
	}
}

// --------------------
// Root search
// --------------------

func (s *Searcher) searchRoot(b *Board, depth int) (int, Move) {
	alpha := -INF
	beta := INF

	moves := b.GeneratePseudoLegalMoves()

	bestMove := Move{}
	bestScore := -INF

	for _, m := range moves {
		if !b.MakeMove(m) {
			continue
		}

		score := -s.alphaBeta(b, depth-1, -beta, -alpha, 1)
		b.unapplyMove()

		if score > bestScore {
			bestScore = score
			bestMove = m
		}
		if score > alpha {
			alpha = score
		}
	}

	return bestScore, bestMove
}

// --------------------
// Alpha-beta
// --------------------

func (s *Searcher) alphaBeta(b *Board, depth int, alpha int, beta int, ply int) int {
	if s.Nodes&4095 == 0 && s.Ctx.Err() != nil {
		return 0
	}

	s.Nodes++

	// Transposition Table probe
	if val, _, ok := s.TT.Probe(b.Hash, depth, alpha, beta); ok {
		return val
	}

	if depth == 0 {
		return s.quiescence(b, alpha, beta)
	}

	moves := b.GeneratePseudoLegalMoves()
	hasLegal := false
	bestMove := Move{}
	origAlpha := alpha

	for _, m := range moves {
		if !b.MakeMove(m) {
			continue
		}

		hasLegal = true

		score := -s.alphaBeta(b, depth-1, -beta, -alpha, ply+1)
		b.unapplyMove()

		if score >= beta {
			s.TT.Store(b.Hash, depth, beta, TTLowerBound, m, ply)
			return beta
		}
		if score > alpha {
			alpha = score
			bestMove = m
		}
	}

	// Checkmate / stalemate
	if !hasLegal {
		if b.IsKingInCheck(b.SideToMove) {
			return -MATE_SCORE + ply
		}
		return 0
	}

	entryType := TTUpperBound
	if alpha > origAlpha {
		entryType = TTExact
	}

	s.TT.Store(b.Hash, depth, alpha, entryType, bestMove, ply)
	return alpha
}

// --------------------
// Quiescence search
// --------------------

func (s *Searcher) quiescence(b *Board, alpha int, beta int) int {
	if s.Nodes&4095 == 0 && s.Ctx.Err() != nil {
		return 0
	}

	s.Nodes++

	score := Evaluate(b)
	if score >= beta {
		return beta
	}
	if score > alpha {
		alpha = score
	}

	// Captures only
	moves := b.GenerateCaptures()
	for _, m := range moves {
		if !b.MakeMove(m) {
			continue
		}

		score := -s.quiescence(b, -beta, -alpha)
		b.unapplyMove()

		if score >= beta {
			return beta
		}
		if score > alpha {
			alpha = score
		}
	}

	return alpha
}
