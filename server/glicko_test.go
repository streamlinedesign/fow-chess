package main

import (
	"math"
	"testing"
)

// The worked example from Glickman's Glicko-2 paper.
func TestGlickoPaperExample(t *testing.T) {
	p := glicko{1500, 200, 0.06}
	got := p.rate([]glickoResult{
		{glicko{1400, 30, 0.06}, 1},
		{glicko{1550, 100, 0.06}, 0},
		{glicko{1700, 300, 0.06}, 0},
	})
	near := func(a, b, tol float64) bool { return math.Abs(a-b) < tol }
	if !near(got.Rating, 1464.06, 0.01) || !near(got.Deviation, 151.52, 0.01) || !near(got.Volatility, 0.05999, 0.00001) {
		t.Errorf("got %+v, want 1464.06 / 151.52 / 0.05999", got)
	}
}

func TestGlickoNewPlayersMoveEvenly(t *testing.T) {
	w := newPlayer.rate([]glickoResult{{newPlayer, 1}})
	b := newPlayer.rate([]glickoResult{{newPlayer, 0}})
	if w.Rating <= 1500 || b.Rating >= 1500 || math.Abs((w.Rating-1500)-(1500-b.Rating)) > 1e-9 {
		t.Errorf("winner %v, loser %v", w.Rating, b.Rating)
	}
	d := newPlayer.rate([]glickoResult{{newPlayer, 0.5}})
	if math.Abs(d.Rating-1500) > 1e-9 {
		t.Errorf("draw between equals moved the rating to %v", d.Rating)
	}
}
