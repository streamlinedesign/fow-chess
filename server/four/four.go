// Package four is the rules engine for four-player chess: a 14 by 14 board with
// the corners cut off (a full 8 by 8 in the middle and three rows on each side),
// four players moving clockwise, and the game won by eating kings. A player
// whose king is eaten, who resigns or who runs out of time is out, and their
// pieces leave the board. The last king standing wins.
//
// Like engine, it knows nothing about networking or players, and has no
// dependencies.
package four

import (
	"errors"
	"fmt"
	"strconv"
)

// Size is the width and height of the board.
const Size = 14

// Color is a seat, in turn order: Red at the bottom, Blue on the left, Green at
// the top and Violet on the right, so play goes clockwise.
type Color uint8

const (
	Red Color = iota
	Blue
	Green
	Violet
)

// Colors is every seat in turn order.
var Colors = [4]Color{Red, Blue, Green, Violet}

// colorLetters are the one-letter names used in board strings and the API.
const colorLetters = "rbgv"

func (c Color) String() string { return string(colorLetters[c]) }

// Name is the colour's full name.
func (c Color) Name() string { return [...]string{"red", "blue", "green", "violet"}[c] }

// ParseColor reads a colour letter.
func ParseColor(s string) (Color, bool) {
	for i := range 4 {
		if s == string(colorLetters[i]) {
			return Color(i), true
		}
	}
	return 0, false
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

const kindLetters = " PNBRQK"

func (k Kind) Letter() byte { return kindLetters[k] }

func kindFromLetter(b byte) Kind {
	for i := 1; i < len(kindLetters); i++ {
		if kindLetters[i] == b&^0x20 { // uppercase
			return Kind(i)
		}
	}
	return NoKind
}

// Piece packs a kind and a colour. The zero value is an empty square.
type Piece uint8

const NoPiece Piece = 0

func NewPiece(c Color, k Kind) Piece { return Piece(k) | Piece(c)<<3 }

func (p Piece) Kind() Kind      { return Kind(p & 7) }
func (p Piece) Color() Color    { return Color(p >> 3) }
func (p Piece) IsEmpty() bool   { return p == NoPiece }
func (p Piece) Is(c Color) bool { return p != NoPiece && p.Color() == c }

// Code is the piece as two letters: colour then kind, like "rK" or "vP".
func (p Piece) Code() string {
	return string([]byte{colorLetters[p.Color()], p.Kind().Letter()})
}

// Square indexes the board row by row from the bottom left: x + y*Size. Some
// indexes are the cut-off corners; see Valid.
type Square int16

const NoSquare Square = -1

// NumSquares is the number of indexes, corners included.
const NumSquares = Size * Size

func At(x, y int) Square { return Square(y*Size + x) }

func (s Square) X() int { return int(s) % Size }
func (s Square) Y() int { return int(s) / Size }

// onBoard reports whether (x, y) is a playable square.
func onBoard(x, y int) bool {
	if x < 0 || y < 0 || x >= Size || y >= Size {
		return false
	}
	return !((x < 3 || x > 10) && (y < 3 || y > 10))
}

// Valid reports whether s is a playable square and not a cut-off corner.
func (s Square) Valid() bool { return s >= 0 && s < NumSquares && onBoard(s.X(), s.Y()) }

func (s Square) Offset(dx, dy int) (Square, bool) {
	x, y := s.X()+dx, s.Y()+dy
	if !onBoard(x, y) {
		return NoSquare, false
	}
	return At(x, y), true
}

// String names the square with a file letter a to n and a rank 1 to 14, so
// the bottom left playable square is d1 and the top right k14.
func (s Square) String() string {
	if s == NoSquare {
		return "-"
	}
	return string(rune('a'+s.X())) + strconv.Itoa(s.Y()+1)
}

// ParseSquare reads a square name, like "d1" or "k14".
func ParseSquare(str string) (Square, error) {
	if len(str) < 2 || len(str) > 3 || str[0] < 'a' || str[0] >= 'a'+Size {
		return NoSquare, fmt.Errorf("bad square %q", str)
	}
	n, err := strconv.Atoi(str[1:])
	if err != nil || n < 1 || n > Size || str[1] == '0' {
		return NoSquare, fmt.Errorf("bad square %q", str)
	}
	sq := At(int(str[0]-'a'), n-1)
	if !sq.Valid() {
		return NoSquare, fmt.Errorf("bad square %q", str)
	}
	return sq, nil
}

// forward is the direction each colour's pawns move in.
var forward = [4][2]int{{0, 1}, {1, 0}, {0, -1}, {-1, 0}}

// rankOf is how far forward from c's back rank the square is: 0 on the back
// rank, 1 where the pawns start.
func rankOf(c Color, s Square) int {
	switch c {
	case Red:
		return s.Y()
	case Blue:
		return s.X()
	case Green:
		return Size - 1 - s.Y()
	}
	return Size - 1 - s.X()
}

// PromotionRank is how far up the board a pawn promotes: the eighth rank
// from its own side, just past the middle.
const PromotionRank = 7

// cell is c's square at file f (0 to 7, from that player's left) and rank r.
func cell(c Color, f, r int) Square {
	switch c {
	case Red:
		return At(3+f, r)
	case Blue:
		return At(r, 10-f)
	case Green:
		return At(10-f, Size-1-r)
	}
	return At(Size-1-r, 3+f)
}

// backRank is how every player's pieces stand, from their own left.
var backRank = [8]Kind{Rook, Knight, Bishop, Queen, King, Bishop, Knight, Rook}

// MoveFlag marks the special moves.
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

// UCI is the move as from and to squares, plus the promotion letter: "h2h4",
// "e7e8q", "a7b7".
func (m Move) UCI() string {
	s := m.From.String() + m.To.String()
	if m.Promo != NoKind {
		s += string(m.Promo.Letter() | 0x20)
	}
	return s
}

// ParseUCI splits a move into its squares and promotion, without checking it's legal.
func ParseUCI(s string) (from, to Square, promo Kind, err error) {
	split := func(s string) (string, string) {
		// A square is a letter and one or two digits.
		i := 1
		for i < len(s) && s[i] >= '0' && s[i] <= '9' {
			i++
		}
		return s[:i], s[i:]
	}
	bad := errors.New("bad move " + strconv.Quote(s))
	if len(s) < 4 {
		return NoSquare, NoSquare, NoKind, bad
	}
	a, rest := split(s)
	if len(rest) < 2 {
		return NoSquare, NoSquare, NoKind, bad
	}
	b, rest := split(rest)
	if from, err = ParseSquare(a); err != nil {
		return NoSquare, NoSquare, NoKind, bad
	}
	if to, err = ParseSquare(b); err != nil {
		return NoSquare, NoSquare, NoKind, bad
	}
	switch rest {
	case "":
	case "q", "r", "b", "n":
		promo = kindFromLetter(rest[0])
	default:
		return NoSquare, NoSquare, NoKind, bad
	}
	return from, to, promo, nil
}
