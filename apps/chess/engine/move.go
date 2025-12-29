package engine

import "fmt"

// --------------------------
// Move representation
// --------------------------

type Move struct {
	From      uint8 // 0–63
	To        uint8 // 0–63
	Promotion Piece // NoPiece if none
	Flags     uint8 // bitflags: MoveNormal, MoveCastle, MoveEP, MovePromo, MoveCapture
}

// Move flags
const (
	MoveNormal  = 0
	MoveCastle  = 1 << 0
	MoveEP      = 1 << 1
	MovePromo   = 1 << 2
	MoveCapture = 1 << 3
)

// --------------------------
// Move encoding / decoding
// --------------------------

// EncodeMove encodes a move into 16-bit integer for storage/transposition tables
func EncodeMove(from, to uint8, promo Piece, flags uint8) uint16 {
	return uint16(from) | (uint16(to) << 6) | (uint16(flags) << 12) | (uint16(promo) << 8)
}

// DecodeMove decodes 16-bit move into Move struct
func DecodeMove(encoded uint16) Move {
	from := uint8(encoded & 0x3F)
	to := uint8((encoded >> 6) & 0x3F)
	promo := Piece((encoded >> 8) & 0xF)
	flags := uint8((encoded >> 12) & 0xF)
	return Move{From: from, To: to, Promotion: promo, Flags: flags}
}

// --------------------------
// Move helpers
// --------------------------

func (m Move) IsPromotion() bool {
	return m.Flags&MovePromo != 0 || m.Promotion != NoPiece
}

func (m Move) IsCapture(b *Board, color Color) bool {
	opp := color ^ 1 // opponent color
	return (b.Occupancy[opp] & 1 << m.To) != 0
}

func (m Move) IsCastle() bool {
	return m.Flags&MoveCastle != 0
}

func (m Move) IsEnPassant() bool {
	return m.Flags&MoveEP != 0
}

// --------------------------
// UCI conversion
// --------------------------

// ToUCI converts move to standard UCI string (e2e4, e7e8q)
func (m Move) ToUCI() string {
	var buf [5]byte // max 5 bytes: 4 for squares + 1 for promotion
	buf[0] = 'a' + (m.From % 8)
	buf[1] = '1' + (m.From / 8)
	buf[2] = 'a' + (m.To % 8)
	buf[3] = '1' + (m.To / 8)

	if m.Promotion != NoPiece {
		switch m.Promotion {
		case Queen:
			buf[4] = 'q'
		case Rook:
			buf[4] = 'r'
		case Bishop:
			buf[4] = 'b'
		case Knight:
			buf[4] = 'n'
		}
		return string(buf[:5])
	}

	return string(buf[:4])
}

// MoveFromUCI parses a UCI string into Move
func MoveFromUCI(s string) Move {
	if len(s) < 4 {
		panic(fmt.Sprintf("invalid UCI move string: %s", s))
	}

	from := uint8((s[1]-'1')*8 + (s[0] - 'a'))
	to := uint8((s[3]-'1')*8 + (s[2] - 'a'))
	var promo Piece = NoPiece
	var flags uint8 = MoveNormal

	if len(s) == 5 {
		flags |= MovePromo
		switch s[4] {
		case 'q':
			promo = Queen
		case 'r':
			promo = Rook
		case 'b':
			promo = Bishop
		case 'n':
			promo = Knight
		default:
			panic(fmt.Sprintf("invalid promotion piece in UCI move: %s", s))
		}
	}

	// Castling detection (optional)
	if from == 4 && to == 6 || from == 60 && to == 62 {
		flags |= MoveCastle
	}
	if from == 4 && to == 2 || from == 60 && to == 58 {
		flags |= MoveCastle
	}

	return Move{From: from, To: to, Promotion: promo, Flags: flags}
}
