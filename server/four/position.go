package four

// Rules are the mods a four-player game can be played with. Eating the king is
// always the rule: there's no check in four-player chess here.
type Rules struct {
	// JumpKing lets kings also jump like a knight.
	JumpKing bool
	// JesterPawns lets pawns also step and take one square in any direction.
	JesterPawns bool
	// Hill wins the game for the first king to reach one of the four centre
	// squares. Only on the cross: the extended board has no hill.
	Hill bool
	// CenterVision shows the whole board to a player holding a centre square.
	CenterVision CenterVision
	// Sync is Sync Move: every player picks a move each round without seeing
	// the others', the moves happen at once, and captures come last, so a
	// piece that moves away dodges whatever was aimed at it. See PlayRound.
	Sync bool
	// Shape is the board: the cross, or the extended board.
	Shape Shape
}

// CenterVision is a fog rule: a piece on one of the player's outposts (see
// Shape.Outposts) shows its player the whole board for as long as it stays.
type CenterVision uint8

const (
	CenterOff CenterVision = iota
	// CenterPawn counts only a pawn on a centre square.
	CenterPawn
	// CenterAny counts any piece.
	CenterAny
)

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

// StartPosition is every player's pieces on their back two rows, Red to move.
func StartPosition(r Rules) Position {
	p := Position{Turn: Red, Castling: 0xff, EP: NoSquare, EPPawn: NoSquare, Rules: r}
	for _, c := range Colors {
		for f, k := range backRank {
			p.Board[r.Shape.Cell(c, f, 0)] = NewPiece(c, k)
			p.Board[r.Shape.Cell(c, f, 1)] = NewPiece(c, Pawn)
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
func (s Shape) castleSquares(c Color) (king, kingRook, queenRook Square) {
	return s.Cell(c, 4, 0), s.Cell(c, 7, 0), s.Cell(c, 0, 0)
}

// castleMask is the castling rights lost when a piece moves from or to sq.
func (s Shape) castleMask(sq Square) uint8 {
	var mask uint8
	for _, c := range Colors {
		k, kr, qr := s.castleSquares(c)
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
		_, kr, qr := p.Shape.castleSquares(us)
		if m.To == p.Shape.Cell(us, 6, 0) {
			p.Board[kr], p.Board[p.Shape.Cell(us, 5, 0)] = NoPiece, p.Board[kr]
		} else {
			p.Board[qr], p.Board[p.Shape.Cell(us, 3, 0)] = NoPiece, p.Board[qr]
		}
	}
	p.Castling &^= p.Shape.castleMask(m.From) | p.Shape.castleMask(m.To)

	p.EP, p.EPPawn = NoSquare, NoSquare
	if m.Is(FlagDoublePush) && !p.Sync {
		p.EP = At((m.From.X()+m.To.X())/2, (m.From.Y()+m.To.Y())/2)
		p.EPPawn = m.To
	}
	p.Turn = p.next(us)
	if captured.Kind() == King {
		p.Eliminate(captured.Color())
	}
	return p, captured
}

// HillOn reports whether King of the Hill is in play: asked for, and on the
// cross, since the extended board has no hill.
func (r Rules) HillOn() bool { return r.Hill && r.Shape == Cross }
