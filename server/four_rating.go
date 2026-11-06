package main

import (
	"database/sql"
	"encoding/json"
	"math"
	"slices"
)

// Ranked four-player games move a War Mode rating of their own, by the
// points in warPoints.

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

// rateFour updates the four players' ratings for a finished ranked game and
// records the changes on it. It returns nil if the game was already rated.
func (h *FourHub) rateFour(matchID string, users [4]int64, places []int, clock TimeControl) (*[4]int, error) {
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
	var before [4]float64
	var rated, firstLoss [4]bool
	for _, id := range ids {
		var rating float64
		var first bool
		err := tx.QueryRow(`SELECT four_rating, four_lost_on IS DISTINCT FROM `+utcToday+` FROM users WHERE id = $1 FOR UPDATE`, id).
			Scan(&rating, &first)
		if err == sql.ErrNoRows {
			continue
		}
		if err != nil {
			return nil, err
		}
		for c, u := range users {
			if u == id {
				before[c], rated[c], firstLoss[c] = rating, true, first
			}
		}
	}
	diff := warPoints(clock, places, before, rated, firstLoss)
	for c := range users {
		if !rated[c] {
			continue
		}
		after := before[c] + float64(diff[c])
		lost := places[c] != 1
		if _, err := tx.Exec(`UPDATE users SET four_rating = $2, four_rated_games = four_rated_games + 1,
			four_lost_on = CASE WHEN $3 THEN `+utcToday+` ELSE four_lost_on END WHERE id = $1`,
			users[c], after, lost); err != nil {
			return nil, err
		}
		if err := rateUnlock(tx, users[c], after); err != nil {
			return nil, err
		}
	}
	js, _ := json.Marshal(diff)
	if _, err := tx.Exec(`UPDATE four_games SET rating_diff = $2 WHERE match_id = $1`, matchID, string(js)); err != nil {
		return nil, err
	}
	return &diff, tx.Commit()
}
