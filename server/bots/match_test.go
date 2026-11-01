package bots

import (
	"math/rand/v2"
	"os"
	"strconv"
	"sync"
	"testing"

	"chess/engine"
)

// playFog plays one fog game between level with and without guarding its
// king, the guarded one White when guardedWhite, and returns White's score: 1
// for a win, 0.5 for a draw.
func playFog(level Level, guardedWhite bool, seed uint64) float64 {
	r := rand.New(rand.NewPCG(seed, seed*7+1))
	p, _ := engine.ParseFEN(engine.StartFEN)
	p.KingCapture = true
	for ply := 0; ply < 300; ply++ {
		me := p.Turn
		seen := p.Vision(me)
		m, ok := choose(level, p, &seen, r, (me == engine.White) == guardedWhite)
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

// TestGuardedBeatsPlain pits a bot that guards its king against the fog
// against the same level that doesn't. It takes a while, so it runs only when
// asked: BOTS_MATCH=2000 go test ./bots -run GuardedBeatsPlain -v
// (BOTS_LEVEL picks the level, BOTS_EXPOSURE the cost of each way in).
func TestGuardedBeatsPlain(t *testing.T) {
	n, _ := strconv.Atoi(os.Getenv("BOTS_MATCH"))
	if n == 0 {
		t.Skip("set BOTS_MATCH to the number of games")
	}
	if v, err := strconv.ParseFloat(os.Getenv("BOTS_EXPOSURE"), 64); err == nil {
		exposureCost = v
	}
	level := Level(os.Getenv("BOTS_LEVEL"))
	if level == "" {
		level = Ramesses
	}
	var mu sync.Mutex
	var wg sync.WaitGroup
	total := 0.0
	sem := make(chan struct{}, 8)
	for i := range n {
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			s := playFog(level, i%2 == 0, uint64(i))
			if i%2 == 1 {
				s = 1 - s
			}
			mu.Lock()
			total += s
			mu.Unlock()
		}()
	}
	wg.Wait()
	t.Logf("%s guarded vs plain: %.3f over %d games", level, total/float64(n), n)
}
