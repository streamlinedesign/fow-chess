package engine

// Vision returns the squares color c can see in fog of war: each of its pieces
// sees its own square and every square it could move to, captures included.
// Pawns have their own vision, apart from how they move: a pawn always sees the
// square straight ahead, even when blocked, and on its starting rank the square
// after that too, even when it couldn't step there. It sees its diagonals only
// when it can capture there. When pawns can step back, they see the square
// straight behind them the same way.
//
// With CenterVision, a player holding an outpost sees the whole board.
func (p *Position) Vision(c Color) [64]bool {
	var seen [64]bool
	if p.Outpost(c) != NoSquare {
		for i := range seen {
			seen[i] = true
		}
		return seen
	}
	for i, pc := range p.Board {
		if !pc.Is(c) {
			continue
		}
		sq := Square(i)
		seen[sq] = true
		if pc.Kind() == Pawn {
			if ahead, ok := sq.Offset(0, pawnDir(c)); ok {
				seen[ahead] = true
				if sq.Rank() == pawnStartRank(c) {
					if two, ok := ahead.Offset(0, pawnDir(c)); ok {
						seen[two] = true
					}
				}
			}
			if behind, ok := sq.Offset(0, -pawnDir(c)); ok && p.BackPawns {
				seen[behind] = true
			}
		}
		for _, m := range p.PieceMoves(sq) {
			seen[m.To] = true
		}
	}
	return seen
}

// CenterVision is a fog rule: a piece on the centre of the opponent's half
// (an outpost) shows its player the whole board for as long as it stays.
type CenterVision uint8

const (
	CenterOff CenterVision = iota
	// CenterPawn counts only a pawn on an outpost.
	CenterPawn
	// CenterAny counts any piece on an outpost.
	CenterAny
)

// Outposts are the centre squares of c's opponent's half: d5 and e5 for
// White, d4 and e4 for Black.
func Outposts(c Color) [2]Square {
	if c == White {
		return [2]Square{SquareAt(3, 4), SquareAt(4, 4)}
	}
	return [2]Square{SquareAt(3, 3), SquareAt(4, 3)}
}

// Outpost is the square of c's piece that gives it the whole board under
// CenterVision, or NoSquare.
func (p *Position) Outpost(c Color) Square {
	if p.CenterVision == CenterOff {
		return NoSquare
	}
	for _, sq := range Outposts(c) {
		pc := p.Board[sq]
		if pc.Is(c) && (p.CenterVision == CenterAny || pc.Kind() == Pawn) {
			return sq
		}
	}
	return NoSquare
}
