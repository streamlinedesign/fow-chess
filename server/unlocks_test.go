package main

import (
	"path/filepath"
	"slices"
	"testing"
	"time"

	"chess/bots"
)

// Beating a bot unlocks the theme it wears; losing to it, beating Rex (who
// wears none), or a guest beating it, unlocks nothing.
func TestBeatingABotUnlocksItsTheme(t *testing.T) {
	site := newTestSite(t)
	ann := site.signup("ann")
	u, _ := site.store.UserByName("ann")
	me := userPlayer(u, "ann-browser")

	_, got := ann.do("GET", "/api/unlocks", "")
	// Cyberpunk is everyone's during the beta; the bots' themes start locked.
	if th := got["themes"].([]any); len(th) != 1 || th[0] != "cyberpunk" {
		t.Fatalf("a new account has %v", th)
	}

	play := func(level bots.Level, botResigns bool) *Match {
		settings := Settings{Fog: true, KingCapture: true, Bot: string(level)}
		settings.normalize()
		m := site.hub.CreateBetween(me, botPlayer(level), settings)
		c := newClient(me.key)
		m.join(c, me)
		m.handle(c, inMsg{Type: "move", UCI: "e2e4"}) // no resigning before the first move
		if botResigns {
			m.handle(newClient("bot:"+string(level)), inMsg{Type: "resign"})
		} else {
			m.handle(c, inMsg{Type: "resign"})
		}
		return m
	}

	rex := play(bots.Rex, true)
	rex.mu.Lock()
	if st := rex.base(); st.Unlocked != "" {
		t.Errorf("beating Rex unlocked %q", st.Unlocked)
	}
	rex.mu.Unlock()
	play(bots.Vercingetorix, false)
	m := play(bots.Solomon, true)
	m.mu.Lock()
	if st := m.base(); st.Unlocked != "galaxy" {
		t.Errorf("the state says %q was unlocked", st.Unlocked)
	}
	m.mu.Unlock()

	deadline := time.Now().Add(5 * time.Second)
	for {
		un, err := site.store.Unlocks(u.ID)
		if err != nil {
			t.Fatal(err)
		}
		if th := un.themes(); slices.Equal(th, []string{"galaxy", "cyberpunk"}) {
			break
		} else if time.Now().After(deadline) {
			t.Fatalf("after beating Rex and Solomon and losing to Vercingetorix: %v", th)
		}
		time.Sleep(20 * time.Millisecond)
	}
	if code, _ := ann.do("PUT", "/api/prefs", `{"white":"galaxy","black":"galaxy","lightTiles":"galaxy","darkTiles":"galaxy","avatar":"galaxy-dark/K"}`); code != 200 {
		t.Errorf("galaxy once unlocked: %d", code)
	}
	if code, _ := ann.do("PUT", "/api/prefs", `{"white":"holy"}`); code != 403 {
		t.Errorf("holy, still locked: %d", code)
	}
	_, got = ann.do("GET", "/api/unlocks", "")
	if th := got["themes"].([]any); len(th) != 2 || th[0] != "galaxy" || th[1] != "cyberpunk" {
		t.Errorf("unlocks: %v", got)
	}

	// A guest who wins has nothing to keep it on.
	store := openTestStore(t, filepath.Join(t.TempDir(), "x"))
	guest := guestPlayer("g")
	g := NewHub(store).CreateBetween(guest, botPlayer(bots.Rex), Settings{Bot: "rex"})
	g.handle(newClient(guest.key), inMsg{Type: "move", UCI: "e2e4"})
	g.handle(newClient("bot:rex"), inMsg{Type: "resign"})
	g.mu.Lock()
	if st := g.base(); st.Unlocked != "" {
		t.Errorf("a guest unlocked %q", st.Unlocked)
	}
	g.mu.Unlock()
}

// Rex inutilis plays the classic pieces; holy, the set he used to wear, is
// earned by reaching 2000 in either rating, and kept if the rating falls.
func TestRatingUnlocksHoly(t *testing.T) {
	if botTheme(string(bots.Rex)) != "" {
		t.Errorf("Rex wears %q", botTheme(string(bots.Rex)))
	}
	if themeLocks["holy"] != lockRating {
		t.Fatalf("holy is locked by %q", themeLocks["holy"])
	}
	store := openTestStore(t, filepath.Join(t.TempDir(), "x"))
	has := func(name string) bool {
		t.Helper()
		u, err := store.UserByName(name)
		if err != nil {
			t.Fatal(err)
		}
		un, err := store.Unlocks(u.ID)
		if err != nil {
			t.Fatal(err)
		}
		return un.has("holy")
	}
	set := func(name, column string, rating float64) {
		t.Helper()
		if _, err := store.db.Exec(`UPDATE users SET `+column+` = $2 WHERE username = $1`, name, rating); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"ann", "ben", "cy"} {
		if _, err := store.CreateUser(name, []byte("x")); err != nil {
			t.Fatal(err)
		}
	}
	set("ann", "rating", 1999.4)
	if has("ann") {
		t.Error("1999 is not 2000")
	}
	set("ann", "rating", 1999.6) // shown as 2000
	set("ben", "four_rating", 2100)
	if !has("ann") || !has("ben") || has("cy") {
		t.Error("2000 in either rating unlocks holy")
	}

	// A rated game that takes cy to 2000 records it, so it stays.
	cy, _ := store.UserByName("cy")
	tx, err := store.db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if err := rateUnlock(tx, cy.ID, 2003); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	set("cy", "rating", 1980)
	if !has("cy") {
		t.Error("holy went away when the rating fell")
	}
}

// A beta theme is everyone's while the beta is open, and afterwards only
// accounts made before it closed keep it.
func TestBetaThemes(t *testing.T) {
	themeLocks["beta-test"] = lockBeta
	t.Cleanup(func() { delete(themeLocks, "beta-test") })
	guest := unlocks{}
	early := unlocks{created: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	late := unlocks{created: time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)}

	t.Setenv("BETA_ENDS", "")
	if !guest.has("beta-test") || !late.has("beta-test") {
		t.Error("an open beta is everyone's")
	}
	t.Setenv("BETA_ENDS", "2026-06-01")
	if guest.has("beta-test") || late.has("beta-test") || !early.has("beta-test") {
		t.Error("after the beta only early accounts keep it")
	}
	if !guest.has("red") || !guest.has("classic") || !guest.has("") || guest.has("galaxy") {
		t.Error("colours are free, themes are not")
	}
}

// Cyberpunk is the beta set: every player has it now, guests included, and
// after BETA_ENDS the accounts made before then keep it.
func TestCyberpunkIsBeta(t *testing.T) {
	if themeLocks["cyberpunk"] != lockBeta {
		t.Fatalf("cyberpunk is locked by %q", themeLocks["cyberpunk"])
	}
	for _, list := range [][]string{setFamilies, tileSkins} {
		if !slices.Contains(list, "cyberpunk") {
			t.Error("cyberpunk missing from a pick list")
		}
	}
	if !slices.Contains(pieceSets, "cyberpunk-light") || !slices.Contains(pieceSets, "cyberpunk-dark") {
		t.Error("cyberpunk avatars missing")
	}
	t.Setenv("BETA_ENDS", "")
	if !(unlocks{}).has("cyberpunk") {
		t.Error("a guest should have cyberpunk during the beta")
	}
	t.Setenv("BETA_ENDS", "2026-06-01")
	early := unlocks{created: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	if (unlocks{}).has("cyberpunk") || !early.has("cyberpunk") {
		t.Error("after the beta only beta accounts keep cyberpunk")
	}
}
