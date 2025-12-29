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

// --------------------------
// Board
// --------------------------
type Board struct {
	// Pieces[color][piece] → bitboard
	Pieces [ColorNB][PieceNB]Bitboard

	// Occupancy per color
	Occupancy [ColorNB]Bitboard

	// All occupied squares
	All Bitboard

	SideToMove Color

	// Castling rights: 0001=WK,0010=WQ,0100=BK,1000=BQ
	Castling uint8

	// En-passant square (0–63), 255 = none
	EnPassant uint8

	HalfMoveClock  uint16
	FullMoveNumber uint16
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
func bit(sq uint8) Bitboard {
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

	// Remove moving piece
	var moved Piece
	for p := Piece(0); p < PieceNB; p++ {
		if b.Pieces[color][p]&fromMask != 0 {
			b.Pieces[color][p] &^= fromMask
			moved = p
			break
		}
	}

	// Capture
	for p := Piece(0); p < PieceNB; p++ {
		b.Pieces[opp][p] &^= toMask
	}

	// Promotion
	if m.Promotion != NoPiece {
		b.Pieces[color][m.Promotion] |= toMask
	} else {
		b.Pieces[color][moved] |= toMask
	}

	b.EnPassant = 255
	b.updateOccupancy()
}

// --------------------------
// Unapply move (undo)
// --------------------------
func (b *Board) unapplyMove(m Move) {
	// For now: simple placeholder. Implement later with MoveState stack.
	// Needed for legality checks and search.
	fmt.Println("unapplyMove: TODO implement")
}

// --------------------------
// High-level MakeMove (game rules)
// --------------------------
func (b *Board) MakeMove(m Move) bool {
	// 1. Pseudo-legal check
	if !b.isPseudoLegal(m) {
		return false
	}

	// Save state for unapply
	prevSide := b.SideToMove
	prevEP := b.EnPassant
	prevCastling := b.Castling
	prevHalf := b.HalfMoveClock
	prevFull := b.FullMoveNumber

	// 2. Apply move
	b.applyMove(m)

	// 3. Flip side
	b.SideToMove ^= 1

	// 4. Check legality: did mover leave own king in check?
	if b.isKingInCheck(prevSide) {
		b.SideToMove = prevSide
		b.EnPassant = prevEP
		b.Castling = prevCastling
		b.HalfMoveClock = prevHalf
		b.FullMoveNumber = prevFull
		b.unapplyMove(m)
		return false
	}

	// 5. Fullmove update
	if b.SideToMove == White {
		b.FullMoveNumber++
	}

	return true
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
