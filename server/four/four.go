// Package four is the rules engine for four-player chess: a 14 by 14 board with
// the corners cut off (a full 8 by 8 in the middle and three rows on each side),
// or the extended 16 by 16 board (four rows on each side, and the corners filled
// with neutral squares); see Shape.
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

// Size is the width and height of the largest board; squares are indexed on
// a grid this size whatever the board's shape.
const Size = 16

// Shape is the board's shape.
type Shape uint8

const (
	// Cross is the 14 by 14 board with the corners cut off: an 8 by 8 middle
	// and three rows for each player.
	Cross Shape = iota
	// Extended is 16 by 16: four rows for each player, and the corners, four by
	// four, are neutral squares anyone may cross.
	Extended
)

// Width is how many squares across the board is.
func (s Shape) Width() int { return [...]int{14, 16}[s] }

// Rows is how many rows each player starts in.
func (s Shape) Rows() int { return [...]int{3, 4}[s] }

// Contains reports whether (x, y) is a square of the board.
func (s Shape) Contains(x, y int) bool {
	w, a := s.Width(), s.Rows()
	if x < 0 || y < 0 || x >= w || y >= w {
		return false
	}
	return s == Extended || !((x < a || x >= w-a) && (y < a || y >= w-a))
}

// Valid reports whether sq is a square of the board.
func (s Shape) Valid(sq Square) bool { return sq >= 0 && sq < NumSquares && s.Contains(sq.X(), sq.Y()) }

// Offset is the square dx, dy away from sq, if it's on the board.
func (s Shape) Offset(sq Square, dx, dy int) (Square, bool) {
	x, y := sq.X()+dx, sq.Y()+dy
	if !s.Contains(x, y) {
		return NoSquare, false
	}
	return At(x, y), true
}

// Cell is c's square at file f (0 to 7, from that player's left) and rank r.
func (s Shape) Cell(c Color, f, r int) Square {
	w, a := s.Width(), s.Rows()
	switch c {
	case Red:
		return At(a+f, r)
	case Blue:
		return At(r, a+7-f)
	case Green:
		return At(a+7-f, w-1-r)
	}
	return At(w-1-r, a+f)
}

// RankOf is how far forward from c's back rank sq is: 0 on the back rank, 1
// where the pawns start.
func (s Shape) RankOf(c Color, sq Square) int {
	w := s.Width()
	switch c {
	case Red:
		return sq.Y()
	case Blue:
		return sq.X()
	case Green:
		return w - 1 - sq.Y()
	}
	return w - 1 - sq.X()
}

// PromotionRank is how far up the board a pawn promotes: on the cross, the
// eighth rank from its own side, just past the middle; on the extended
// board, the far edge.
func (s Shape) PromotionRank() int { return [...]int{7, 15}[s] }

// Centre is the four middle squares.
func (s Shape) Centre() [4]Square {
	m := s.Width()/2 - 1
	return [4]Square{At(m, m), At(m+1, m), At(m, m+1), At(m+1, m+1)}
}

// InCentre reports whether sq is one of the four middle squares.
func (s Shape) InCentre(sq Square) bool {
	for _, c := range s.Centre() {
		if c == sq {
			return true
		}
	}
	return false
}

// Outposts are the squares that give c the whole board under centre vision.
// On the cross every player shares the four middle squares. On the extended
// board each player has two of them, like d5 and e5 in a two-player game:
// the pair just past the middle on the far side, side by side as that player
// sees the board.
func (s Shape) Outposts(c Color) []Square {
	ctr := s.Centre() // bottom left, bottom right, top left, top right
	if s == Cross {
		return ctr[:]
	}
	return [4][]Square{
		Red:    {ctr[2], ctr[3]},
		Blue:   {ctr[1], ctr[3]},
		Green:  {ctr[0], ctr[1]},
		Violet: {ctr[0], ctr[2]},
	}[c]
}

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

// Square indexes a Size by Size grid row by row from the bottom left:
// x + y*Size. Which of them are on the board is the Shape's to say.
type Square int16

const NoSquare Square = -1

// NumSquares is the number of indexes, corners included.
const NumSquares = Size * Size

func At(x, y int) Square { return Square(y*Size + x) }

func (s Square) X() int { return int(s) % Size }
func (s Square) Y() int { return int(s) / Size }

// String names the square with a file letter a to p and a rank 1 to 16: on the
// cross the bottom left playable square is d1 and the top right k14; on the
// extended board a1 to p16.
func (s Square) String() string {
	if s == NoSquare {
		return "-"
	}
	return string(rune('a'+s.X())) + strconv.Itoa(s.Y()+1)
}

// ParseSquare reads a square name, like "d1" or "k14". Whether the square is
// on a given board is the board's to say; see Shape.Valid.
func ParseSquare(str string) (Square, error) {
	if len(str) < 2 || len(str) > 3 || str[0] < 'a' || str[0] >= 'a'+Size {
		return NoSquare, fmt.Errorf("bad square %q", str)
	}
	n, err := strconv.Atoi(str[1:])
	if err != nil || n < 1 || n > Size || str[1] == '0' {
		return NoSquare, fmt.Errorf("bad square %q", str)
	}
	return At(int(str[0]-'a'), n-1), nil
}

// forward is the direction each colour's pawns move in.
var forward = [4][2]int{{0, 1}, {1, 0}, {0, -1}, {-1, 0}}

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
