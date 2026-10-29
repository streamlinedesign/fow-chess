package sim

import (
	"math/rand/v2"

	"chess/engine"
)

// ghostLife is how many of its own turns the hunter keeps believing an enemy
// piece is where it last saw it. A king's ghost never fades: it's the best
// guess of where to go.
const ghostLife = 6

// hunter looks two plies ahead on the board it believes in: what it sees,
// plus enemy pieces where it last saw them. It eats the king when it can,
// keeps its own king out of reach of every piece it knows of, and otherwise
// trades well, sees more of the board, and closes in on the enemy king.
type hunter struct {
	rng *rand.Rand
	// wary also guards against what the fog might hide: lines into its
	// king from squares it can't see.
	wary   bool
	ghosts [64]engine.Piece
	age    [64]int
	// lost counts the enemy pieces of each kind this bot has taken.
	lost [7]int
}

func (h *hunter) remember(v *View) {
	them := v.Me.Other()
	var seen [7]int
	for i, pc := range v.Board {
		if v.Seen[i] {
			h.ghosts[i], h.age[i] = engine.NoPiece, 0
			if pc.Is(them) {
				h.ghosts[i] = pc
				seen[pc.Kind()]++
			}
			continue
		}
		if h.ghosts[i] != engine.NoPiece {
			h.age[i]++
			if h.age[i] > ghostLife && h.ghosts[i].Kind() != engine.King {
				h.ghosts[i] = engine.NoPiece
			}
		}
	}
	// A piece seen somewhere new isn't also where it was seen before: drop the
	// oldest ghosts of any kind the enemy can't have that many of.
	for k := engine.Pawn; k <= engine.King; k++ {
		limit := armyCount[k] - h.lost[k]
		if k != engine.King && k != engine.Pawn {
			limit += armyCount[engine.Pawn] - h.lost[engine.Pawn] // promotions
		}
		n := seen[k]
		for i := range h.ghosts {
			if h.ghosts[i].Kind() == k && !v.Seen[i] {
				n++
			}
		}
		for n > limit {
			oldest := -1
			for i := range h.ghosts {
				if h.ghosts[i].Kind() == k && !v.Seen[i] && (oldest < 0 || h.age[i] > h.age[oldest]) {
					oldest = i
				}
			}
			if oldest < 0 {
				break
			}
			h.ghosts[oldest] = engine.NoPiece
			n--
		}
	}
}

var armyCount = [7]int{engine.Pawn: 8, engine.Knight: 2, engine.Bishop: 2, engine.Rook: 2, engine.Queen: 1, engine.King: 1}

const kingValue = 100000

// noise is added to every move's score (in hundredths of a pawn) so that
// close choices go either way and games don't all follow one line.
const noise = 60

func (h *hunter) Move(v *View) engine.Move {
	h.remember(v)
	pos := beliefPosition(v, &h.ghosts)
	target := h.target(v)
	best, bestScore := v.Legal[0], -1<<62
	for _, m := range v.Legal {
		victim := v.Board[m.To]
		if victim.Kind() == engine.King {
			return m
		}
		after := pos.Apply(m)
		score := h.worstReply(&after, v.Me) + h.outlook(&after, v.Me, target)*10 + h.rng.IntN(noise)
		if score > bestScore {
			best, bestScore = m, score
		}
	}
	if k := v.Board[best.To].Kind(); k != engine.NoKind {
		h.lost[k]++
	}
	return best
}

// worstReply is the material score (from me's side) after the opponent's best
// reply on the believed board.
func (h *hunter) worstReply(p *engine.Position, me engine.Color) int {
	var replies []engine.Move
	if p.KingCapture {
		replies = p.PseudoMoves()
	} else {
		replies = p.LegalMoves()
	}
	if len(replies) == 0 {
		return material(p, me)
	}
	worst := 1 << 62
	for _, r := range replies {
		if p.Board[r.To].Kind() == engine.King {
			return -kingValue
		}
		after := p.Apply(r)
		if s := material(&after, me); s < worst {
			worst = s
		}
	}
	return worst
}

// material is me's pieces minus the other side's, in hundredths of a pawn.
func material(p *engine.Position, me engine.Color) int {
	s := 0
	for _, pc := range p.Board {
		if pc.IsEmpty() || pc.Kind() == engine.King {
			continue
		}
		if pc.Color() == me {
			s += value[pc.Kind()] * 100
		} else {
			s -= value[pc.Kind()] * 100
		}
	}
	return s
}

// outlook rewards seeing more of the board and bringing pieces closer to
// where the enemy king is believed to be.
func (h *hunter) outlook(p *engine.Position, me engine.Color, target engine.Square) int {
	seen := p.Vision(me)
	s := 0
	for _, ok := range seen {
		if ok {
			s++
		}
	}
	if h.wary {
		s -= 15 * exposure(p, me, &seen)
	}
	if target == engine.NoSquare {
		return s
	}
	for i, pc := range p.Board {
		if pc.Is(me) && pc.Kind() != engine.King {
			s += 7 - distance(engine.Square(i), target)
		}
	}
	return s
}

// target is where the hunter thinks the enemy king is: seen or remembered,
// or else its usual square while the enemy might not have moved it.
func (h *hunter) target(v *View) engine.Square {
	for i, g := range h.ghosts {
		if g.Kind() == engine.King {
			return engine.Square(i)
		}
	}
	if v.Me == engine.White {
		return engine.SquareAt(4, 7)
	}
	return engine.SquareAt(4, 0)
}

func distance(a, b engine.Square) int {
	return max(abs(a.File()-b.File()), abs(a.Rank()-b.Rank()))
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// exposure counts the ways into me's king from squares me can't see: each
// straight or diagonal line from the king that reaches the fog before it
// meets a piece, and each knight's jump away that's in the fog. Anything
// could be waiting there.
func exposure(p *engine.Position, me engine.Color, seen *[64]bool) int {
	king := p.KingSquare(me)
	if king == engine.NoSquare {
		return 0
	}
	n := 0
	for _, d := range [8][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}, {1, 1}, {1, -1}, {-1, 1}, {-1, -1}} {
		for s, ok := king.Offset(d[0], d[1]); ok; s, ok = s.Offset(d[0], d[1]) {
			if !seen[s] {
				n++
				break
			}
			if !p.Board[s].IsEmpty() {
				break
			}
		}
	}
	for _, d := range [8][2]int{{1, 2}, {2, 1}, {2, -1}, {1, -2}, {-1, -2}, {-2, -1}, {-2, 1}, {-1, 2}} {
		if s, ok := king.Offset(d[0], d[1]); ok && !seen[s] {
			n++
		}
	}
	return n
}
