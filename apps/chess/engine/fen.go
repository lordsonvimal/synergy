package engine

import (
	"fmt"
	"strconv"
	"strings"
)

var fenCharToPiece = map[byte]Piece{
	'P': Pawn,
	'N': Knight,
	'B': Bishop,
	'R': Rook,
	'Q': Queen,
	'K': King,
}

// BoardFromFENDisplay parses only the piece-placement section of a FEN string
// into a display-only Board (no castling rights, en-passant, or clocks needed).
func BoardFromFENDisplay(fen string) (*Board, error) {
	placement := fen
	if idx := strings.IndexByte(fen, ' '); idx >= 0 {
		placement = fen[:idx]
	}
	ranks := strings.Split(placement, "/")
	if len(ranks) != 8 {
		return nil, fmt.Errorf("BoardFromFENDisplay: expected 8 ranks, got %d", len(ranks))
	}
	b := &Board{EnPassant: 255, SideToMove: White}
	for rankInv, rankStr := range ranks {
		rankIdx := 7 - rankInv
		fileIdx := 0
		for i := 0; i < len(rankStr); i++ {
			ch := rankStr[i]
			if ch >= '1' && ch <= '8' {
				fileIdx += int(ch - '0')
				continue
			}
			upper := ch
			color := White
			if ch >= 'a' && ch <= 'z' {
				upper = ch - 32
				color = Black
			}
			p, ok := fenCharToPiece[upper]
			if !ok {
				return nil, fmt.Errorf("BoardFromFENDisplay: unknown piece %q", ch)
			}
			sq := uint8(rankIdx*8 + fileIdx)
			b.Pieces[color][p] |= bit(sq)
			fileIdx++
		}
	}
	b.updateOccupancy()
	return b, nil
}

// BoardFromFEN parses a full FEN string into a Board with all state fields set.
func BoardFromFEN(fen string) (*Board, error) {
	parts := strings.Fields(fen)
	if len(parts) < 1 {
		return nil, fmt.Errorf("BoardFromFEN: empty FEN")
	}

	b, err := BoardFromFENDisplay(parts[0])
	if err != nil {
		return nil, err
	}

	if len(parts) >= 2 {
		switch parts[1] {
		case "w":
			b.SideToMove = White
		case "b":
			b.SideToMove = Black
		default:
			return nil, fmt.Errorf("BoardFromFEN: invalid side %q", parts[1])
		}
	}

	if len(parts) >= 3 {
		b.Castling = 0
		for _, ch := range parts[2] {
			switch ch {
			case 'K':
				b.Castling |= 0b0001
			case 'Q':
				b.Castling |= 0b0010
			case 'k':
				b.Castling |= 0b0100
			case 'q':
				b.Castling |= 0b1000
			}
		}
	}

	b.EnPassant = 255
	if len(parts) >= 4 && parts[3] != "-" && len(parts[3]) == 2 {
		b.EnPassant = (parts[3][1]-'1')*8 + (parts[3][0]-'a')
	}

	if len(parts) >= 5 {
		if v, err := strconv.ParseUint(parts[4], 10, 16); err == nil {
			b.HalfMoveClock = uint16(v)
		}
	}
	if len(parts) >= 6 {
		if v, err := strconv.ParseUint(parts[5], 10, 16); err == nil {
			b.FullMoveNumber = uint16(v)
		}
	}

	b.Hash = b.BoardHash()
	return b, nil
}
