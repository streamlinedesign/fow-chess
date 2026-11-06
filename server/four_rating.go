package main

import (
	"database/sql"
	"encoding/json"
	"math"
	"slices"
)

// Ranked four-player games move a four-player Glicko-2 rating of their own.
// A game counts as a one-against-one result against each of the other three:
// a win against everyone who finished below you, a draw against anyone who
// finished level, a loss against everyone above.

// fourRatings reads the players' four-player ratings, 0 for a seat with no user.
func (h *FourHub) fourRatings(users [4]int64) [4]int {
	var out [4]int
	if h.db == nil {
		return out
	}
	for c, id := range users {
		if id == 0 {
			continue
		}
		var r float64
		if err := h.db.QueryRow(`SELECT four_rating FROM users WHERE id = $1`, id).Scan(&r); err == nil {
			out[c] = int(math.Round(r))
		}
	}
	return out
}

// fourResults is what a player in seat c scored against each other seat,
// given the places they finished in.
func fourResults(c int, places []int, ratings [4]glicko, rated [4]bool) []glickoResult {
	var out []glickoResult
	for o := range places {
		if o == c || !rated[o] {
			continue
		}
		score := 0.5
		switch {
		case places[c] < places[o]:
			score = 1
		case places[c] > places[o]:
			score = 0
		}
		out = append(out, glickoResult{ratings[o], score})
	}
	return out
}

// rateFour updates the four players' ratings for a finished ranked game and
// records the changes on it. It returns nil if the game was already rated.
func (h *FourHub) rateFour(matchID string, users [4]int64, places []int) (*[4]int, error) {
	tx, err := h.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var done bool
	err = tx.QueryRow(`SELECT rating_diff IS NOT NULL FROM four_games WHERE match_id = $1 FOR UPDATE`, matchID).Scan(&done)
	if err != nil || done {
		return nil, err
	}
	// Lock in id order so games finishing at once can't deadlock.
	ids := slices.DeleteFunc(slices.Clone(users[:]), func(id int64) bool { return id == 0 })
	slices.Sort(ids)
	var before [4]glicko
	var rated [4]bool
	for _, id := range ids {
		var g glicko
		err := tx.QueryRow(`SELECT four_rating, four_deviation, four_volatility FROM users WHERE id = $1 FOR UPDATE`, id).
			Scan(&g.Rating, &g.Deviation, &g.Volatility)
		if err == sql.ErrNoRows {
			continue
		}
		if err != nil {
			return nil, err
		}
		for c, u := range users {
			if u == id {
				before[c], rated[c] = g, true
			}
		}
	}
	var diff [4]int
	for c := range users {
		if !rated[c] {
			continue
		}
		after := before[c].rate(fourResults(c, places, before, rated))
		diff[c] = int(math.Round(after.Rating)) - int(math.Round(before[c].Rating))
		if _, err := tx.Exec(`UPDATE users SET four_rating = $2, four_deviation = $3, four_volatility = $4,
			four_rated_games = four_rated_games + 1 WHERE id = $1`,
			users[c], after.Rating, after.Deviation, after.Volatility); err != nil {
			return nil, err
		}
		if err := rateUnlock(tx, users[c], after.Rating); err != nil {
			return nil, err
		}
	}
	js, _ := json.Marshal(diff)
	if _, err := tx.Exec(`UPDATE four_games SET rating_diff = $2 WHERE match_id = $1`, matchID, string(js)); err != nil {
		return nil, err
	}
	return &diff, tx.Commit()
}
