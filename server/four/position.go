package four

// Rules are the mods a four-player game can be played with. Eating the king is
// always the rule: there's no check in four-player chess here.
type Rules struct {
	// JumpKing lets kings also jump like a knight.
	JumpKing bool
	// BackPawns lets pawns also step one square straight back.
	BackPawns bool
	// JesterPawns lets pawns also step and take one square in any direction.
	JesterPawns bool
	// Hill wins the game for the first king to reach one of the four centre squares.
	Hill bool
	// CenterVision shows the whole board to a player holding a centre square.
	CenterVision CenterVision
}

// CenterVision is a fog rule: a piece on one of the four centre squares shows
// its player the whole board for as long as it stays.
type CenterVision uint8

const (
	CenterOff CenterVision = iota
	// CenterPawn counts only a pawn on a centre square.
	CenterPawn
	// CenterAny counts any piece.
	CenterAny
)

// Centre is the four middle squares: the hill, and the outposts that give
// centre vision. Every player shares them.
var Centre = [4]Square{At(6, 6), At(7, 6), At(6, 7), At(7, 7)}

// InCentre reports whether sq is one of the four middle squares.
func InCentre(sq Square) bool {
	for _, c := range Centre {
		if c == sq {
			return true
		}
	}
	return false
}

// Position is a value type: Apply returns a new Position and leaves the old one untouched.
type Position struct {
	Board [NumSquares]Piece
	Turn  Color
	// Out marks the players who are out of the game.
	Out [4]bool
	// Castling holds two bits per colour: 1<<(2c) for the king's side, 1<<(2c+1) for the queen's.
	Castling uint8
	// EP is the square a pawn just skipped with a double step, and EPPawn
	// where that pawn stands; the next player may take it en passant.
	EP, EPPawn Square
	Rules
}

// StartPosition is every player's pieces in their three rows, Red to move.
func StartPosition(r Rules) Position {
	p := Position{Turn: Red, Castling: 0xff, EP: NoSquare, EPPawn: NoSquare, Rules: r}
	for _, c := range Colors {
		for f, k := range backRank {
			p.Board[cell(c, f, 0)] = NewPiece(c, k)
			p.Board[cell(c, f, 1)] = NewPiece(c, Pawn)
		}
	}
	return p
}

// KingSquare is where c's king is, or NoSquare.
func (p *Position) KingSquare(c Color) Square {
	k := NewPiece(c, King)
	for sq, pc := range p.Board {
		if pc == k {
			return Square(sq)
		}
	}
	return NoSquare
}

// Alive is how many players are still in.
func (p *Position) Alive() int {
	n := 0
	for _, out := range p.Out {
		if !out {
			n++
		}
	}
	return n
}

// next is the first player still in after c, in turn order.
func (p *Position) next(c Color) Color {
	for i := 1; i <= 4; i++ {
		n := Color((int(c) + i) % 4)
		if !p.Out[n] {
			return n
		}
	}
	return c
}

// Eliminate takes c out of the game: its pieces leave the board, and if it was
// c's turn the turn passes on.
func (p *Position) Eliminate(c Color) {
	if p.Out[c] {
		return
	}
	p.Out[c] = true
	for sq, pc := range p.Board {
		if pc.Is(c) {
			p.Board[sq] = NoPiece
		}
	}
	p.Castling &^= 3 << (2 * c)
	if p.EPPawn != NoSquare && p.Board[p.EPPawn].IsEmpty() {
		p.EP, p.EPPawn = NoSquare, NoSquare
	}
	if p.Turn == c {
		p.Turn = p.next(c)
	}
}

// castleSquares are, for each colour, the king's start square and the rooks' on
// the king's and the queen's side.
func castleSquares(c Color) (king, kingRook, queenRook Square) {
	return cell(c, 4, 0), cell(c, 7, 0), cell(c, 0, 0)
}

// castleMask is the castling rights lost when a piece moves from or to sq.
func castleMask(sq Square) uint8 {
	var mask uint8
	for _, c := range Colors {
		k, kr, qr := castleSquares(c)
		switch sq {
		case k:
			mask |= 3 << (2 * c)
		case kr:
			mask |= 1 << (2 * c)
		case qr:
			mask |= 2 << (2 * c)
		}
	}
	return mask
}

// Apply plays m, which must come from this position's move generator, and
// returns the result with the piece it took. Eating a king takes that player
// out of the game.
func (p Position) Apply(m Move) (Position, Piece) {
	pc := p.Board[m.From]
	us := pc.Color()
	captured := p.Board[m.To]
	if m.Is(FlagEnPassant) {
		captured = p.Board[p.EPPawn]
		p.Board[p.EPPawn] = NoPiece
	}
	p.Board[m.From] = NoPiece
	if m.Promo != NoKind {
		pc = NewPiece(us, m.Promo)
	}
	p.Board[m.To] = pc

	if m.Is(FlagCastle) {
		_, kr, qr := castleSquares(us)
		if m.To == cell(us, 6, 0) {
			p.Board[kr], p.Board[cell(us, 5, 0)] = NoPiece, p.Board[kr]
		} else {
			p.Board[qr], p.Board[cell(us, 3, 0)] = NoPiece, p.Board[qr]
		}
	}
	p.Castling &^= castleMask(m.From) | castleMask(m.To)

	p.EP, p.EPPawn = NoSquare, NoSquare
	if m.Is(FlagDoublePush) {
		p.EP = At((m.From.X()+m.To.X())/2, (m.From.Y()+m.To.Y())/2)
		p.EPPawn = m.To
	}
	p.Turn = p.next(us)
	if captured.Kind() == King {
		p.Eliminate(captured.Color())
	}
	return p, captured
}
