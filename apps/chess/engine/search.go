package engine

import (
	"context"
	"math/bits"
	"time"
)

const (
	INF                = 1_000_000
	MATE_SCORE         = 100_000
	NULLMOVE_REDUCTION = 2
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
	TT       *TranspositionTable
	Nodes    uint64
	Ctx      context.Context
	MaxDepth int

	moveBuf [][]Move // preallocated per-ply moves
	killer  [][]Move // killer moves per-ply
	history [ColorNB][PieceNB][64]int
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
	s.MaxDepth = maxDepth
	s.moveBuf = make([][]Move, maxDepth+2)
	s.killer = make([][]Move, maxDepth+2)

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

	// TT move first
	if ttMove, ok := s.TT.GetMove(b.Hash); ok {
		moves = moveFirst(moves, ttMove)
	}

	for _, m := range moves {
		if !b.MakeMove(m) {
			continue
		}

		score := -s.alphaBeta(b, depth-1, -beta, -alpha, 1, true)
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

func (s *Searcher) alphaBeta(b *Board, depth, alpha, beta, ply int, allowNull bool) int {
	if s.Nodes&4095 == 0 && s.Ctx.Err() != nil {
		return 0
	}
	s.Nodes++

	// Transposition Table
	if val, _, ok := s.TT.Probe(b.Hash, depth, alpha, beta); ok {
		return val
	}

	if depth <= 0 {
		return s.quiescence(b, alpha, beta)
	}

	// Null-move pruning
	if allowNull && depth > NULLMOVE_REDUCTION+1 && !b.IsKingInCheck(b.SideToMove) {
		prevHash := b.Hash
		prevSide := b.SideToMove
		b.SideToMove ^= 1
		b.Hash ^= ZSide // flip side hash
		score := -s.alphaBeta(b, depth-1-NULLMOVE_REDUCTION, -beta, -beta+1, ply+1, false)
		b.SideToMove = prevSide
		b.Hash = prevHash
		if score >= beta {
			return beta
		}
	}

	moves := b.GeneratePseudoLegalMoves()
	hasLegal := false
	bestMove := Move{}
	origAlpha := alpha

	// TT move first
	if ttMove, ok := s.TT.GetMove(b.Hash); ok {
		moves = moveFirst(moves, ttMove)
	}

	// Separate captures and quiets
	captures := []Move{}
	quiets := []Move{}
	for _, m := range moves {
		if m.Flags&MoveCapture != 0 {
			captures = append(captures, m)
		} else {
			quiets = append(quiets, m)
		}
	}

	captures = sortCapturesMVV(captures, b)
	moves = append(captures, s.orderQuiets(b, quiets, ply)...)

	for _, m := range moves {
		color := b.SideToMove
		piece := b.pieceOnSquare(m.From)

		if !b.MakeMove(m) {
			continue
		}

		hasLegal = true
		score := -s.alphaBeta(b, depth-1, -beta, -alpha, ply+1, true)
		b.unapplyMove()

		if score >= beta {
			// TT store lower bound
			s.TT.Store(b.Hash, depth, beta, TTLowerBound, m, ply)
			if m.Flags&MoveCapture == 0 {
				s.recordKiller(m, ply)
			}
			return beta
		}

		if score > alpha {
			alpha = score
			bestMove = m
			if m.Flags&MoveCapture == 0 {
				s.history[color][piece][m.To] += depth * depth
			}
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

func (s *Searcher) quiescence(b *Board, alpha, beta int) int {
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

	moves := b.GenerateCaptures()
	moves = sortCapturesMVV(moves, b)

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

// --------------------
// Helper: Killer + History ordering
// --------------------

func (s *Searcher) recordKiller(m Move, ply int) {
	if len(s.killer[ply]) < 2 {
		s.killer[ply] = append(s.killer[ply], m)
	} else {
		s.killer[ply][0] = s.killer[ply][1]
		s.killer[ply][1] = m
	}
}

func (s *Searcher) orderQuiets(b *Board, quietMoves []Move, ply int) []Move {
	ordered := []Move{}
	// Killer moves first
	for _, k := range s.killer[ply] {
		for i, m := range quietMoves {
			if m == k {
				ordered = append(ordered, m)
				quietMoves = append(quietMoves[:i], quietMoves[i+1:]...)
				break
			}
		}
	}

	// Remaining sorted by history heuristic
	for _, m := range quietMoves {
		score := s.history[b.SideToMove][b.pieceOnSquare(m.From)][m.To]
		inserted := false
		for i, om := range ordered {
			oscore := s.history[b.SideToMove][b.pieceOnSquare(om.From)][om.To]
			if score > oscore {
				ordered = append(ordered[:i], append([]Move{m}, ordered[i:]...)...)
				inserted = true
				break
			}
		}
		if !inserted {
			ordered = append(ordered, m)
		}
	}
	return ordered
}

// --------------------
// Helper: sort captures MVV-LVA
// --------------------

func sortCapturesMVV(moves []Move, b *Board) []Move {
	for i := 1; i < len(moves); i++ {
		j := i
		for j > 0 {
			attacker := b.pieceOnSquare(moves[j].From)
			victim := b.pieceOnSquare(moves[j].To)
			score := pieceValue[victim]*10 - pieceValue[attacker]
			prevAttacker := b.pieceOnSquare(moves[j-1].From)
			prevVictim := b.pieceOnSquare(moves[j-1].To)
			prevScore := pieceValue[prevVictim]*10 - pieceValue[prevAttacker]
			if score > prevScore {
				moves[j], moves[j-1] = moves[j-1], moves[j]
				j--
			} else {
				break
			}
		}
	}
	return moves
}

// --------------------
// Move ordering helper
// --------------------

func moveFirst(moves []Move, best Move) []Move {
	for i, m := range moves {
		if m == best {
			if i == 0 {
				return moves
			}
			moves[0], moves[i] = moves[i], moves[0]
			break
		}
	}
	return moves
}
