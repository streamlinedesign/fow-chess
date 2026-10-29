package sim

import (
	"math/rand/v2"
	"strings"
	"testing"

	"chess/engine"
)

// Every bot finishes games in every mode, with and without fog, and a seed
// plays the same game every time.
func TestPlay(t *testing.T) {
	seed := uint64(1)
	for name := range Bots {
		for _, mode := range []string{"standard", "backRow", "battleship"} {
			for _, fog := range []bool{true, false} {
				for _, rules := range []engine.Rules{{KingCapture: true}, {}, {KingCapture: true, Hill: true, JumpKing: true, BackPawns: true}} {
					seed++
					c := Config{White: name, Black: "greedy", Mode: mode, Fog: fog, Rules: rules, Seed: seed, Moves: true, Guarded: seed%2 == 0}
					a, err := Play(c)
					if err != nil {
						t.Fatal(err)
					}
					b, _ := Play(c)
					if a.Moves != b.Moves || a.Result != b.Result {
						t.Fatalf("%s %s: seed %d played two different games", name, mode, seed)
					}
					if a.Result == "*" {
						t.Fatalf("%s %s: game didn't finish", name, mode)
					}
					replay(t, a, rules)
				}
			}
		}
	}
}

// replay plays a record's moves through the engine and checks it ends the same way.
func replay(t *testing.T, r Record, rules engine.Rules) {
	t.Helper()
	start := r.StartFEN
	if start == "" {
		start = engine.StartFEN
	}
	g, err := engine.NewGameFromFEN(start, rules)
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range strings.Fields(r.Moves) {
		if _, err := g.PlayUCI(m); err != nil {
			t.Fatalf("seed %d: %s: %v", r.Seed, m, err)
		}
	}
	if g.Over() && (g.Outcome().String() != r.Result || g.Method().String() != r.Method) {
		t.Fatalf("seed %d: replay ended %s by %s, record says %s by %s", r.Seed, g.Outcome(), g.Method(), r.Result, r.Method)
	}
}

// Random setups put each army only where the site's setup modes allow.
func TestRandomSetup(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	for range 200 {
		fen, err := RandomSetup("battleship", true, true, rng)
		if err != nil {
			t.Fatal(err)
		}
		p, err := engine.ParseFEN(fen)
		if err != nil {
			t.Fatal(err)
		}
		var count [16]int
		for sq, pc := range p.Board {
			if pc.IsEmpty() {
				continue
			}
			count[pc]++
			rank := engine.Square(sq).Rank()
			if pc.Color() == engine.White && rank > 3 || pc.Color() == engine.Black && rank < 4 {
				t.Fatalf("%s: piece outside its rows", fen)
			}
			if pc.Kind() == engine.King && engine.OnHill(engine.Square(sq)) {
				t.Fatalf("%s: king starts on the hill", fen)
			}
		}
		if count[engine.NewPiece(engine.White, engine.Pawn)] != 8 || count[engine.NewPiece(engine.Black, engine.Queen)] != 1 {
			t.Fatalf("%s: wrong army", fen)
		}
	}
	fen, _ := RandomSetup("backRow", false, false, rng)
	if !strings.Contains(fen, "/pppppppp/8/8/8/8/PPPPPPPP/") {
		t.Fatalf("back row setup moved pawns: %s", fen)
	}
}

// A fogged view shows nothing the player's pieces can't see.
func TestViewHidesFog(t *testing.T) {
	g := engine.NewGameWithRules(engine.Rules{KingCapture: true})
	v := viewOf(g, engine.White, true, engine.Rules{KingCapture: true})
	for i, pc := range v.Board {
		if pc.Is(engine.Black) {
			t.Fatalf("white sees a black piece on %s at the start", engine.Square(i))
		}
	}
	if len(v.Legal) != 20 {
		t.Fatalf("want 20 legal moves, got %d", len(v.Legal))
	}
}
