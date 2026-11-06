package four

import "math/rand/v2"

// pieceValue is what a piece is worth to the bot.
var pieceValue = [...]float64{0, 1, 3, 3, 5, 9, 1000}

// BotMove picks a move for the player to move, seeing the whole board, at
// the strongest level.
func BotMove(p *Position, rng *rand.Rand) (Move, bool) {
	return BotMoveSeeing(p, nil, Strongest, rng)
}

// Strength is how well a four-player bot plays.
type Strength int

const (
	// Greedy takes what it can and keeps its pieces out of reach of what it
	// sees, one move at a time: the bot of before 2026-11.
	Greedy Strength = iota + 1
	// Careful weighs everything that can happen in the round (see
	// BotMoveSeeing), but not always well.
	Careful
	// Strongest weighs it all every move.
	Strongest
)

// BotMoveSeeing picks a move for the player to move, who sees the squares in
// seen (nil for the whole board): pieces of other players it can't see are
// left out of its thinking, so it never cheats.
//
// It looks one round ahead and weighs what can happen in it. In Sync Move the
// others choose their moves from the same position at the same time, so a
// piece is in danger on any square another player could move to now, and a
// piece that moves away dodges. Played in turns, the danger is whatever the
// others could take after the bot's move. For every move it adds up what it
// takes (less likely to work against a piece its owner can see coming), what
// it puts or leaves in danger (less when another piece covers it), the safety
// of its king above all, and a little for heading to the middle and pushing
// pawns towards promotion.
func BotMoveSeeing(p *Position, seen *[NumSquares]bool, strength Strength, rng *rand.Rand) (Move, bool) {
	return botMoveWith(p, seen, strength, nil, rng)
}

// botMoveWith is BotMoveSeeing with weights w in place of the level's, for
// tests.
func botMoveWith(p *Position, seen *[NumSquares]bool, strength Strength, w *weights, rng *rand.Rand) (Move, bool) {
	if strength <= Greedy {
		return greedyMove(p, seen, rng)
	}
	moves := p.LegalMoves()
	if len(moves) == 0 {
		return Move{}, false
	}
	us := p.Turn
	known := *p
	if seen != nil {
		for sq, pc := range known.Board {
			if !pc.IsEmpty() && pc.Color() != us && !seen[sq] {
				known.Board[sq] = NoPiece
			}
		}
	}
	b := bot{us: us, known: &known, w: careful, seen: seen}
	if strength >= Strongest {
		b.w = strongest
	}
	if w != nil {
		b.w = *w
	}
	if known.Sync {
		b.reach = b.reaches(&known)
	}
	scores := make([]float64, len(moves))
	for i, m := range moves {
		scores[i] = b.score(m) + rng.Float64()*b.w.noise
	}
	best := 0
	for i := range moves {
		if scores[i] > scores[best] {
			best = i
		}
	}
	return moves[best], true
}

// weights are what a bot cares about, in pawns.
type weights struct {
	// hit is how likely another player is to take a piece of ours they can
	// reach, and dodge how likely a piece we attack is moved away.
	hit, dodge float64
	// king is what the bot's own king is worth when it weighs danger, and
	// take what taking another king is worth to it.
	king, take float64
	// threat is what lining up a capture for next round is worth, as a share
	// of the piece's value; centre what heading for the middle is worth, and
	// pawn each step a pawn takes forward.
	threat, centre, pawn float64
	// fog is what each way into the king from squares the bot can't see
	// costs, and outpost what holding a vision square (the whole board under
	// centre vision) is worth.
	fog, outpost float64
	// noise is how much chance goes into each move's score.
	noise float64
}

// strongest was tuned in bot matches (see TestFourLadder): changing any one
// weight by half or double made no clear difference against itself. Lining
// up next round's captures (threat) made no difference against itself
// either, but against the Greedy bot it won 85% of games instead of 62%, and
// it ends games: a third as many were still going after 400 rounds. careful
// is the same bot without it and with more chance in what it plays. Under
// fog both guard the king and hold the vision squares (see TestFourFog).
var (
	strongest = weights{hit: 0.4, dodge: 0.45, king: 60, take: 40, centre: 1.0 / 60, pawn: 0.1, threat: 0.3, fog: 0.5, outpost: 1.5, noise: 0.05}
	careful   = weights{hit: 0.4, dodge: 0.45, king: 60, take: 40, centre: 1.0 / 60, pawn: 0.1, fog: 0.5, outpost: 1.5, noise: 4}
)

type bot struct {
	us    Color
	known *Position
	w     weights
	// seen is what the bot sees under fog, nil without it.
	seen *[NumSquares]bool
	// reach counts, for each square, how many other players could move a
	// piece there this round, and cheapest is the least valuable such piece.
	reach *reachMap
}

type reachMap struct {
	count    [NumSquares]uint8
	cheapest [NumSquares]float64
	players  [NumSquares]uint8 // a bit per player
}

// reaches maps where the other players still in could move to in p.
func (b *bot) reaches(p *Position) *reachMap {
	r := &reachMap{}
	for sq, pc := range p.Board {
		if pc.IsEmpty() || pc.Color() == b.us || p.Out[pc.Color()] {
			continue
		}
		v := pieceValue[pc.Kind()]
		if pc.Kind() == King {
			v = 50
		}
		for _, m := range p.PieceMoves(Square(sq)) {
			bit := uint8(1) << pc.Color()
			if r.players[m.To]&bit == 0 {
				r.players[m.To] |= bit
				r.count[m.To]++
			}
			if r.cheapest[m.To] == 0 || v < r.cheapest[m.To] {
				r.cheapest[m.To] = v
			}
		}
	}
	return r
}

func (b *bot) worth(k Kind) float64 {
	if k == King {
		return b.w.king
	}
	return pieceValue[k]
}

func (b *bot) score(m Move) float64 {
	p := b.known
	pc := p.Board[m.From]
	score := 0.0

	// What it takes.
	target := p.Board[m.To]
	if m.Is(FlagEnPassant) {
		target = p.Board[p.EPPawn]
	}
	if !target.IsEmpty() {
		gain := pieceValue[target.Kind()]
		if target.Kind() == King {
			gain = b.w.take
		}
		if p.Sync {
			// The owner sees it coming and may move it away.
			gain *= 1 - b.w.dodge
		}
		score += gain
	}
	if m.Promo != NoKind {
		score += pieceValue[m.Promo] - 1
	}
	if p.HillOn() && pc.Kind() == King && p.Shape.InCentre(m.To) {
		score += 1000
	}

	after, _ := p.Apply(m)
	after.Turn = b.us
	reach := b.reach
	if !p.Sync {
		reach = b.reaches(&after)
	}
	cover := b.cover(&after)

	// What it leaves in danger: every piece of ours on a square someone else
	// could reach. In Sync Move landing on the same square as someone else
	// knocks both out, so the moved piece counts even on an empty square.
	for sq, own := range after.Board {
		if !own.Is(b.us) {
			continue
		}
		n := reach.count[sq]
		if n == 0 {
			continue
		}
		chance := 1.0
		for range n {
			chance *= 1 - b.w.hit
		}
		chance = 1 - chance
		loss := b.worth(own.Kind())
		if cover[sq] && own.Kind() != King {
			// We'd take back: lose the difference, if any.
			loss = max(loss-reach.cheapest[sq], loss*0.15)
		}
		score -= loss * chance
	}

	// What the moved piece threatens next: worth a little, a king most.
	if b.w.threat > 0 {
		for _, t := range after.PieceMoves(m.To) {
			if v := after.Board[t.To]; t.Is(FlagCapture) && !v.IsEmpty() {
				if v.Kind() == King {
					score += b.w.threat * 10
				} else {
					score += b.w.threat * pieceValue[v.Kind()]
				}
			}
		}
	}

	// Under fog: the ways into the king from what it can't see, and the
	// vision squares that show the whole board.
	if b.seen != nil {
		k := after.KingSquare(b.us)
		if k != NoSquare {
			score -= b.w.fog * float64(b.exposure(&after, k))
		}
		if p.CenterVision != CenterOff && (p.CenterVision == CenterAny || pc.Kind() == Pawn) {
			for _, o := range p.Shape.Outposts(b.us) {
				if m.To == o {
					score += b.w.outpost
				}
				if m.From == o {
					score -= b.w.outpost
				}
			}
		}
	}

	// Heading for the middle, and pawns forward.
	if pc.Kind() != King {
		w := float64(p.Shape.Width())/2 - 0.5
		dx, dy := float64(m.To.X())-w, float64(m.To.Y())-w
		fx, fy := float64(m.From.X())-w, float64(m.From.Y())-w
		score += (fx*fx + fy*fy - dx*dx - dy*dy) * b.w.centre
	}
	if pc.Kind() == Pawn {
		score += b.w.pawn * float64(p.Shape.RankOf(b.us, m.To)-p.Shape.RankOf(b.us, m.From))
	}
	return score
}

// cover marks the squares our pieces could take back on in p.
func (b *bot) cover(p *Position) *[NumSquares]bool {
	var c [NumSquares]bool
	q := *p
	// Make every square of ours look like someone else's so our moves onto
	// them show up as captures.
	other := NewPiece((b.us+1)%4, Pawn)
	for sq, pc := range q.Board {
		if pc.Is(b.us) {
			q.Board[sq] = other
		}
	}
	for sq, pc := range p.Board {
		if !pc.Is(b.us) {
			continue
		}
		q.Board[sq] = pc
		for _, m := range q.PieceMoves(Square(sq)) {
			if p.Board[m.To].Is(b.us) {
				c[m.To] = true
			}
		}
		q.Board[sq] = other
	}
	return &c
}

// exposure counts the ways into the king on k from squares the bot can't
// see: each line from it that reaches the fog before a piece, and each
// knight's jump away that's in the fog. Anything could be waiting there.
func (b *bot) exposure(p *Position, k Square) int {
	n := 0
	for _, d := range kingDeltas {
		for sq, ok := p.Shape.Offset(k, d.dx, d.dy); ok; sq, ok = p.Shape.Offset(sq, d.dx, d.dy) {
			if !b.seen[sq] {
				n++
				break
			}
			if !p.Board[sq].IsEmpty() {
				break
			}
		}
	}
	for _, d := range knightDeltas {
		if sq, ok := p.Shape.Offset(k, d.dx, d.dy); ok && !b.seen[sq] {
			n++
		}
	}
	return n
}
