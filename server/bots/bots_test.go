package bots

import (
	"math/rand/v2"
	"testing"

	"chess/engine"
)

func pos(t *testing.T, fen string) engine.Position {
	t.Helper()
	p, err := engine.ParseFEN(fen)
	if err != nil {
		t.Fatal(err)
	}
	p.KingCapture = true
	return p
}

func rng() *rand.Rand { return rand.New(rand.NewPCG(1, 2)) }

func TestEveryLevelPlaysALegalMove(t *testing.T) {
	p := pos(t, engine.StartFEN)
	legal := p.LegalMoves()
	for _, l := range Levels {
		m, ok := Choose(l, p, nil, rng())
		if !ok {
			t.Fatalf("%s: no move", l)
		}
		found := false
		for _, lm := range legal {
			found = found || lm == m
		}
		if !found {
			t.Errorf("%s played %s, not a legal move", l, m)
		}
	}
}

func TestTakesTheKingItSees(t *testing.T) {
	// White's rook on e1 looks straight at the black king on e8.
	p := pos(t, "4k3/8/8/8/8/8/8/4RK2 w - - 0 1")
	seen := p.Vision(engine.White)
	for _, l := range Levels[1:] {
		m, _ := Choose(l, p, &seen, rng())
		if m.UCI() != "e1e8" {
			t.Errorf("%s played %s, should take the king", l, m)
		}
	}
}

func TestDoesNotHuntAKingItCannotSee(t *testing.T) {
	// The black king is behind its pawns and out of White's sight. A hidden
	// king must not read as already captured, which would make every move
	// look winning and the choice random: the bot should still take the
	// free knight it does see.
	p := pos(t, "4k3/3ppp2/8/4n3/8/8/4R3/4K3 w - - 0 1")
	seen := p.Vision(engine.White)
	if seen[engine.SquareAt(4, 7)] {
		t.Fatal("test setup: the king should be hidden")
	}
	for i := 0; i < 5; i++ {
		m, _ := Choose(Ramesses, p, &seen, rand.New(rand.NewPCG(uint64(i), 9)))
		if m.UCI() != "e2e5" {
			t.Errorf("run %d: Ramesses played %s instead of taking the knight", i, m)
		}
	}
}

func captureOf(p engine.Position, m engine.Move) engine.Kind { return p.Board[m.To].Kind() }

func TestAvoidsHangingTheQueen(t *testing.T) {
	// White's queen is attacked by a visible rook; Vercingetorix should move it or take the rook.
	p := pos(t, "4k3/8/8/8/3r4/8/8/3QK3 w - - 0 1")
	seen := p.Vision(engine.White)
	m, _ := Choose(Vercingetorix, p, &seen, rng())
	if m.UCI() != "d1d4" {
		t.Errorf("Vercingetorix played %s, should take the rook", m)
	}
}
