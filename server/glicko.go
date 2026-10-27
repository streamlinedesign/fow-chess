package main

import "math"

// Glicko-2, from Mark Glickman's paper (http://www.glicko.net/glicko/glicko2.pdf).
// Every rated game is its own rating period, as on most chess sites, so a
// rating moves after each game instead of in batches.

type glicko struct {
	Rating     float64
	Deviation  float64
	Volatility float64
}

type glickoResult struct {
	Opponent glicko
	Score    float64 // 1 win, 0.5 draw, 0 loss
}

const (
	glickoScale  = 173.7178
	glickoTau    = 0.5 // how fast volatility may change
	glickoMinDev = 45  // a floor, so ratings never stop moving
	glickoMaxDev = 350
)

var newPlayer = glicko{Rating: 1500, Deviation: glickoMaxDev, Volatility: 0.06}

// rate returns p's rating after the given results.
func (p glicko) rate(results []glickoResult) glicko {
	mu := (p.Rating - 1500) / glickoScale
	phi := p.Deviation / glickoScale
	if len(results) == 0 {
		return p
	}

	g := func(phi float64) float64 { return 1 / math.Sqrt(1+3*phi*phi/(math.Pi*math.Pi)) }
	var vInv, delta float64
	for _, r := range results {
		muJ := (r.Opponent.Rating - 1500) / glickoScale
		gJ := g(r.Opponent.Deviation / glickoScale)
		e := 1 / (1 + math.Exp(-gJ*(mu-muJ)))
		vInv += gJ * gJ * e * (1 - e)
		delta += gJ * (r.Score - e)
	}
	v := 1 / vInv
	delta *= v

	// New volatility by the Illinois algorithm (step 5 of the paper).
	a := math.Log(p.Volatility * p.Volatility)
	f := func(x float64) float64 {
		ex := math.Exp(x)
		d := phi*phi + v + ex
		return ex*(delta*delta-phi*phi-v-ex)/(2*d*d) - (x-a)/(glickoTau*glickoTau)
	}
	A := a
	var B float64
	if delta*delta > phi*phi+v {
		B = math.Log(delta*delta - phi*phi - v)
	} else {
		k := 1.0
		for f(a-k*glickoTau) < 0 {
			k++
		}
		B = a - k*glickoTau
	}
	fA, fB := f(A), f(B)
	for i := 0; math.Abs(B-A) > 1e-6 && i < 100; i++ {
		C := A + (A-B)*fA/(fB-fA)
		fC := f(C)
		if fC*fB <= 0 {
			A, fA = B, fB
		} else {
			fA /= 2
		}
		B, fB = C, fC
	}
	vol := math.Exp(A / 2)

	phiStar := math.Sqrt(phi*phi + vol*vol)
	phiNew := 1 / math.Sqrt(1/(phiStar*phiStar)+1/v)
	muNew := mu + phiNew*phiNew*delta/v

	return glicko{
		Rating:     muNew*glickoScale + 1500,
		Deviation:  min(max(phiNew*glickoScale, glickoMinDev), glickoMaxDev),
		Volatility: vol,
	}
}
