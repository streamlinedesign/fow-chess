// Package bots picks moves for the computer opponents. A bot plays from its
// own view of the board: under fog it knows only the squares its pieces can
// see, exactly like a human in the same seat, so it never cheats.
package bots

import (
	"math"
	"math/rand/v2"
	"sort"

	"chess/engine"
)

// Level is a bot's strength.
type Level string

const (
	// Rex plays a random legal move. Rex inutilis: the useless king.
	Rex Level = "rex"
	// Vercingetorix grabs what it can see and avoids the obvious.
	Vercingetorix Level = "vercingetorix"
	// Ramesses looks a few moves ahead.
	Ramesses Level = "ramesses"
	// Solomon looks further and values what it can see.
	Solomon Level = "solomon"
)

// Levels, weakest first.
var Levels = []Level{Rex, Vercingetorix, Ramesses, Solomon}

// Name is the name a bot plays under.
func (l Level) Name() string {
	switch l {
	case Rex:
		return "Rex inutilis"
	case Vercingetorix:
		return "Vercingetorix"
	case Ramesses:
		return "Ramesses II"
	case Solomon:
		return "Solomon"
	}
	return ""
}

// Valid reports whether l names a bot.
func (l Level) Valid() bool { return l.Name() != "" }

// depth is how many plies each level searches.
func (l Level) depth() int {
	switch l {
	case Vercingetorix:
		return 2
	case Ramesses:
		return 3
	case Solomon:
		return 4
	}
	return 0
}

// Choose picks a move for the side to move in pos, who sees the squares in
// seen (nil for the whole board). Candidates are pos's legal moves, so the
// move is always playable; what the bot thinks will follow is worked out on
// the board as it sees it, with the pieces it can't see left out. It returns
// false when there is no legal move.
func Choose(level Level, pos engine.Position, seen *[64]bool, rng *rand.Rand) (engine.Move, bool) {
	moves := pos.LegalMoves()
	if len(moves) == 0 {
		return engine.Move{}, false
	}
	if rng == nil {
		rng = rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))
	}
	if level.depth() == 0 {
		return moves[rng.IntN(len(moves))], true
	}
	known := knownBoard(pos, seen)
	s := &search{level: level, rng: rng}
	// A king the bot can't see isn't missing, it's hidden: its absence from
	// the known board must not look like a capture.
	for _, c := range []engine.Color{engine.White, engine.Black} {
		s.kingKnown[c] = known.KingSquare(c) != engine.NoSquare
	}
	best, bestScore := moves[0], math.Inf(-1)
	for _, m := range shuffled(moves, rng) {
		// Each move gets a full window, so its score is exact and the noise
		// below compares like with like. A little noise keeps the bot from
		// playing the same game twice.
		s.nodes = 0
		score := -s.negamax(known.Apply(m), level.depth()-1, math.Inf(-1), math.Inf(1)) + rng.Float64()*0.05
		if score > bestScore {
			best, bestScore = m, score
		}
	}
	return best, true
}

// knownBoard is pos as the side to move sees it: pieces on squares it can't
// see are left out, and the search plays without check, since the bot can't
// tell what a hidden piece attacks anyway.
func knownBoard(pos engine.Position, seen *[64]bool) engine.Position {
	known := pos
	known.KingCapture = true
	if seen == nil {
		return known
	}
	for sq := range known.Board {
		if !seen[sq] && known.Board[sq].Is(pos.Turn.Other()) {
			known.Board[sq] = engine.NoPiece
		}
	}
	return known
}

type search struct {
	level     Level
	rng       *rand.Rand
	nodes     int
	kingKnown [2]bool
}

// maxNodes caps the search below each candidate move, so a bot never takes
// long whatever the position.
const maxNodes = 20000

// win is the score for taking the king.
const win = 100000

// negamax scores pos for the side to move, looking depth plies ahead.
func (s *search) negamax(pos engine.Position, depth int, alpha, beta float64) float64 {
	s.nodes++
	us := pos.Turn
	// The side to move has no king on the board it knows: it lost it.
	if s.kingKnown[us] && pos.KingSquare(us) == engine.NoSquare {
		return -win - float64(depth)
	}
	if k := pos.KingSquare(us.Other()); pos.Hill && k != engine.NoSquare && engine.OnHill(k) {
		return -win - float64(depth)
	}
	if depth <= 0 || s.nodes > maxNodes {
		return s.evaluate(pos)
	}
	moves := pos.LegalMoves()
	if len(moves) == 0 {
		// Nothing of this side's in sight can move: the hidden pieces will, so
		// score the board as it stands rather than call it a stalemate.
		return s.evaluate(pos)
	}
	orderMoves(pos, moves)
	best := math.Inf(-1)
	for _, m := range moves {
		score := -s.negamax(pos.Apply(m), depth-1, -beta, -alpha)
		if score > best {
			best = score
		}
		if score > alpha {
			alpha = score
		}
		if alpha >= beta {
			break
		}
	}
	return best
}

var value = [...]float64{engine.NoKind: 0, engine.Pawn: 1, engine.Knight: 3, engine.Bishop: 3, engine.Rook: 5, engine.Queen: 9, engine.King: 0}

// evaluate scores pos for the side to move: material it can see, and for the
// stronger bots, how much of the board it sees and pieces near the centre.
func (s *search) evaluate(pos engine.Position) float64 {
	us := pos.Turn
	score := 0.0
	for sq, pc := range pos.Board {
		if pc.IsEmpty() {
			continue
		}
		v := value[pc.Kind()]
		if s.level != Vercingetorix && pc.Kind() != engine.King {
			v += centre(engine.Square(sq)) * 0.03
		}
		if pc.Is(us) {
			score += v
		} else {
			score -= v
		}
	}
	if s.level == Solomon {
		// Seeing more of the board is worth a little: fog hides the danger.
		seen := pos.Vision(us)
		n := 0
		for _, ok := range seen {
			if ok {
				n++
			}
		}
		score += float64(n) * 0.02
	}
	return score
}

// centre is 1 on the four centre squares, falling to 0 at the edge.
func centre(sq engine.Square) float64 {
	f, r := float64(sq.File())-3.5, float64(sq.Rank())-3.5
	return 1 - (math.Abs(f)+math.Abs(r)-1)/6
}

// orderMoves puts captures of the most valuable pieces first, which makes the
// pruning bite.
func orderMoves(pos engine.Position, moves []engine.Move) {
	sort.SliceStable(moves, func(i, j int) bool {
		return captureValue(pos, moves[i]) > captureValue(pos, moves[j])
	})
}

func captureValue(pos engine.Position, m engine.Move) float64 {
	pc := pos.Board[m.To]
	if pc.IsEmpty() {
		return 0
	}
	if pc.Kind() == engine.King {
		return win
	}
	return value[pc.Kind()]
}

func shuffled(moves []engine.Move, rng *rand.Rand) []engine.Move {
	out := append([]engine.Move(nil), moves...)
	rng.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
	return out
}
