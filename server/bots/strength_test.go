package bots

import (
	"cmp"
	"math"
	"math/rand/v2"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"chess/engine"
)

// A player in a strength match: picks a move for the side to move in pos, who
// sees seen (nil without fog).
type player func(pos engine.Position, seen *[64]bool, r *rand.Rand) (engine.Move, bool)

// kingWarning plays the matches with the king warning, as Eat the King does.
var kingWarning = os.Getenv("STRENGTH_WARNING") != "off"

// botPlayer is a bot as it plays now, or as it played before 2026-11 with
// the prefix "old:" (legacy_test.go).
func botPlayer(name string) player {
	if l, ok := strings.CutPrefix(name, "old:"); ok {
		return func(pos engine.Position, seen *[64]bool, r *rand.Rand) (engine.Move, bool) {
			return legacyChoose(Level(l), pos, seen, r, Level(l) == Ramesses || Level(l) == Solomon)
		}
	}
	return func(pos engine.Position, seen *[64]bool, r *rand.Rand) (engine.Move, bool) {
		return ChooseWarned(Level(name), pos, seen, kingWarning, r)
	}
}

// playGame plays one game of Eat the King as the site sets it up (fog, king
// capture, centre vision) unless fog is false, and returns White's score.
func playGame(white, black player, fog bool, seed uint64, think *[2]time.Duration, moves *[2]int) float64 {
	r := rand.New(rand.NewPCG(seed, seed*7+1))
	p, _ := engine.ParseFEN(engine.StartFEN)
	p.KingCapture = true
	if fog {
		p.CenterVision = engine.CenterAny
	}
	// A couple of random opening moves so the games differ.
	for range 2 {
		ms := p.LegalMoves()
		p = p.Apply(ms[r.IntN(len(ms))])
	}
	for ply := 0; ply < 300; ply++ {
		me := p.Turn
		var seen *[64]bool
		if fog {
			v := p.Vision(me)
			seen = &v
		}
		pl := white
		if me == engine.Black {
			pl = black
		}
		start := time.Now()
		m, ok := pl(p, seen, r)
		think[me] += time.Since(start)
		moves[me]++
		if !ok {
			return 0.5
		}
		p = p.Apply(m)
		if p.KingSquare(me.Other()) == engine.NoSquare {
			if me == engine.White {
				return 1
			}
			return 0
		}
	}
	return 0.5
}

// match plays n games between a and b, each taking White in half of them, and
// returns a's score and its average time per move.
func match(n int, a, b player, fog bool) (score float64, wins, draws, losses int, aMove, bMove time.Duration) {
	var mu sync.Mutex
	var wg sync.WaitGroup
	var aThink, bThink time.Duration
	var aMoves, bMoves int
	sem := make(chan struct{}, runtime.GOMAXPROCS(0))
	for i := range n {
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			var think [2]time.Duration
			var moves [2]int
			seed := uint64(i / 2) // each opening is played from both sides
			var s float64
			ai := engine.White
			if i%2 == 0 {
				s = playGame(a, b, fog, seed, &think, &moves)
			} else {
				s = 1 - playGame(b, a, fog, seed, &think, &moves)
				ai = engine.Black
			}
			mu.Lock()
			defer mu.Unlock()
			score += s
			switch s {
			case 1:
				wins++
			case 0:
				losses++
			default:
				draws++
			}
			aThink += think[ai]
			aMoves += moves[ai]
			bThink += think[ai.Other()]
			bMoves += moves[ai.Other()]
		}()
	}
	wg.Wait()
	return score / float64(n), wins, draws, losses, aThink / time.Duration(max(aMoves, 1)), bThink / time.Duration(max(bMoves, 1))
}

// elo is the rating difference a score s implies.
func elo(s float64) float64 {
	s = min(max(s, 0.001), 0.999)
	return -400 * math.Log10(1/s-1)
}

// TestStrength plays bot A against bot B, Solomon now against Solomon
// before unless STRENGTH_A and STRENGTH_B say otherwise (a level, or "old:"
// and a level for how it played before 2026-11). It takes a while, so it runs
// only when asked:
//
//	STRENGTH_GAMES=400 go test ./bots -run Strength -v
//	STRENGTH_GAMES=400 STRENGTH_A=ramesses STRENGTH_B=old:ramesses go test ./bots -run Strength -v
//
// STRENGTH_FOG=off plays without fog of war, STRENGTH_WARNING=off without the
// king warning. PLAN_DEPTH, PLAN_SAMPLES and
// PLAN_NODES change A's plan, to try one out.
func TestStrength(t *testing.T) {
	n, _ := strconv.Atoi(os.Getenv("STRENGTH_GAMES"))
	if n == 0 {
		t.Skip("set STRENGTH_GAMES to the number of games")
	}
	a, b := cmp.Or(os.Getenv("STRENGTH_A"), "solomon"), cmp.Or(os.Getenv("STRENGTH_B"), "old:solomon")
	fog := os.Getenv("STRENGTH_FOG") != "off"
	if pl, ok := plans[Level(a)]; ok {
		for env, field := range map[string]*int{"PLAN_DEPTH": &pl.fogDepth, "PLAN_SAMPLES": &pl.samples, "PLAN_NODES": &pl.nodes} {
			if v, err := strconv.Atoi(os.Getenv(env)); err == nil {
				*field = v
			}
		}
		plans[Level(a)] = pl
	}
	s, w, d, l, ta, tb := match(n, botPlayer(a), botPlayer(b), fog)
	// Standard error of the mean score, for a rough 95% interval.
	se := math.Sqrt((float64(w)*(1-s)*(1-s)+float64(d)*(0.5-s)*(0.5-s)+float64(l)*s*s)/float64(n)) / math.Sqrt(float64(n))
	t.Logf("fog %v: %s vs %s: +%d =%d -%d, score %.3f (±%.3f), about %+.0f Elo (%+.0f to %+.0f); %v vs %v a move",
		fog, a, b, w, d, l, s, 2*se, elo(s), elo(s-2*se), elo(s+2*se), ta.Round(time.Millisecond), tb.Round(time.Millisecond))
}
