package bots

import (
	"math/rand/v2"
	"testing"

	"chess/engine"
)

// sight must see exactly what the engine's Vision sees, under every rule.
func TestSightMatchesVision(t *testing.T) {
	r := rand.New(rand.NewPCG(3, 4))
	for game := range 200 {
		p, _ := engine.ParseFEN(engine.StartFEN)
		p.KingCapture = true
		p.JumpKing = game%5 == 1
		p.JesterPawns = game%5 == 2
		if game%3 == 0 {
			p.CenterVision = engine.CenterAny
		}
		for range 120 {
			for _, c := range []engine.Color{engine.White, engine.Black} {
				if got, want := sight(&p, c), p.Vision(c); got != want {
					t.Fatalf("game %d, %s, %s: sight differs from Vision", game, p.FEN(), c)
				}
			}
			ms := p.LegalMoves()
			if len(ms) == 0 || p.KingSquare(engine.White) == engine.NoSquare || p.KingSquare(engine.Black) == engine.NoSquare {
				break
			}
			p = p.Apply(ms[r.IntN(len(ms))])
		}
	}
}
