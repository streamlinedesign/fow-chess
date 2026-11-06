package main

import (
	"database/sql"
	"log"
	"math"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"chess/engine"
)

// Themed skins are earned: some are worn by a bot, and beating that bot
// unlocks its piece set, its tiles and its avatars; the rest come with a
// title (titles.go), earned by the highest rating reached. The plain colours
// are never locked. Rex inutilis, the weakest bot, plays the classic pieces
// and unlocks nothing.

// lockBeta is the reason for a theme everyone has during the beta, and keeps
// after it if their account was made before BETA_ENDS.
const lockBeta = "beta"

// lockRating is the reason for a theme earned by reaching a rating, as
// shown, in either rating, Eat the King or War Mode. Once reached it is kept.
func lockRating(n int) string { return "rating:" + strconv.Itoa(n) }

// ratingUnlocks are the ratings that earn a title and its theme, lowest first.
var ratingUnlocks = []int{1150, 1300, 1450, 1600, 1800, 2000, 2500}

// themeLocks is what unlocks each locked theme: "bot:<level>", lockRating or
// lockBeta. A theme that isn't here is free. Must match THEME_LOCK in
// web/src/sets.ts and BOT_SET in web/src/settings.ts.
var themeLocks = map[string]string{
	"king":        lockRating(2500),
	"grandmaster": lockRating(2000),
	"holy":        lockRating(1800),
	"gold":        lockRating(1600),
	"silver":      lockRating(1450),
	"bronze":      lockRating(1300),
	"blueprint":   lockRating(1150),
	"survey":      "bot:vercingetorix",
	"ancient":     "bot:ramesses",
	"galaxy":      "bot:solomon",
	"cyberpunk":   lockBeta,
}

// betaEnds is when the beta closed, from BETA_ENDS (2006-01-02 or RFC 3339);
// zero while it's open.
func betaEnds() time.Time {
	v := os.Getenv("BETA_ENDS")
	if t, err := time.Parse(time.RFC3339, v); err == nil {
		return t
	}
	if t, err := time.Parse(time.DateOnly, v); err == nil {
		return t
	}
	return time.Time{}
}

// unlocks is what a viewer has: their reasons, the highest rating they have
// reached, and when their account was made (zero for a guest).
type unlocks struct {
	reasons []string
	peak    int
	created time.Time
}

// has is whether a theme (or a plain colour) is unlocked.
func (u unlocks) has(theme string) bool {
	reason, locked := themeLocks[theme]
	if !locked {
		return true
	}
	if reason == lockBeta {
		end := betaEnds()
		return end.IsZero() || time.Now().Before(end) || (!u.created.IsZero() && u.created.Before(end))
	}
	return slices.Contains(u.reasons, reason)
}

// themes is every unlocked theme, in picker order.
func (u unlocks) themes() []string {
	out := []string{}
	for _, f := range setFamilies {
		if _, locked := themeLocks[f]; locked && u.has(f) {
			out = append(out, f)
		}
	}
	return out
}

// strip sets whatever in a skin is still locked back to the default.
func (u unlocks) strip(s *skin) {
	if !u.has(s.White) {
		s.White = "classic"
	}
	if !u.has(s.Black) {
		s.Black = "classic"
	}
	if !u.has(s.LightTiles) {
		s.LightTiles = ""
	}
	if !u.has(s.DarkTiles) {
		s.DarkTiles = ""
	}
}

// avatar is whether an avatar's set is unlocked.
func (u unlocks) avatar(a string) bool {
	set, _, _ := strings.Cut(a, "/")
	return a == "" || u.has(familyOf(set))
}

// allows is whether every part of a look is unlocked.
func (u unlocks) allows(l lookResponse) bool {
	s := l.skin
	u.strip(&s)
	return s == l.skin && u.avatar(l.Avatar)
}

// Unlocks is what a signed-in player has unlocked; a guest (id 0) has only what everyone has.
func (s *Store) Unlocks(id int64) (unlocks, error) {
	var u unlocks
	if id == 0 || s == nil {
		return u, nil
	}
	var now float64
	if err := s.db.QueryRow(`SELECT created_at, GREATEST(rating, four_rating) FROM users WHERE id = $1`, id).
		Scan(&u.created, &now); err != nil {
		return u, err
	}
	// The peak is the rating right now (before a rated game has recorded it)
	// or the highest rating mark recorded, so a mark reached once stays.
	u.peak = int(math.Round(now))
	rows, err := s.db.Query(`SELECT reason FROM unlocks WHERE user_id = $1`, id)
	if err != nil {
		return u, err
	}
	defer rows.Close()
	var recorded []string
	for rows.Next() {
		var r string
		if err := rows.Scan(&r); err != nil {
			return u, err
		}
		if n, ok := ratingOf(r); ok {
			u.peak = max(u.peak, n)
		} else {
			recorded = append(recorded, r)
		}
	}
	for _, n := range ratingUnlocks {
		if u.peak >= n {
			u.reasons = append(u.reasons, lockRating(n))
		}
	}
	u.reasons = append(u.reasons, recorded...)
	return u, rows.Err()
}

// stripLocked sets what a player sent in a skin but hasn't unlocked back to
// the default; a guest has only what everyone has. False if it couldn't tell.
// Called before taking a match's lock, so a match never waits on the database.
func (s *Store) stripLocked(playerID string, sk *skin) bool {
	if sk == nil {
		return true
	}
	var id int64
	if rest, ok := strings.CutPrefix(playerID, "u:"); ok {
		id, _ = strconv.ParseInt(rest, 10, 64)
	}
	u, err := s.Unlocks(id)
	if err != nil {
		return false
	}
	u.strip(sk)
	return true
}

// ratingOf is the rating in a "rating:<n>" reason.
func ratingOf(reason string) (int, bool) {
	rest, ok := strings.CutPrefix(reason, "rating:")
	if !ok {
		return 0, false
	}
	n, err := strconv.Atoi(rest)
	return n, err == nil
}

// rateUnlock records each rating unlock a player's rating has just reached,
// in the transaction that rated them, so it stays theirs if the rating falls
// again.
func rateUnlock(tx *sql.Tx, id int64, rating float64) error {
	for _, n := range ratingUnlocks {
		if math.Round(rating) < float64(n) {
			continue
		}
		if _, err := tx.Exec(`INSERT INTO unlocks (user_id, reason, created_at) VALUES ($1, $2, now()) ON CONFLICT DO NOTHING`, id, lockRating(n)); err != nil {
			return err
		}
	}
	return nil
}

// Unlock records a reason for a player; a second time does nothing.
func (s *Store) Unlock(id int64, reason string) error {
	_, err := s.db.Exec(`INSERT INTO unlocks (user_id, reason, created_at) VALUES ($1, $2, now()) ON CONFLICT DO NOTHING`, id, reason)
	return err
}

// botWinner is the signed-in player who just beat the bot in this round, or 0.
// Callers hold m.mu.
func (m *Match) botWinner() int64 {
	c := m.botColor()
	if c < 0 || !m.game.Over() {
		return 0
	}
	won := engine.WhiteWins
	if c == white {
		won = engine.BlackWins
	}
	if m.game.Outcome() != won {
		return 0
	}
	return m.users[1-c]
}

// rewardBotWin unlocks the bot's theme for a signed-in player who beat it,
// once a round. A bot that wears no theme unlocks nothing. Callers hold m.mu.
func (m *Match) rewardBotWin() {
	if m.rewarded == m.round+1 || botTheme(m.settings.Bot) == "" {
		return
	}
	id := m.botWinner()
	if id == 0 {
		return
	}
	m.rewarded = m.round + 1
	reason := "bot:" + m.settings.Bot
	go func() {
		if err := m.store.Unlock(id, reason); err != nil {
			log.Printf("unlock %s for %d: %v", reason, id, err)
		}
	}()
}

// botTheme is the theme a bot wears and unlocks, or "".
func botTheme(level string) string {
	for theme, reason := range themeLocks {
		if reason == "bot:"+level {
			return theme
		}
	}
	return ""
}

func unlockRoutes(mux *http.ServeMux, store *Store, auth *Auth) {
	// What the viewer may wear: the locked themes they have, and what unlocks each.
	mux.HandleFunc("GET /api/unlocks", func(w http.ResponseWriter, r *http.Request) {
		var id int64
		if u := auth.user(r); u != nil {
			id = u.ID
		}
		u, err := store.Unlocks(id)
		if serverError(w, err) {
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		writeJSON(w, map[string]any{"themes": u.themes(), "locks": themeLocks})
	})
}
