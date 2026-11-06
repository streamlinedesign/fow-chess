package bots

// The Solomon that shipped before 2026-11, kept so the strength match in
// strength_test.go has something fixed to measure against. Only the tests use it.

import (
	"math"
	"math/rand/v2"

	"chess/engine"
)

const legacyMaxNodes = 20000

// choose is Choose, with guarded saying whether the bot guards its king
// against the fog (see exposure).
func legacyChoose(level Level, pos engine.Position, seen *[64]bool, rng *rand.Rand, guarded bool) (engine.Move, bool) {
	moves := pos.LegalMoves()
	if len(moves) == 0 {
		return engine.Move{}, false
	}
	if rng == nil {
		rng = rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))
	}
	if legacyDepth(level) == 0 {
		return moves[rng.IntN(len(moves))], true
	}
	known := knownBoard(pos, seen)
	s := &legacySearch{level: level, rng: rng, me: pos.Turn, guarded: guarded && seen != nil}
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
		score := -s.negamax(known.Apply(m), legacyDepth(level)-1, math.Inf(-1), math.Inf(1)) + rng.Float64()*0.05
		if score > bestScore {
			best, bestScore = m, score
		}
	}
	return best, true
}

type legacySearch struct {
	level     Level
	rng       *rand.Rand
	nodes     int
	kingKnown [2]bool
	// me is the bot's colour, and guarded whether it scores the ways into its
	// king from the fog.
	me      engine.Color
	guarded bool
}

// negamax scores pos for the side to move, looking depth plies ahead.
func (s *legacySearch) negamax(pos engine.Position, depth int, alpha, beta float64) float64 {
	s.nodes++
	us := pos.Turn
	// The side to move has no king on the board it knows: it lost it.
	if s.kingKnown[us] && pos.KingSquare(us) == engine.NoSquare {
		return -win - float64(depth)
	}
	if k := pos.KingSquare(us.Other()); pos.Hill && k != engine.NoSquare && engine.OnHill(k) {
		return -win - float64(depth)
	}
	if depth <= 0 || s.nodes > legacyMaxNodes {
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

// evaluate scores pos for the side to move: material it can see, and for the
// stronger bots, how much of the board it sees and pieces near the centre.
func (s *legacySearch) evaluate(pos engine.Position) float64 {
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
	if s.guarded {
		seen := pos.Vision(s.me)
		cost := exposureCost * float64(exposure(&pos, s.me, &seen))
		if us == s.me {
			score -= cost
		} else {
			score += cost
		}
	}
	return score
}

func legacyDepth(l Level) int {
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
