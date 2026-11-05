package main

import (
	"path/filepath"
	"testing"
	"time"

	"chess/bots"
)

func TestSyncMove(t *testing.T) {
	store := openTestStore(t, filepath.Join(t.TempDir(), "x"))
	settings := Settings{Mods: true, Sync: true, Fog: true, Time: TimeControl{Base: 180, Inc: 2}}
	settings.normalize()
	if !settings.KingCapture || !settings.valid() {
		t.Fatalf("sync settings: %+v", settings)
	}
	alice, bob := guestPlayer("alice"), guestPlayer("bob")
	m := NewHub(store).Create(alice, settings)
	cw, cb := newClient(alice.key), newClient(bob.key)
	m.join(cw, alice)
	m.join(cb, bob)
	s := lastState(t, cb)
	if len(s["legal"].([]any)) == 0 || len(s["ticking"].([]any)) != 2 {
		t.Fatalf("black should have moves and both clocks run: %v %v", s["legal"], s["ticking"])
	}

	m.handle(cb, inMsg{Type: "move", UCI: "e7e5"})
	m.handle(cb, inMsg{Type: "move", UCI: "d7d5"}) // locked in: ignored
	s = lastState(t, cb)
	if s["planned"] != "e7e5" || len(s["legal"].([]any)) != 0 || s["ply"].(float64) != 0 {
		t.Fatalf("after sending: planned %v legal %v", s["planned"], s["legal"])
	}
	if w := lastState(t, cw); w["sent"].(map[string]any)["b"] != true || w["planned"] != nil {
		t.Fatalf("white should see that black sent, not what: %v", w)
	}
	m.handle(cw, inMsg{Type: "move", UCI: "e2e4"})
	if got := m.game.Tokens(); len(got) != 1 || got[0] != "s:e2e4,e7e5" {
		t.Fatalf("tokens %v", got)
	}
	s = lastState(t, cw)
	if s["ply"].(float64) != 2 || s["history"].([]any)[1] != "?" || s["last"].(map[string]any)["to"] != "e4" {
		t.Fatalf("white's view: %v %v", s["history"], s["last"])
	}
	if !near(m.clock[white], 182*time.Second) {
		t.Errorf("white has %v, want about 182s", m.clock[white])
	}

	// Black runs out while White has sent a move: Black loses on time.
	m.handle(cw, inMsg{Type: "move", UCI: "g1f3"})
	rewind(m, 190*time.Second)
	m.checkFlag()
	if m.game.Outcome().String() != "1-0" || m.game.Method().String() != "time" {
		t.Fatalf("got %s by %s", m.game.Outcome(), m.game.Method())
	}
	var rec *gameRecord
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if rec, _ = store.LatestGame(m.id); rec != nil && rec.Result != "" {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	g, err := replay(rec)
	if err != nil || g.Outcome().String() != "1-0" || g.Method().String() != "time" || len(g.History()) != 2 {
		t.Fatalf("replay: %v %v %v", err, g.Outcome(), g.Method())
	}
}

func TestSyncMoveBot(t *testing.T) {
	store := openTestStore(t, filepath.Join(t.TempDir(), "x"))
	ann := guestPlayer("ann-browser")
	settings := Settings{Mods: true, Sync: true, Fog: true, Bot: "rex"}
	settings.normalize()
	m := NewHub(store).CreateBetween(ann, botPlayer(bots.Rex), settings)
	c := newClient(ann.key)
	m.join(c, ann)
	waitFor(t, m, "the bot to send its move", func() bool { return m.pending[black] != "" })
	m.handle(c, inMsg{Type: "move", UCI: "e2e4"})
	waitFor(t, m, "the bot's second move", func() bool { return len(m.game.History()) == 2 && m.pending[black] != "" })
}
