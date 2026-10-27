package engine

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const StartFEN = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

// Position is a value type: Apply returns a new Position and leaves the old one untouched.
type Position struct {
	Board    [64]Piece
	Turn     Color
	Castling uint8
	EP       Square // square a pawn can capture onto en passant, or NoSquare
	Halfmove int    // plies since the last capture or pawn move
	Fullmove int

	// KingCapture switches off check: any move a piece can make is legal, kings
	// may move into danger, and a game is won by capturing the king. This is a
	// rule setting, not part of the FEN.
	KingCapture bool
}

func StartPosition() Position {
	p, _ := ParseFEN(StartFEN)
	return p
}

func ParseFEN(fen string) (Position, error) {
	var p Position
	f := strings.Fields(fen)
	if len(f) < 4 {
		return p, errors.New("fen: need at least 4 fields")
	}

	rank, file := 7, 0
	for i := 0; i < len(f[0]); i++ {
		c := f[0][i]
		switch {
		case c == '/':
			if file != 8 {
				return p, fmt.Errorf("fen: rank %d has %d files", rank+1, file)
			}
			rank, file = rank-1, 0
		case c >= '1' && c <= '8':
			file += int(c - '0')
		default:
			k := kindFromLetter(c)
			if k == NoKind || file > 7 || rank < 0 {
				return p, fmt.Errorf("fen: bad board at %q", c)
			}
			color := White
			if c >= 'a' {
				color = Black
			}
			p.Board[SquareAt(file, rank)] = NewPiece(color, k)
			file++
		}
	}
	if rank != 0 || file != 8 {
		return p, errors.New("fen: board must have 8 ranks of 8 files")
	}

	switch f[1] {
	case "w":
		p.Turn = White
	case "b":
		p.Turn = Black
	default:
		return p, fmt.Errorf("fen: bad side to move %q", f[1])
	}

	if f[2] != "-" {
		for _, c := range f[2] {
			switch c {
			case 'K':
				p.Castling |= CastleWK
			case 'Q':
				p.Castling |= CastleWQ
			case 'k':
				p.Castling |= CastleBK
			case 'q':
				p.Castling |= CastleBQ
			default:
				return p, fmt.Errorf("fen: bad castling %q", f[2])
			}
		}
	}

	p.EP = NoSquare
	if f[3] != "-" {
		sq, err := ParseSquare(f[3])
		if err != nil {
			return p, fmt.Errorf("fen: %w", err)
		}
		p.EP = sq
	}

	p.Fullmove = 1
	if len(f) >= 6 {
		var err1, err2 error
		p.Halfmove, err1 = strconv.Atoi(f[4])
		p.Fullmove, err2 = strconv.Atoi(f[5])
		if err1 != nil || err2 != nil {
			return p, errors.New("fen: bad move counters")
		}
	}
	return p, nil
}

func (p *Position) FEN() string {
	var b strings.Builder
	p.writeBoard(&b)
	b.WriteByte(' ')
	b.WriteString(p.Turn.String())
	b.WriteByte(' ')
	p.writeCastling(&b)
	b.WriteByte(' ')
	b.WriteString(p.EP.String())
	fmt.Fprintf(&b, " %d %d", p.Halfmove, p.Fullmove)
	return b.String()
}

func (p *Position) writeBoard(b *strings.Builder) {
	for rank := 7; rank >= 0; rank-- {
		empty := 0
		for file := 0; file < 8; file++ {
			pc := p.Board[SquareAt(file, rank)]
			if pc.IsEmpty() {
				empty++
				continue
			}
			if empty > 0 {
				b.WriteByte('0' + byte(empty))
				empty = 0
			}
			b.WriteByte(pc.Letter())
		}
		if empty > 0 {
			b.WriteByte('0' + byte(empty))
		}
		if rank > 0 {
			b.WriteByte('/')
		}
	}
}

func (p *Position) writeCastling(b *strings.Builder) {
	if p.Castling == 0 {
		b.WriteByte('-')
		return
	}
	for i, c := range "KQkq" {
		if p.Castling&(1<<i) != 0 {
			b.WriteRune(c)
		}
	}
}

// repetitionKey identifies a position for the repetition rule: same pieces, side to
// move, castling rights, and en passant only when a capture there is actually legal.
func (p *Position) repetitionKey() string {
	var b strings.Builder
	p.writeBoard(&b)
	b.WriteByte(' ')
	b.WriteString(p.Turn.String())
	b.WriteByte(' ')
	p.writeCastling(&b)
	if p.EP != NoSquare {
		for _, m := range p.LegalMoves() {
			if m.Is(FlagEnPassant) {
				b.WriteByte(' ')
				b.WriteString(p.EP.String())
				break
			}
		}
	}
	return b.String()
}

func (p *Position) KingSquare(c Color) Square {
	k := NewPiece(c, King)
	for sq, pc := range p.Board {
		if pc == k {
			return Square(sq)
		}
	}
	return NoSquare
}

// InCheck reports whether the side to move is in check. Always false when
// playing without check.
func (p *Position) InCheck() bool {
	if p.KingCapture {
		return false
	}
	k := p.KingSquare(p.Turn)
	return k != NoSquare && p.Attacked(k, p.Turn.Other())
}

// Apply plays m, which must come from this position's move generator, and returns the result.
func (p Position) Apply(m Move) Position {
	pc := p.Board[m.From]
	us := pc.Color()

	p.Halfmove++
	if pc.Kind() == Pawn || m.Is(FlagCapture) {
		p.Halfmove = 0
	}

	p.Board[m.From] = NoPiece
	if m.Is(FlagEnPassant) {
		p.Board[SquareAt(m.To.File(), m.From.Rank())] = NoPiece
	}
	if m.Promo != NoKind {
		pc = NewPiece(us, m.Promo)
	}
	p.Board[m.To] = pc

	if m.Is(FlagCastle) {
		rank := m.From.Rank()
		if m.To.File() == 6 {
			p.Board[SquareAt(7, rank)], p.Board[SquareAt(5, rank)] = NoPiece, p.Board[SquareAt(7, rank)]
		} else {
			p.Board[SquareAt(0, rank)], p.Board[SquareAt(3, rank)] = NoPiece, p.Board[SquareAt(0, rank)]
		}
	}

	p.Castling &^= castleMask(m.From) | castleMask(m.To)

	p.EP = NoSquare
	if m.Is(FlagDoublePush) {
		p.EP = SquareAt(m.From.File(), (m.From.Rank()+m.To.Rank())/2)
	}

	if us == Black {
		p.Fullmove++
	}
	p.Turn = us.Other()
	return p
}

// castleMask is the castling rights lost when a piece moves from or to sq.
func castleMask(sq Square) uint8 {
	switch sq {
	case 4: // e1
		return CastleWK | CastleWQ
	case 60: // e8
		return CastleBK | CastleBQ
	case 0: // a1
		return CastleWQ
	case 7: // h1
		return CastleWK
	case 56: // a8
		return CastleBQ
	case 63: // h8
		return CastleBK
	}
	return 0
}
