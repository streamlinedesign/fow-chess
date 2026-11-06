package main

import "math"

// Rank scores. Both ranked games, Eat the King and War Mode, keep a score of
// their own that starts at startRating and moves by whole points. The winner
// takes points from a range set by the clock, more for beating someone rated
// above you; the losers give up the same amount or part of it. A player's
// first loss of the day (UTC) in each game costs less.
//
// Keep in step with the "Rank scores" section of the rules (web/src/Rules.tsx
// and the i18n files).

const startRating = 1000

// pointsSpread is how far apart two ratings must be for the winner to get
// the top or the bottom of the range.
const pointsSpread = 400

// longClock is whether a clock counts as a long game: an hour or more each,
// or a clock per move (daily and weekly). Anything shorter is a short game.
func longClock(t TimeControl) bool {
	return t.PerMove > 0 || t.Base >= 3600
}

// pointsRange is the fewest and most points a win is worth on this clock.
func pointsRange(t TimeControl) (lo, hi int) {
	if longClock(t) {
		return 22, 30
	}
	return 18, 23
}

// winValue is what a win is worth on this clock to a player rated winner
// against an opponent (or the average of the opponents) rated loser: the
// middle of the range at equal ratings, the top when the loser is rated
// pointsSpread or more above, the bottom when pointsSpread or more below.
func winValue(t TimeControl, winner, loser float64) int {
	lo, hi := pointsRange(t)
	d := min(max(loser-winner, -pointsSpread), pointsSpread)
	x := (d + pointsSpread) / (2 * pointsSpread)
	return lo + int(math.Round(x*float64(hi-lo)))
}

// half is half of n points, rounded up.
func half(n int) int { return (n + 1) / 2 }

// duelPoints is how Eat the King moves the winner's and the loser's ratings:
// the loser loses what the winner wins, half of it on their first loss of
// the day. A draw moves nobody. A Sniper Win counts sniperWeight of a win.
func duelPoints(t TimeControl, winner, loser float64, firstLoss, sniper bool) (win, lose int) {
	w := winValue(t, winner, loser)
	if sniper {
		w = int(math.Round(float64(w) * sniperWeight))
	}
	l := w
	if firstLoss {
		l = half(w)
	}
	return w, -l
}

// warPoints is how a War Mode game moves each seat's rating, given where
// each finished (1 best, level players share a place), their ratings, which
// seats are rated, and whose loss would be their first of the day.
//
// The winner gets winValue against the average of the other rated players.
// Second place loses that amount, third and fourth half of it. On a first
// loss of the day second place loses half, third nothing, fourth a quarter.
// When no one finished first alone (a draw) the players sharing first place
// don't move, and everyone below loses as above, by the points the game
// would have been worth at their ratings.
func warPoints(t TimeControl, places []int, ratings [4]float64, rated, firstLoss [4]bool) [4]int {
	var out [4]int
	winner, firsts := -1, 0
	for c := range places {
		if rated[c] && places[c] == 1 {
			winner, firsts = c, firsts+1
		}
	}
	// The stake: what first place wins, against the rest of the table.
	var stake int
	if firsts == 1 {
		stake = winValue(t, ratings[winner], fieldAverage(ratings, rated, winner))
		out[winner] = stake
	}
	for c := range places {
		if !rated[c] || places[c] == 1 {
			continue
		}
		s := stake
		if firsts != 1 {
			// A draw: what a win would have been worth to an average player at the table.
			avg := fieldAverage(ratings, rated, -1)
			s = winValue(t, avg, avg)
		}
		out[c] = -warLoss(places[c], s, firstLoss[c])
	}
	return out
}

// warLoss is what a player who finished in this place (2 to 4) loses when
// first place won stake points.
func warLoss(place, stake int, firstLoss bool) int {
	switch {
	case place == 2 && firstLoss:
		return half(stake)
	case place == 2:
		return stake
	case place == 3 && firstLoss:
		return 0
	case place == 3:
		return half(stake)
	case firstLoss: // fourth
		return (stake + 2) / 4
	}
	return half(stake)
}

// fieldAverage is the average rating of the rated seats other than skip.
func fieldAverage(ratings [4]float64, rated [4]bool, skip int) float64 {
	var sum float64
	n := 0
	for c := range ratings {
		if rated[c] && c != skip {
			sum += ratings[c]
			n++
		}
	}
	if n == 0 {
		return startRating
	}
	return sum / float64(n)
}

// utcToday is today's date in UTC, in SQL: the day a first loss is counted in.
const utcToday = `(now() AT TIME ZONE 'UTC')::date`
