package four

import (
	"math/rand/v2"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

type fourPlayer func(p *Position, rng *rand.Rand) (Move, bool)

// fogged plays f from what the player to move sees.
func fogged(s Strength, w *weights) fourPlayer {
	return func(p *Position, r *rand.Rand) (Move, bool) {
		v := p.Vision(p.Turn)
		return botMoveWith(p, &v, s, w, r)
	}
}

// playFour plays one War Mode game (extended board, Battleship setups, Sync
// Move, jester pawns, no fog) with players[c] in each seat, and returns the
// winner, or false for a draw.
func playFour(players [4]fourPlayer, setup func(Shape, *rand.Rand) string, rules Rules, seed uint64) (Color, bool, [4]bool) {
	rng := rand.New(rand.NewPCG(seed, 99))
	var setups [4]string
	for c := range setups {
		setups[c] = setup(rules.Shape, rng)
	}
	g, err := NewGameFromSetups(rules, setups)
	if err != nil {
		panic(err)
	}
	for round := 0; round < 400 && !g.Over(); round++ {
		pos := g.Position()
		if !pos.Sync {
			m, ok := players[pos.Turn](&pos, rng)
			if !ok {
				break
			}
			if _, err := g.PlayUCI(m.UCI()); err != nil {
				panic(err)
			}
			continue
		}
		var moves [4]string
		for _, c := range Colors {
			if pos.Out[c] {
				continue
			}
			q := pos
			q.Turn = c
			if m, ok := players[c](&q, rng); ok {
				moves[c] = m.UCI()
			}
		}
		if _, err := g.PlayRound(moves); err != nil {
			panic(err)
		}
	}
	w, ok := g.Winner()
	return w, ok, g.Position().Out
}

// fourMatch seats a in one seat and b in the other three, a's seat going
// round, and returns a's share of the wins (0.25 is level) and of the draws.
func fourMatch(n int, a, b fourPlayer, setup func(Shape, *rand.Rand) string, rules Rules) (wins, draws float64) {
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, runtime.GOMAXPROCS(0))
	for i := range n {
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			seat := Color(i % 4)
			players := [4]fourPlayer{b, b, b, b}
			players[seat] = a
			w, ok, out := playFour(players, setup, rules, uint64(i/4))
			mu.Lock()
			defer mu.Unlock()
			switch {
			case ok && w == seat:
				wins++
			case !ok && !out[seat]:
				// Unfinished: a share of the win with everyone still in.
				alive := 0
				for _, o := range out {
					if !o {
						alive++
					}
				}
				draws++
				wins += 1 / float64(alive)
			case !ok:
				draws++
			}
		}()
	}
	wg.Wait()
	return wins / float64(n), draws / float64(n)
}

func level(s Strength) fourPlayer {
	return func(p *Position, r *rand.Rand) (Move, bool) { return BotMoveSeeing(p, nil, s, r) }
}

// TestFourLadder checks each level beats the one below.
func TestFourLadder(t *testing.T) {
	n, _ := strconv.Atoi(os.Getenv("FOUR_GAMES"))
	if n == 0 {
		t.Skip("set FOUR_GAMES to the number of games")
	}
	if f := os.Getenv("FOUR_W"); f != "" {
		// e.g. FOUR_W=hit=0.3,threat=0.1 tries those weights for Strongest.
		for _, kv := range strings.Split(f, ",") {
			k, v, _ := strings.Cut(kv, "=")
			x, _ := strconv.ParseFloat(v, 64)
			*map[string]*float64{"hit": &strongest.hit, "dodge": &strongest.dodge, "king": &strongest.king, "take": &strongest.take,
				"threat": &strongest.threat, "centre": &strongest.centre, "pawn": &strongest.pawn, "noise": &strongest.noise}[k] = x
		}
	} else {
		w, _ := fourMatch(n, level(Careful), level(Greedy), BotSetup, warMode)
		t.Logf("Careful vs three Greedy: %.3f", w)
	}
	w, _ := fourMatch(n, level(Strongest), level(Careful), BotSetup, warMode)
	t.Logf("Strongest vs three Careful: %.3f", w)
}

// TestFourFog plays War Mode with fog and vision squares: the bot that
// guards its king against the fog and holds the vision squares, against
// three that don't. Over 400 games it won 0.296 (0.25 is level); the fog
// cost or the vision square alone did less, and doubling either no better.
func TestFourFog(t *testing.T) {
	n, _ := strconv.Atoi(os.Getenv("FOUR_GAMES"))
	if n == 0 {
		t.Skip("set FOUR_GAMES to the number of games")
	}
	blind := strongest
	blind.fog, blind.outpost = 0, 0
	rules := warMode
	rules.CenterVision = CenterAny
	tries := []weights{strongest}
	for _, try := range tries {
		w, _ := fourMatch(n, fogged(Strongest, &try), fogged(Strongest, &blind), BotSetup, rules)
		t.Logf("fog %.2f outpost %.2f vs three without: %.3f", try.fog, try.outpost, w)
	}
}

var warMode = Rules{Shape: Extended, Sync: true, JesterPawns: true}

// TestFourBotStrength seats today's bot against three of the one that shipped
// before, in War Mode. It takes a while, so it runs only when asked:
// FOUR_GAMES=400 go test ./four -run FourBotStrength -v
func TestFourBotStrength(t *testing.T) {
	n, _ := strconv.Atoi(os.Getenv("FOUR_GAMES"))
	if n == 0 {
		t.Skip("set FOUR_GAMES to the number of games")
	}
	start := time.Now()
	w, d := fourMatch(n, BotMove, level(Greedy), BotSetup, warMode)
	t.Logf("War Mode, new bot vs three old: score %.3f (0.25 is level; a game still going after 400 rounds is shared among those still in), unfinished %.3f, over %d games in %v", w, d, n, time.Since(start).Round(time.Second))
}

// fourSetupMatch is fourMatch with the same move bot everywhere, comparing
// setups: a's setup in one seat, b's in the other three.
func fourSetupMatch(n int, a, b func(Shape, *rand.Rand) string, rules Rules) float64 {
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, runtime.GOMAXPROCS(0))
	score := 0.0
	for i := range n {
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			seat := Color(i % 4)
			setup := func(s Shape, r *rand.Rand) string { return b(s, r) }
			k := 0
			mixed := func(s Shape, r *rand.Rand) string {
				defer func() { k++ }()
				if Color(k) == seat {
					return a(s, r)
				}
				return setup(s, r)
			}
			w, ok, out := playFour([4]fourPlayer{BotMove, BotMove, BotMove, BotMove}, mixed, rules, uint64(i/4))
			mu.Lock()
			defer mu.Unlock()
			switch {
			case ok && w == seat:
				score++
			case !ok && !out[seat]:
				alive := 0
				for _, o := range out {
					if !o {
						alive++
					}
				}
				score += 1 / float64(alive)
			}
		}()
	}
	wg.Wait()
	return score / float64(n)
}

func guardedSetup(s Shape, r *rand.Rand) string {
	n := s.SetupLen()
	for {
		b := []byte(army + string(make([]byte, 0)))
		cells := make([]byte, n)
		for i := range cells {
			cells[i] = '.'
		}
		perm := r.Perm(n)
		for i, pc := range b {
			cells[perm[i]] = pc
		}
		k := -1
		for i, c := range cells {
			if c == 'K' {
				k = i
			}
		}
		if k >= 8 {
			continue
		}
		ok := true
		for _, d := range [][2]int{{-1, 0}, {1, 0}, {-1, 1}, {0, 1}, {1, 1}} {
			f, row := k%8+d[0], k/8+d[1]
			if f < 0 || f > 7 {
				continue
			}
			if cells[row*8+f] == '.' {
				ok = false
			}
		}
		if ok {
			return string(cells)
		}
	}
}

func forwardSetup(s Shape, r *rand.Rand) string {
	std := []byte(BotSetup(s, r))
	// Pawns from the second row to the front row.
	front := (s.Rows() - 1) * 8
	for f := range 8 {
		std[front+f], std[8+f] = std[8+f], std[front+f]
	}
	return string(std)
}

func TestFourSetups(t *testing.T) {
	n, _ := strconv.Atoi(os.Getenv("FOUR_GAMES"))
	if n == 0 {
		t.Skip("set FOUR_GAMES to the number of games")
	}
	t.Logf("guarded vs standard-ish: %.3f", fourSetupMatch(n, guardedSetup, BotSetup, warMode))
	t.Logf("forward vs standard-ish: %.3f", fourSetupMatch(n, forwardSetup, BotSetup, warMode))
}
