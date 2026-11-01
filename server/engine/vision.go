package engine

// Vision returns the squares color c can see in fog of war: each of its pieces
// sees its own square and every square it could move to, captures included.
// Pawns have their own vision, apart from how they move: a pawn always sees the
// square straight ahead, even when blocked, and on its starting rank the square
// after that too, even when it couldn't step there. It sees its diagonals only
// when it can capture there. When pawns can step back, they see the square
// straight behind them the same way.
func (p *Position) Vision(c Color) [64]bool {
	var seen [64]bool
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
