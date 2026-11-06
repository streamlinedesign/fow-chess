package main

import (
	"path/filepath"
	"slices"
	"testing"
	"time"

	"chess/bots"
)

// Beating a bot unlocks the theme it wears; losing to it, or a guest beating
// it, unlocks nothing.
func TestBeatingABotUnlocksItsTheme(t *testing.T) {
	// Long after the beta, so beta themes (cyberpunk) don't count for this new account.
	t.Setenv("BETA_ENDS", "2000-01-01")
	site := newTestSite(t)
	ann := site.signup("ann")
	u, _ := site.store.UserByName("ann")
	me := userPlayer(u, "ann-browser")

	_, got := ann.do("GET", "/api/unlocks", "")
	if th := got["themes"].([]any); len(th) != 0 {
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

	play(bots.Rex, false)
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
		if th := un.themes(); slices.Equal(th, []string{"galaxy"}) {
			break
		} else if time.Now().After(deadline) {
			t.Fatalf("after beating Solomon and losing to Rex: %v", th)
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
	if th := got["themes"].([]any); len(th) != 1 || th[0] != "galaxy" {
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
