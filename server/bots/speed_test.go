package bots

import (
	"math/rand/v2"
	"testing"
	"time"

	"chess/engine"
)

// A bot must answer quickly even from a busy middlegame.
func TestSolomonIsQuick(t *testing.T) {
	p := pos(t, "r1bqkb1r/pppp1ppp/2n2n2/4p3/2B1P3/5N2/PPPP1PPP/RNBQK2R w KQkq - 4 4")
	start := time.Now()
	for i := 0; i < 3; i++ {
		if _, ok := Choose(Solomon, p, nil, rng()); !ok {
			t.Fatal("no move")
		}
	}
	if d := time.Since(start) / 3; d > 2*time.Second {
		t.Errorf("Solomon took %v per move", d)
	} else {
		t.Logf("Solomon: %v per move", d)
	}
	_ = engine.White
}

// A busy Solomon thinks less than it would otherwise.
func TestBusySolomonIsQuicker(t *testing.T) {
	p := pos(t, "r1bqkb1r/pppp1ppp/2n2n2/4p3/2B1P3/5N2/PPPP1PPP/RNBQK2R w KQkq - 4 4")
	seen := p.Vision(p.Turn)
	timeIt := func(choose func(Level, engine.Position, *[64]bool, bool, *rand.Rand) (engine.Move, bool)) time.Duration {
		start := time.Now()
		for i := 0; i < 3; i++ {
			if _, ok := choose(Solomon, p, &seen, true, rng()); !ok {
				t.Fatal("no move")
			}
		}
		return time.Since(start) / 3
	}
	full, busy := timeIt(ChooseWarned), timeIt(ChooseBusy)
	t.Logf("Solomon: %v per move, %v when busy", full, busy)
	if busy*2 > full {
		t.Errorf("busy Solomon took %v, not well under %v", busy, full)
	}
}
