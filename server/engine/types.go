// Package engine implements chess rules: positions, move generation, legality
// and game results. It knows nothing about networking or players, so variants
// such as fog of war can be layered on top of it.
package engine

import "fmt"

type Color uint8

const (
	White Color = iota
	Black
)

func (c Color) Other() Color { return c ^ 1 }

func (c Color) String() string {
	if c == White {
		return "w"
	}
	return "b"
}

type Kind uint8

const (
	NoKind Kind = iota
	Pawn
	Knight
	Bishop
	Rook
	Queen
	King
)

// kindLetters is indexed by Kind and gives the FEN/SAN letter (uppercase).
const kindLetters = " PNBRQK"

func (k Kind) Letter() byte { return kindLetters[k] }

func kindFromLetter(b byte) Kind {
	switch b | 0x20 { // lowercase
	case 'p':
		return Pawn
	case 'n':
		return Knight
	case 'b':
		return Bishop
	case 'r':
		return Rook
	case 'q':
		return Queen
	case 'k':
		return King
	}
	return NoKind
}

// Piece packs a kind and a color into one byte. The zero value is an empty square.
type Piece uint8

const NoPiece Piece = 0

func NewPiece(c Color, k Kind) Piece { return Piece(k) | Piece(c)<<3 }

func (p Piece) Kind() Kind      { return Kind(p & 7) }
func (p Piece) Color() Color    { return Color(p >> 3) }
func (p Piece) IsEmpty() bool   { return p == NoPiece }
func (p Piece) Is(c Color) bool { return p != NoPiece && p.Color() == c }

// FEN letter: uppercase for white, lowercase for black.
func (p Piece) Letter() byte {
	l := p.Kind().Letter()
	if p.Color() == Black {
		l |= 0x20
	}
	return l
}

// Square indexes the board from a1 = 0 to h8 = 63.
type Square int8

const NoSquare Square = -1

func SquareAt(file, rank int) Square { return Square(rank*8 + file) }

func (s Square) File() int { return int(s) & 7 }
func (s Square) Rank() int { return int(s) >> 3 }

// Offset returns the square df files and dr ranks away, or false if that's off the board.
func (s Square) Offset(df, dr int) (Square, bool) {
	f, r := s.File()+df, s.Rank()+dr
	if f < 0 || f > 7 || r < 0 || r > 7 {
		return NoSquare, false
	}
	return SquareAt(f, r), true
}

func (s Square) String() string {
	if s == NoSquare {
		return "-"
	}
	return string([]byte{'a' + byte(s.File()), '1' + byte(s.Rank())})
}

func ParseSquare(str string) (Square, error) {
	if len(str) != 2 || str[0] < 'a' || str[0] > 'h' || str[1] < '1' || str[1] > '8' {
		return NoSquare, fmt.Errorf("bad square %q", str)
	}
	return SquareAt(int(str[0]-'a'), int(str[1]-'1')), nil
}

type MoveFlag uint8

const (
	FlagCapture MoveFlag = 1 << iota
	FlagEnPassant
	FlagCastle
	FlagDoublePush
)

type Move struct {
	From, To Square
	Promo    Kind
	Flags    MoveFlag
}

func (m Move) Is(f MoveFlag) bool { return m.Flags&f != 0 }

// UCI returns the move in long algebraic form, e.g. "e2e4" or "e7e8q".
func (m Move) UCI() string {
	s := m.From.String() + m.To.String()
	if m.Promo != NoKind {
		s += string(m.Promo.Letter() | 0x20)
	}
	return s
}

func (m Move) String() string { return m.UCI() }

// Castling rights bits.
const (
	CastleWK uint8 = 1 << iota
	CastleWQ
	CastleBK
	CastleBQ
)
