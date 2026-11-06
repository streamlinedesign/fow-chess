package bots

import (
	"math"
	"math/rand/v2"
	"time"

	"chess/engine"
)

// The searching bots, Vercingetorix, Ramesses II and Solomon, share one
// search: iterative deepening with alpha-beta, a transposition table,
// captures played out to the end (quiescence), and killer and history move
// ordering. They play from the board they can see. What sets them apart is
// their plan (see plans): how far they look while part of the board is
// hidden, and whether they imagine the pieces they can't see.
//
// Two things the strength matches (strength_test.go) taught them. Under fog a
// deep search hurts: the board it searches is missing every piece in the fog,
// and the deeper it looks the more it trusts that board, so depth 3 beat
// depth 5 clearly. And imagining the hidden pieces helps a lot: fill the fog
// with them at random, many times over, look two plies deep on each board,
// and play the move that does best on average.

// plan is how a bot searches.
type plan struct {
	// fogDepth is how many plies it looks ahead while part of the board is
	// hidden; with the whole board in sight it looks as deep as nodes allow.
	fogDepth int
	// samples is how many boards with the hidden pieces imagined it searches,
	// fogDepth deep each; 0 searches the board as seen.
	samples int
	// nodes is how many positions it may look at for a move.
	nodes int
	// guarded scores the ways into its king from the fog (see exposure).
	guarded bool
	// think, when set, is a shorter limit than ThinkTime.
	think time.Duration
}

// In 400-game matches under Eat the King's rules each level beat its old self
// (Vercingetorix by about 215 Elo, Ramesses II 110, Solomon 265), the new
// Vercingetorix beat the old Ramesses II, Ramesses II beat Vercingetorix by
// about 110 and Solomon beat Ramesses II by about 70.
var plans = map[Level]plan{
	Vercingetorix: {fogDepth: 1, nodes: 5000},
	Ramesses:      {fogDepth: 3, nodes: 40000, guarded: true},
	Solomon:       {fogDepth: 2, samples: 32, nodes: 300000, guarded: true},
}

// thinkTime is the most time a search with plan pl may take.
func thinkTime(pl plan) time.Duration {
	if ThinkTime > 0 && pl.think > 0 {
		return min(ThinkTime, pl.think)
	}
	return ThinkTime
}

// busy is the plan with a quarter of the boards and positions, and at most
// 150 ms to think, for a server
// with other bots waiting to think (see ChooseBusy).
func (pl plan) busy() plan {
	if pl.samples > 0 {
		pl.samples = max(pl.samples/4, 4)
	}
	pl.nodes /= 4
	pl.think = 150 * time.Millisecond
	return pl
}

// ThinkTime is the most time any bot takes for a move, whatever the server
// is doing. 0 is no limit, so a seed always plays the same game: the node
// budget alone stops the search.
var ThinkTime = 600 * time.Millisecond

// fogCost is what each way into the king from the fog costs, in pawns, and
// clampScore the most one imagined board can count for or against a move,
// so one imagined king capture doesn't outweigh everything else.
const (
	fogCost    = 0.6
	clampScore = 20.0
	// warnedCost is what leaving the king where the warning says it's
	// attacked costs a bot that doesn't imagine the hidden pieces.
	warnedCost = 3.0
)

const (
	maxPly  = 64
	ttSize  = 1 << 16
	mateish = win - 1000 // scores beyond this are a king taken
)

type ttFlag uint8

const (
	ttExact ttFlag = iota
	ttLower
	ttUpper
)

type ttEntry struct {
	key   uint64
	move  engine.Move
	score float64
	depth int8
	flag  ttFlag
}

type deep struct {
	plan      plan
	me        engine.Color
	kingKnown [2]bool
	guarded   bool
	// The fog as it was when the search began: me's view, which doesn't change
	// with the moves the search tries (see evaluate).
	nodes    int
	budget   int
	deadline time.Time
	stopped  bool
	tt       []ttEntry
	// hidden counts the opponent's pieces the bot can't see, by kind: what a
	// player knows from the pieces they've taken and the ones in sight.
	hidden [7]int
	// warned is 1 when the king warning says the bot's king, on kingFrom, is
	// attacked, -1 when it says it isn't, and 0 without the warning.
	warned   int8
	kingFrom engine.Square
	killers  [maxPly][2]engine.Move
	history  [2][64][64]int32
}

var zobrist = func() (z struct {
	piece    [16][64]uint64
	turn     uint64
	castling [16]uint64
	ep       [64]uint64
}) {
	r := rand.New(rand.NewPCG(0x50101, 0x404))
	for i := range z.piece {
		for j := range z.piece[i] {
			z.piece[i][j] = r.Uint64()
		}
	}
	z.turn = r.Uint64()
	for i := range z.castling {
		z.castling[i] = r.Uint64()
	}
	for i := range z.ep {
		z.ep[i] = r.Uint64()
	}
	return
}()

func hash(p *engine.Position) uint64 {
	var h uint64
	for sq, pc := range p.Board {
		if pc != engine.NoPiece {
			h ^= zobrist.piece[pc][sq]
		}
	}
	if p.Turn == engine.Black {
		h ^= zobrist.turn
	}
	h ^= zobrist.castling[p.Castling&15]
	if p.EP != engine.NoSquare {
		h ^= zobrist.ep[p.EP]
	}
	return h
}

// chooseDeep is Choose for the searching bots.
func chooseDeep(pl plan, pos engine.Position, seen *[64]bool, warning bool, rng *rand.Rand) (engine.Move, bool) {
	moves := pos.LegalMoves()
	if len(moves) == 0 {
		return engine.Move{}, false
	}
	known := knownBoard(pos, seen)
	s := &deep{
		plan:     pl,
		me:       pos.Turn,
		guarded:  pl.guarded && seen != nil,
		budget:   pl.nodes,
		deadline: time.Now().Add(thinkTime(pl)),
		tt:       make([]ttEntry, ttSize),
	}
	for _, c := range []engine.Color{engine.White, engine.Black} {
		s.kingKnown[c] = known.KingSquare(c) != engine.NoSquare
	}
	for sq, pc := range pos.Board {
		if pc.Is(pos.Turn.Other()) && known.Board[sq].IsEmpty() {
			s.hidden[pc.Kind()]++
		}
	}
	if k := pos.KingSquare(s.me); warning && seen != nil && k != engine.NoSquare {
		s.kingFrom = k
		s.warned = -1
		if pos.Attacked(k, s.me.Other()) {
			s.warned = 1
		}
	}
	// A little noise on each candidate keeps the bot from playing the same
	// game twice; it only ever decides between moves that score the same.
	moves = shuffled(moves, rng)
	noise := make([]float64, len(moves))
	for i := range noise {
		noise[i] = rng.Float64() * 0.05
	}
	fogged := seen != nil && !allSeen(seen)
	if fogged && pl.samples > 0 {
		if m, ok := s.sampled(known, seen, moves, noise, rng); ok {
			return m, true
		}
	}
	best := 0
	limit := maxPly - 1
	if fogged {
		limit = pl.fogDepth
	}
	for depth := 1; depth <= limit; depth++ {
		alpha, beta := math.Inf(-1), math.Inf(1)
		iterBest := -1
		for i, m := range moves {
			child := known.Apply(m)
			var score float64
			if i == 0 {
				score = -s.negamax(child, depth-1, 1, -beta+noise[i], -alpha+noise[i]) + noise[i]
			} else {
				// Prove it's worse than the best with a null window first.
				score = -s.negamax(child, depth-1, 1, -alpha+noise[i]-0.001, -alpha+noise[i]) + noise[i]
				if score > alpha && !s.stopped {
					score = -s.negamax(child, depth-1, 1, -beta+noise[i], -alpha+noise[i]) + noise[i]
				}
			}
			if s.stopped {
				break
			}
			if score > alpha {
				alpha, iterBest = score, i
			}
		}
		if iterBest >= 0 {
			best = iterBest
			// The best move goes first next time round.
			moves[0], moves[best] = moves[best], moves[0]
			noise[0], noise[best] = noise[best], noise[0]
			best = 0
			if alpha > mateish || alpha < -mateish {
				break // a king falls whatever happens
			}
		}
		if s.stopped {
			break
		}
	}
	return moves[best], true
}

func (s *deep) tick() bool {
	s.nodes++
	if s.nodes >= s.budget || s.nodes&1023 == 0 && ThinkTime > 0 && time.Now().After(s.deadline) {
		s.stopped = true
	}
	return s.stopped
}

// lost reports a position that's over for the side to move, scored for it.
func (s *deep) lost(pos *engine.Position, ply int) (float64, bool) {
	us := pos.Turn
	if s.kingKnown[us] && pos.KingSquare(us) == engine.NoSquare {
		return -win + float64(ply), true
	}
	if k := pos.KingSquare(us.Other()); pos.Hill && k != engine.NoSquare && engine.OnHill(k) {
		return -win + float64(ply), true
	}
	return 0, false
}

func (s *deep) negamax(pos engine.Position, depth, ply int, alpha, beta float64) float64 {
	if score, over := s.lost(&pos, ply); over {
		return score
	}
	if depth <= 0 || ply >= maxPly-1 {
		return s.quiesce(pos, ply, alpha, beta, 0)
	}
	if s.tick() {
		return 0
	}
	key := hash(&pos)
	e := &s.tt[key&(ttSize-1)]
	var ttMove engine.Move
	if e.key == key {
		ttMove = e.move
		if int(e.depth) >= depth {
			sc := fromTT(e.score, ply)
			switch {
			case e.flag == ttExact,
				e.flag == ttLower && sc >= beta,
				e.flag == ttUpper && sc <= alpha:
				return sc
			}
		}
	}
	moves := pos.LegalMoves()
	if len(moves) == 0 {
		if pos.Turn == s.me {
			return s.evaluate(&pos)
		}
		// Nothing of the opponent's in sight can move: a hidden piece will,
		// somewhere the bot can't see. Let it pass, so the bot can still plan
		// its own next moves.
		pass := pos
		pass.Turn = s.me
		pass.EP = engine.NoSquare
		return -s.negamax(pass, depth-1, ply+1, -beta, -alpha)
	}
	s.order(&pos, moves, ttMove, ply)
	if pos.Board[moves[0].To].Kind() == engine.King {
		return win - float64(ply+1)
	}
	us := pos.Turn
	origAlpha := alpha
	best, bestMove := math.Inf(-1), moves[0]
	for i, m := range moves {
		child := pos.Apply(m)
		quiet := pos.Board[m.To].IsEmpty() && m.Promo == engine.NoKind
		var score float64
		if i == 0 {
			score = -s.negamax(child, depth-1, ply+1, -beta, -alpha)
		} else {
			r := 0
			// Late quiet moves get a shallower look first.
			if depth >= 3 && i >= 4 && quiet && m != s.killers[ply][0] && m != s.killers[ply][1] {
				r = 1
				if i >= 12 && depth >= 5 {
					r = 2
				}
			}
			score = -s.negamax(child, depth-1-r, ply+1, -alpha-0.001, -alpha)
			if score > alpha && !s.stopped {
				score = -s.negamax(child, depth-1, ply+1, -beta, -alpha)
			}
		}
		if s.stopped {
			return 0
		}
		if score > best {
			best, bestMove = score, m
		}
		if score > alpha {
			alpha = score
		}
		if alpha >= beta {
			if quiet {
				if s.killers[ply][0] != m {
					s.killers[ply][1] = s.killers[ply][0]
					s.killers[ply][0] = m
				}
				s.history[us][m.From][m.To] += int32(depth * depth)
			}
			break
		}
	}
	flag := ttExact
	switch {
	case best <= origAlpha:
		flag = ttUpper
	case best >= beta:
		flag = ttLower
	}
	*e = ttEntry{key: key, move: bestMove, score: toTT(best, ply), depth: int8(depth), flag: flag}
	return best
}

// quiesce plays out the captures so a search never stops in the middle of an
// exchange.
func (s *deep) quiesce(pos engine.Position, ply int, alpha, beta float64, qdepth int) float64 {
	if score, over := s.lost(&pos, ply); over {
		return score
	}
	if s.tick() {
		return 0
	}
	stand := s.evaluate(&pos)
	if stand >= beta || qdepth >= 8 || ply >= maxPly-1 {
		return stand
	}
	if stand > alpha {
		alpha = stand
	}
	moves := pos.LegalMoves()
	caps := moves[:0]
	for _, m := range moves {
		if !pos.Board[m.To].IsEmpty() || m.Promo == engine.Queen {
			caps = append(caps, m)
		}
	}
	s.order(&pos, caps, engine.Move{}, -1)
	for _, m := range caps {
		victim := pos.Board[m.To].Kind()
		if victim == engine.King {
			return win - float64(ply+1)
		}
		// A capture that can't lift the score to alpha even for nothing back.
		if stand+value[victim]+1 < alpha && m.Promo == engine.NoKind {
			continue
		}
		score := -s.quiesce(pos.Apply(m), ply+1, -beta, -alpha, qdepth+1)
		if s.stopped {
			return 0
		}
		if score > alpha {
			alpha = score
			if alpha >= beta {
				break
			}
		}
	}
	return alpha
}

func toTT(score float64, ply int) float64 {
	switch {
	case score > mateish:
		return score + float64(ply)
	case score < -mateish:
		return score - float64(ply)
	}
	return score
}

func fromTT(score float64, ply int) float64 {
	switch {
	case score > mateish:
		return score - float64(ply)
	case score < -mateish:
		return score + float64(ply)
	}
	return score
}

// order sorts moves best first: the table's move, then king captures and
// other captures, most valuable victim by least valuable attacker, then the
// killers and the moves that cut off before.
func (s *deep) order(pos *engine.Position, moves []engine.Move, ttMove engine.Move, ply int) {
	keys := make([]int32, len(moves))
	for i, m := range moves {
		var k int32
		switch victim := pos.Board[m.To]; {
		case m == ttMove:
			k = 1 << 30
		case victim.Kind() == engine.King:
			k = 1 << 29
		case !victim.IsEmpty():
			k = 1<<28 + int32(value[victim.Kind()]*100)*16 - int32(value[pos.Board[m.From].Kind()]*10)
		case m.Promo == engine.Queen:
			k = 1<<28 + 800
		case ply >= 0 && m == s.killers[ply][0]:
			k = 1<<27 + 1
		case ply >= 0 && m == s.killers[ply][1]:
			k = 1 << 27
		default:
			k = min(s.history[pos.Turn][m.From][m.To], 1<<26)
		}
		keys[i] = k
	}
	// Insertion sort: the lists are short.
	for i := 1; i < len(moves); i++ {
		for j := i; j > 0 && keys[j] > keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
			moves[j], moves[j-1] = moves[j-1], moves[j]
		}
	}
}

// evaluate scores pos for the side to move, in pawns.
func (s *deep) evaluate(pos *engine.Position) float64 {
	us := pos.Turn
	score := 0.0
	for sq, pc := range pos.Board {
		if pc.IsEmpty() {
			continue
		}
		v := value[pc.Kind()] + placement(pc, engine.Square(sq))
		if pc.Is(us) {
			score += v
		} else {
			score -= v
		}
	}
	// Seeing more of the board is worth a little: fog hides the danger.
	mine := sight(pos, s.me)
	n := 0
	for _, ok := range mine {
		if ok {
			n++
		}
	}
	mineScore := float64(n) * 0.02
	if s.guarded {
		mineScore -= fogCost * float64(exposure(pos, s.me, &mine))
		if s.warned == 1 && pos.KingSquare(s.me) == s.kingFrom {
			// Attacked from somewhere: staying put is the worst choice.
			mineScore -= warnedCost
		}
	}
	if us == s.me {
		score += mineScore
	} else {
		score -= mineScore
	}
	return score
}

// placement is what a piece's square adds to its value: pieces towards the
// centre, pawns as they advance.
func placement(pc engine.Piece, sq engine.Square) float64 {
	switch pc.Kind() {
	case engine.King:
		return 0
	case engine.Pawn:
		r := sq.Rank()
		if pc.Color() == engine.Black {
			r = 7 - r
		}
		return centre(sq)*0.03 + float64(max(r-2, 0))*0.05
	}
	return centre(sq) * 0.06
}

func allSeen(seen *[64]bool) bool {
	for _, ok := range seen {
		if !ok {
			return false
		}
	}
	return true
}

// sampled picks a move by imagining where the pieces the bot can't see might
// be: it fills the fog with them plan.samples times, searches each board, and
// plays the move that does best on average. It reports false if it ran out of
// budget before finishing one board.
func (s *deep) sampled(known engine.Position, seen *[64]bool, moves []engine.Move, noise []float64, rng *rand.Rand) (engine.Move, bool) {
	total := make([]float64, len(moves))
	done := 0
	for range s.plan.samples {
		board := s.imagine(known, seen, rng)
		// With the king warning, imagine only boards that agree with it.
		for try := 0; s.warned != 0 && try < 20; try++ {
			k := board.KingSquare(s.me)
			if board.Attacked(k, s.me.Other()) == (s.warned == 1) {
				break
			}
			board = s.imagine(known, seen, rng)
		}
		s.kingKnown = [2]bool{true, true}
		scores := make([]float64, len(moves))
		for i, m := range moves {
			sc := -s.negamax(board.Apply(m), s.plan.fogDepth-1, 1, math.Inf(-1), math.Inf(1))
			if s.stopped {
				break
			}
			scores[i] = max(-clampScore, min(clampScore, sc))
		}
		if s.stopped {
			break
		}
		for i := range total {
			total[i] += scores[i]
		}
		done++
	}
	if done == 0 {
		return engine.Move{}, false
	}
	best, bestScore := 0, math.Inf(-1)
	for i := range moves {
		if sc := total[i]/float64(done) + noise[i]; sc > bestScore {
			best, bestScore = i, sc
		}
	}
	return moves[best], true
}

// imagine puts the opponent's hidden pieces on squares the bot can't see:
// back on their starting square when that's still possible, otherwise
// somewhere in the fog, more likely the nearer it is to the opponent's side.
func (s *deep) imagine(known engine.Position, seen *[64]bool, rng *rand.Rand) engine.Position {
	them := s.me.Other()
	var free []int
	for sq := range 64 {
		if !seen[sq] && known.Board[sq].IsEmpty() {
			free = append(free, sq)
		}
	}
	start, _ := engine.ParseFEN(engine.StartFEN)
	for kind := engine.Pawn; kind <= engine.King; kind++ {
		for range s.hidden[kind] {
			if len(free) == 0 {
				return known
			}
			pc := engine.NewPiece(them, kind)
			// Its starting squares that are still free, half the time.
			var home []int
			for j, sq := range free {
				if start.Board[sq] == pc {
					home = append(home, j)
				}
			}
			pick := -1
			if len(home) > 0 && rng.IntN(2) == 0 {
				pick = home[rng.IntN(len(home))]
			} else {
				totalW := 0.0
				w := make([]float64, len(free))
				for j, sq := range free {
					r := sq >> 3
					if them == engine.White {
						r = 7 - r // distance from the opponent's back row
					}
					depthIn := float64(7 - r) // 7 on the opponent's back row
					if kind == engine.Pawn && (sq>>3 == 0 || sq>>3 == 7) {
						continue
					}
					w[j] = 1 + depthIn*depthIn
					totalW += w[j]
				}
				if totalW == 0 {
					continue
				}
				x := rng.Float64() * totalW
				for j := range w {
					if x -= w[j]; x <= 0 && w[j] > 0 {
						pick = j
						break
					}
				}
				if pick < 0 {
					pick = len(free) - 1
				}
			}
			known.Board[free[pick]] = pc
			free = append(free[:pick], free[pick+1:]...)
		}
	}
	return known
}
