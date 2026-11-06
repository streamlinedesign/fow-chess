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
	// Ramesses looks a few moves ahead and guards its king against the fog.
	Ramesses Level = "ramesses"
	// Solomon imagines where the pieces it can't see might be.
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

// Choose picks a move for the side to move in pos, who sees the squares in
// seen (nil for the whole board). Candidates are pos's legal moves, so the
// move is always playable; what the bot thinks will follow is worked out on
// the board as it sees it, with the pieces it can't see left out. It returns
// false when there is no legal move.
func Choose(level Level, pos engine.Position, seen *[64]bool, rng *rand.Rand) (engine.Move, bool) {
	return ChooseWarned(level, pos, seen, false, rng)
}

// ChooseWarned is Choose in a game with the king warning, where a player is
// told when their king is attacked, even by a piece they can't see. The bot
// works that out from pos, as the server does for a person, and uses only
// that one fact.
func ChooseWarned(level Level, pos engine.Position, seen *[64]bool, warning bool, rng *rand.Rand) (engine.Move, bool) {
	return choose(level, pos, seen, warning, false, rng)
}

// ChooseBusy is ChooseWarned for a server with other bots waiting to think:
// Solomon imagines a quarter of the boards and every searching bot looks at a
// quarter of the positions within 150 ms, so a crowd of bots doesn't hold up everyone else.
// It plays weaker, but answers in a fraction of the time.
func ChooseBusy(level Level, pos engine.Position, seen *[64]bool, warning bool, rng *rand.Rand) (engine.Move, bool) {
	return choose(level, pos, seen, warning, true, rng)
}

func choose(level Level, pos engine.Position, seen *[64]bool, warning, busy bool, rng *rand.Rand) (engine.Move, bool) {
	if rng == nil {
		rng = rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))
	}
	if pl, ok := plans[level]; ok {
		if busy {
			pl = pl.busy()
		}
		return chooseDeep(pl, pos, seen, warning, rng)
	}
	moves := pos.LegalMoves()
	if len(moves) == 0 {
		return engine.Move{}, false
	}
	return moves[rng.IntN(len(moves))], true
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

// win is the score for taking the king.
const win = 100000

var value = [...]float64{engine.NoKind: 0, engine.Pawn: 1, engine.Knight: 3, engine.Bishop: 3, engine.Rook: 5, engine.Queen: 9, engine.King: 0}

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
