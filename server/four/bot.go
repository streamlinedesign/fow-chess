package four

import "math/rand/v2"

// pieceValue is what a piece is worth to the bot.
var pieceValue = [...]float64{0, 1, 3, 3, 5, 9, 1000}

// BotMove picks a move for the player to move, using only what that player
// can see: eat a king when it can, take what it can without losing more,
// keep its pieces out of reach of the pieces it sees, and push towards the
// middle. It's a sparring partner for filling empty seats, not a strong player.
func BotMove(p *Position, rng *rand.Rand) (Move, bool) {
	moves := p.LegalMoves()
	if len(moves) == 0 {
		return Move{}, false
	}
	us := p.Turn
	seen := p.Vision(us)
	kingSq := p.KingSquare(us)
	kingInDanger := kingSq != NoSquare && p.Attacked(kingSq, us, nil)
	best, bestScore := moves[0], -1e9
	for _, m := range moves {
		pc := p.Board[m.From]
		score := rng.Float64() * 0.3
		target := p.Board[m.To]
		if m.Is(FlagEnPassant) {
			target = NewPiece(Red, Pawn)
		}
		score += pieceValue[target.Kind()]
		if p.Hill && pc.Kind() == King && InCentre(m.To) {
			score += 1000
		}
		if m.Promo != NoKind {
			score += pieceValue[m.Promo] - 1
		}
		after, _ := p.Apply(m)
		mover := after.Board[m.To]
		if after.Attacked(m.To, us, &seen) {
			score -= pieceValue[mover.Kind()] * 0.9
		}
		if kingInDanger {
			if k := after.KingSquare(us); k != NoSquare && !after.Attacked(k, us, &seen) {
				score += 50
			}
		}
		// Head for the middle, pawns most of all.
		dx, dy := float64(m.To.X())-6.5, float64(m.To.Y())-6.5
		fx, fy := float64(m.From.X())-6.5, float64(m.From.Y())-6.5
		closer := (fx*fx + fy*fy - dx*dx - dy*dy) / 40
		if pc.Kind() == Pawn {
			closer += 0.15
		}
		if pc.Kind() != King {
			score += closer
		}
		if score > bestScore {
			best, bestScore = m, score
		}
	}
	return best, true
}
