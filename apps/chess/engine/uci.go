// engine/uci.go
package engine

func EncodeUCI(m Move) string {
	uci := squareToAlgebraic(m.From) +
		squareToAlgebraic(m.To)

	if m.Promotion != NoPiece {
		uci += promotionChar(m.Promotion)
	}

	return uci
}

func promotionChar(p Piece) string {
	switch p {
	case Queen:
		return "q"
	case Rook:
		return "r"
	case Bishop:
		return "b"
	case Knight:
		return "n"
	default:
		return ""
	}
}
