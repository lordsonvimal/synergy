package engine

import "fmt"

// --------------------------
// Color
// --------------------------
type Color uint8

const (
	White Color = 0
	Black Color = 1
)

const ColorNB = 2

// --------------------------
// Piece
// --------------------------
type Piece uint8

const (
	Pawn Piece = iota
	Knight
	Bishop
	Rook
	Queen
	King
)

const PieceNB = 6
const NoPiece Piece = 255

// Piece letters for FEN
var pieceToFEN = map[Piece]map[Color]string{
	Pawn:    {White: "P", Black: "p"},
	Knight:  {White: "N", Black: "n"},
	Bishop:  {White: "B", Black: "b"},
	Rook:    {White: "R", Black: "r"},
	Queen:   {White: "Q", Black: "q"},
	King:    {White: "K", Black: "k"},
	NoPiece: {White: "", Black: ""},
}

type MoveState struct {
	From, To      uint8
	MovingPiece   Piece
	CapturedPiece Piece
	Promotion     Piece
	PrevEP        uint8
	PrevCastling  uint8
	PrevHalfMove  uint16
	PrevFullMove  uint16
	PrevHash      uint64
	Flags         uint8 // MoveNormal, MoveCastle, MoveEP, MovePromo
}

// --------------------------
// Board
// --------------------------
type Board struct {
	// Pieces[color][piece] → bitboard
	Pieces [ColorNB][PieceNB]uint64

	// Occupancy per color
	Occupancy [ColorNB]uint64

	// All occupied squares
	All uint64

	SideToMove Color

	// Castling rights: 0001=WK,0010=WQ,0100=BK,1000=BQ
	Castling uint8

	// En-passant square (0–63), 255 = none
	EnPassant uint8

	HalfMoveClock  uint16
	FullMoveNumber uint16

	Hash uint64 // Zobrist hash

	MoveStack []MoveState
}

// --------------------------
// Constructor
// --------------------------
func NewBoard() *Board {
	b := &Board{}
	b.Reset()
	return b
}

// --------------------------
// Reset board to initial position
// --------------------------
func (b *Board) Reset() {
	// Pawns
	b.Pieces[White][Pawn] = 0x000000000000FF00
	b.Pieces[Black][Pawn] = 0x00FF000000000000

	// Rooks
	b.Pieces[White][Rook] = 0x0000000000000081
	b.Pieces[Black][Rook] = 0x8100000000000000

	// Knights
	b.Pieces[White][Knight] = 0x0000000000000042
	b.Pieces[Black][Knight] = 0x4200000000000000

	// Bishops
	b.Pieces[White][Bishop] = 0x0000000000000024
	b.Pieces[Black][Bishop] = 0x2400000000000000

	// Queens
	b.Pieces[White][Queen] = 0x0000000000000008
	b.Pieces[Black][Queen] = 0x0800000000000000

	// Kings
	b.Pieces[White][King] = 0x0000000000000010
	b.Pieces[Black][King] = 0x1000000000000000

	b.updateOccupancy()

	b.SideToMove = White
	b.Castling = 0b1111
	b.EnPassant = 255
	b.HalfMoveClock = 0
	b.FullMoveNumber = 1
}

// --------------------------
// Bit helpers
// --------------------------
func bit(sq uint8) uint64 {
	return 1 << sq
}

// --------------------------
// Update Occupancy
// --------------------------
func (b *Board) updateOccupancy() {
	b.Occupancy[White] = 0
	b.Occupancy[Black] = 0

	for p := Piece(0); p < PieceNB; p++ {
		b.Occupancy[White] |= b.Pieces[White][p]
		b.Occupancy[Black] |= b.Pieces[Black][p]
	}

	b.All = b.Occupancy[White] | b.Occupancy[Black]
}

// --------------------------
// Piece at square
// --------------------------
func (b *Board) PieceAt(sq uint8) (Color, Piece, bool) {
	mask := bit(sq)

	for c := Color(0); c < ColorNB; c++ {
		for p := Piece(0); p < PieceNB; p++ {
			if b.Pieces[c][p]&mask != 0 {
				return c, p, true
			}
		}
	}
	return 0, NoPiece, false
}

// --------------------------
// Apply move (low-level, mutates bitboards)
// --------------------------
func (b *Board) applyMove(m Move) {
	fromMask := bit(m.From)
	toMask := bit(m.To)

	color := b.SideToMove
	opp := color ^ 1

	// --------------------
	// Identify moving piece
	// --------------------
	var moved Piece = NoPiece
	for p := Piece(0); p < PieceNB; p++ {
		if b.Pieces[color][p]&fromMask != 0 {
			b.Pieces[color][p] &^= fromMask
			moved = p
			break
		}
	}

	// --------------------
	// Capture handling
	// --------------------
	captured := NoPiece

	if m.Flags&MoveEP != 0 {
		var capSq uint8
		if color == White {
			capSq = m.To - 8
		} else {
			capSq = m.To + 8
		}
		b.Pieces[opp][Pawn] &^= bit(capSq)
		captured = Pawn
	} else {
		for p := Piece(0); p < PieceNB; p++ {
			if b.Pieces[opp][p]&toMask != 0 {
				b.Pieces[opp][p] &^= toMask
				captured = p
				break
			}
		}
	}

	// --------------------
	// Promotion / normal move
	// --------------------
	if m.Flags&MovePromo != 0 {
		b.Pieces[color][m.Promotion] |= toMask
	} else {
		b.Pieces[color][moved] |= toMask
	}

	// --------------------
	// Castling rook move
	// --------------------
	if m.Flags&MoveCastle != 0 {
		switch m.To {
		case 62: // White king side
			b.Pieces[White][Rook] &^= bit(63)
			b.Pieces[White][Rook] |= bit(61)
		case 58: // White queen side
			b.Pieces[White][Rook] &^= bit(56)
			b.Pieces[White][Rook] |= bit(59)
		case 6: // Black king side
			b.Pieces[Black][Rook] &^= bit(7)
			b.Pieces[Black][Rook] |= bit(5)
		case 2: // Black queen side
			b.Pieces[Black][Rook] &^= bit(0)
			b.Pieces[Black][Rook] |= bit(3)
		}
	}

	// --------------------
	// Castling rights update
	// --------------------
	if moved == King {
		if color == White {
			b.Castling &^= 0b0011
		} else {
			b.Castling &^= 0b1100
		}
	}

	if moved == Rook || captured == Rook {
		switch m.From {
		case 63:
			b.Castling &^= 0b0001
		case 56:
			b.Castling &^= 0b0010
		case 7:
			b.Castling &^= 0b0100
		case 0:
			b.Castling &^= 0b1000
		}
		switch m.To {
		case 63:
			b.Castling &^= 0b0001
		case 56:
			b.Castling &^= 0b0010
		case 7:
			b.Castling &^= 0b0100
		case 0:
			b.Castling &^= 0b1000
		}
	}

	// --------------------
	// En-passant square
	// --------------------
	b.EnPassant = NoSquare
	diff := int(m.To) - int(m.From)
	if moved == Pawn && (diff == 16 || diff == -16) {
		if color == White {
			b.EnPassant = m.From + 8
		} else {
			b.EnPassant = m.From - 8
		}
	}

	// --------------------
	// Halfmove clock
	// --------------------
	if moved == Pawn || captured != NoPiece {
		b.HalfMoveClock = 0
	} else {
		b.HalfMoveClock++
	}

	// --------------------
	// Update occupancy
	// --------------------
	b.updateOccupancy()
}

// --------------------------
// Unapply move (undo)
// --------------------------
func (b *Board) unapplyMove() {
	if len(b.MoveStack) == 0 {
		panic("unapplyMove: no moves to undo")
	}

	// Pop last move
	state := b.MoveStack[len(b.MoveStack)-1]
	b.MoveStack = b.MoveStack[:len(b.MoveStack)-1]

	color := b.SideToMove ^ 1 // The side that actually moved
	opp := color ^ 1

	// 1. Remove moving piece from destination
	if state.Flags&MovePromo != 0 {
		// remove promoted piece
		b.Pieces[color][state.Promotion] &^= 1 << state.To
	} else {
		b.Pieces[color][state.MovingPiece] &^= 1 << state.To
	}

	// 2. Restore moving piece to original square
	b.Pieces[color][state.MovingPiece] |= 1 << state.From

	// 3. Restore captured piece
	if state.CapturedPiece != NoPiece {
		if state.Flags&MoveEP != 0 {
			// En-passant capture: captured pawn is behind the destination square
			var capSq uint8
			if color == White {
				capSq = state.To - 8
			} else {
				capSq = state.To + 8
			}
			b.Pieces[opp][state.CapturedPiece] |= 1 << capSq
		} else {
			// Normal capture
			b.Pieces[opp][state.CapturedPiece] |= 1 << state.To
		}
	}

	// 4. Restore rook for castling
	if state.Flags&MoveCastle != 0 {
		switch state.To {
		case 62: // White kingside
			b.Pieces[White][Rook] &^= 1 << 61
			b.Pieces[White][Rook] |= 1 << 63
		case 58: // White queenside
			b.Pieces[White][Rook] &^= 1 << 59
			b.Pieces[White][Rook] |= 1 << 56
		case 6: // Black kingside
			b.Pieces[Black][Rook] &^= 1 << 5
			b.Pieces[Black][Rook] |= 1 << 7
		case 2: // Black queenside
			b.Pieces[Black][Rook] &^= 1 << 3
			b.Pieces[Black][Rook] |= 1 << 0
		}
	}

	// 5. Recalculate occupancy
	b.updateOccupancy()

	// 6. Restore en passant, castling rights, half/full move counters
	b.EnPassant = state.PrevEP
	b.Castling = state.PrevCastling
	b.HalfMoveClock = state.PrevHalfMove
	b.FullMoveNumber = state.PrevFullMove

	// 7. Restore Zobrist hash
	b.Hash = state.PrevHash

	// 8. Flip side back
	b.SideToMove ^= 1
}

// --------------------------
// High-level MakeMove (game rules)
// --------------------------
func (b *Board) MakeMove(m Move) bool {
	// 1. Identify moving piece and captured piece
	movingPiece := b.pieceOnSquare(m.From)
	captured := b.pieceOnSquare(m.To)
	promoted := m.Promotion

	// 2. Set move flags
	m.Flags = MoveNormal
	if promoted != NoPiece {
		m.Flags |= MovePromo
	}
	if movingPiece == King && (m.To == m.From+2 || m.To == m.From-2) {
		m.Flags |= MoveCastle
	}
	if movingPiece == Pawn && captured == NoPiece && m.From%8 != m.To%8 {
		m.Flags |= MoveEP
	}

	// 3. Efficient legality check BEFORE committing
	if !b.isPseudoLegalEfficient(m) {
		return false
	}

	// 4. Save current state for undo
	prevSide := b.SideToMove
	prevEP := b.EnPassant
	prevCastling := b.Castling
	prevHalf := b.HalfMoveClock
	prevFull := b.FullMoveNumber
	prevHash := b.Hash

	// 5. Apply move
	b.applyMove(m)

	// 6. Update hash
	b.Hash = b.UpdateHash(m, prevSide, captured, promoted)

	// 7. Save MoveState for undo
	state := MoveState{
		From:          m.From,
		To:            m.To,
		MovingPiece:   movingPiece,
		CapturedPiece: captured,
		Promotion:     promoted,
		PrevEP:        prevEP,
		PrevCastling:  prevCastling,
		PrevHalfMove:  prevHalf,
		PrevFullMove:  prevFull,
		PrevHash:      prevHash,
		Flags:         m.Flags,
	}
	b.MoveStack = append(b.MoveStack, state)

	// 8. Flip side
	b.SideToMove ^= 1

	// 9. Update fullmove number
	if b.SideToMove == White {
		b.FullMoveNumber++
	}

	return true
}

// Helper: get piece on a square
func (b *Board) pieceOnSquare(sq uint8) Piece {
	for color := Color(0); color < 2; color++ {
		for p := Piece(0); p < PieceNB; p++ {
			if b.Pieces[color][p]&(1<<sq) != 0 {
				return p
			}
		}
	}
	return NoPiece
}

// String returns board position in FEN-like notation
func (b *Board) String() string {
	boardStr := ""

	for rank := 7; rank >= 0; rank-- {
		empty := 0
		for file := 0; file < 8; file++ {
			sq := uint8(rank*8 + file)
			found := false
			for color := Color(0); color < ColorNB; color++ {
				for p := Piece(0); p < PieceNB; p++ {
					if b.Pieces[color][p]&(1<<sq) != 0 {
						if empty > 0 {
							boardStr += fmt.Sprintf("%d", empty)
							empty = 0
						}
						boardStr += pieceToFEN[p][color]
						found = true
						break
					}
				}
				if found {
					break
				}
			}
			if !found {
				empty++
			}
		}
		if empty > 0 {
			boardStr += fmt.Sprintf("%d", empty)
		}
		if rank > 0 {
			boardStr += "/"
		}
	}

	// Side to move
	side := "w"
	if b.SideToMove == Black {
		side = "b"
	}

	// Castling rights placeholder (implement properly later)
	castling := "-"
	// En passant placeholder
	enpassant := "-"
	// Halfmove clock, fullmove number
	halfmove := 0
	fullmove := b.FullMoveNumber

	return fmt.Sprintf("%s %s %s %s %d %d", boardStr, side, castling, enpassant, halfmove, fullmove)
}

// --------------------------
// Board debug helper
// --------------------------
func (b *Board) PrintBoard() {
	fmt.Printf("Side: %v, FullMove: %d\n", b.SideToMove, b.FullMoveNumber)
	for r := 7; r >= 0; r-- {
		for f := 0; f < 8; f++ {
			sq := uint8(r*8 + f)
			c, p, ok := b.PieceAt(sq)
			if ok {
				fmt.Printf("%c", pieceChar(c, p))
			} else {
				fmt.Print(".")
			}
		}
		fmt.Println()
	}
	fmt.Println()
}

func pieceChar(c Color, p Piece) byte {
	base := "PNBRQK"
	ch := base[p]
	if c == Black {
		ch += 32
	}
	return ch
}
