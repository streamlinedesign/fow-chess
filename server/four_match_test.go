package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"chess/four"
)

// lastFour is the newest state a client has been sent.
func lastFour(t *testing.T, c *client) fourState {
	t.Helper()
	var s fourState
	msgs := c.drain()
	for i := len(msgs) - 1; i >= 0; i-- {
		if json.Unmarshal(msgs[i], &s) == nil && s.Type == "state" {
			return s
		}
	}
	t.Fatal("no state sent")
	return s
}

func fourPlayers(m *FourMatch) (ps []*player, cs []*client) {
	for _, name := range []string{"ann", "ben", "cat", "dan"} {
		p := guestPlayer(name + "-browser")
		c := newClient(p.key)
		m.join(c, p)
		ps, cs = append(ps, p), append(cs, c)
	}
	return ps, cs
}

func TestFourSeatsAndTurns(t *testing.T) {
	hub := NewFourHub(nil)
	ann := guestPlayer("ann-browser")
	m := hub.Create(ann, fourSettings{Fog: true, CenterVision: "any"})
	_, cs := fourPlayers(m)
	for i, c := range cs {
		s := lastFour(t, c)
		if s.You != four.Colors[i].String() || s.Status != "playing" {
			t.Fatalf("player %d: you %q status %s", i, s.You, s.Status)
		}
		if len(s.Board) != four.NumSquares {
			t.Fatalf("board has %d squares", len(s.Board))
		}
		if (i == 0) != (len(s.Legal) > 0) {
			t.Errorf("player %d has %d legal moves", i, len(s.Legal))
		}
	}
	m.handle(cs[1], fourIn{Type: "move", UCI: "b5d5"}) // not Blue's turn
	m.handle(cs[0], fourIn{Type: "move", UCI: "e2e4"})
	m.handle(cs[1], fourIn{Type: "move", UCI: "b5d5"})
	s := lastFour(t, cs[2])
	if s.Turn != "g" || s.Ply != 2 {
		t.Fatalf("turn %s ply %d", s.Turn, s.Ply)
	}
	if s.History[0].SAN != "?" || s.Last != nil {
		t.Errorf("green should not know the other moves: %+v %+v", s.History, s.Last)
	}
	red := lastFour(t, cs[0])
	if red.History[0].SAN != "e2-e4" || red.Last == nil || red.Last.To != "e4" {
		t.Errorf("red's own move: %+v %+v", red.History, red.Last)
	}
	// A spectator with fog is kept behind.
	sp := newClient("watcher")
	m.join(sp, guestPlayer("watcher"))
	w := lastFour(t, sp)
	if w.You != "" || w.Watching != "r" || w.Behind != 2 {
		t.Errorf("spectator: you %q watching %q behind %d", w.You, w.Watching, w.Behind)
	}
	// Red resigns on Green's turn: Red is out, Green still to move.
	m.handle(cs[0], fourIn{Type: "resign"})
	s = lastFour(t, cs[2])
	if s.Players["r"].Out != "resignation" || s.Turn != "g" {
		t.Errorf("after red resigns: %+v turn %s", s.Players["r"], s.Turn)
	}
	// A player who is out sees everything.
	if r := lastFour(t, cs[0]); r.Board[four.At(6, 12)] == "?" {
		t.Errorf("red is out and should see the whole board")
	}
}

func TestFourBotsFillAndFinish(t *testing.T) {
	hub := NewFourHub(nil)
	hub.botPause = func() time.Duration { return 0 }
	ann := guestPlayer("ann-browser")
	m := hub.Create(ann, fourSettings{Fog: true})
	c := newClient(ann.key)
	m.join(c, ann)
	if s := lastFour(t, c); s.Status != "waiting" {
		t.Fatalf("status %s", s.Status)
	}
	m.handle(c, fourIn{Type: "fill"})
	s := lastFour(t, c)
	if s.Status != "playing" || !s.Players["v"].Bot || s.Players["v"].Name != fourBotName {
		t.Fatalf("after fill: %s %+v", s.Status, s.Players["v"])
	}
	// Ann resigns at once; the three bots play it out.
	m.handle(c, fourIn{Type: "resign"})
	deadline := time.Now().Add(20 * time.Second)
	for {
		m.mu.Lock()
		over := m.game.Over()
		m.mu.Unlock()
		if over {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("bots still playing after 20s")
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func TestFourClock(t *testing.T) {
	hub := NewFourHub(nil)
	ann := guestPlayer("ann-browser")
	m := hub.Create(ann, fourSettings{Time: TimeControl{Base: 60}})
	_, cs := fourPlayers(m)
	m.mu.Lock()
	// Red has run out.
	m.turnStart = time.Now().Add(-61 * time.Second)
	m.mu.Unlock()
	m.handle(cs[0], fourIn{Type: "move", UCI: "e2e4"})
	s := lastFour(t, cs[1])
	if s.Players["r"].Out != "time" || s.Turn != "b" || s.Clock == nil || s.Clock.Running != "b" {
		t.Errorf("red should be out of time: %+v turn %s clock %+v", s.Players["r"], s.Turn, s.Clock)
	}
}

func TestFourSurvivesRestart(t *testing.T) {
	store := openTestStore(t, filepath.Join(t.TempDir(), "four.db"))
	hub := NewFourHub(store)
	ann := guestPlayer("ann-browser")
	m := hub.Create(ann, fourSettings{Fog: true, JesterPawns: true})
	_, cs := fourPlayers(m)
	m.handle(cs[0], fourIn{Type: "move", UCI: "e2e4"})
	m.handle(cs[1], fourIn{Type: "resign"})
	hub.flush()

	again := NewFourHub(store)
	m2 := again.Get(m.id)
	if m2 == nil {
		t.Fatal("match not restored")
	}
	if m2.game.Tokens() != "e2e4 -b" || !m2.settings.JesterPawns || m2.seats != m.seats {
		t.Errorf("restored %q %+v %v", m2.game.Tokens(), m2.settings, m2.seats)
	}
	if again.Get("NOSUCH") != nil {
		t.Errorf("found a match that doesn't exist")
	}
}

func TestFourSettingsValid(t *testing.T) {
	for _, tc := range []TimeControl{{}, {Base: 300}, {Base: 60}, {PerMove: 24 * 60 * 60}} {
		if !(fourSettings{Time: tc}).valid() {
			t.Errorf("%+v should be allowed", tc)
		}
	}
	if (fourSettings{Time: TimeControl{Base: 7}}).valid() || (fourSettings{CenterVision: "x"}).valid() {
		t.Errorf("made-up settings allowed")
	}
}

// The demo game in seed/ must still replay to the result it claims.
func TestFourSeedReplays(t *testing.T) {
	sql, err := os.ReadFile("../seed/002_four_player_game.sql")
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`'((?:[a-n]\d{1,2}[a-n]\d{1,2}[qrbn]? ?)+)',\s*'(\w+)', '([a-z ]+)'`).FindSubmatch(sql)
	if m == nil {
		t.Fatal("no moves in the seed")
	}
	g := four.NewGame(fourSettings{CenterVision: "any"}.rules())
	for _, tok := range strings.Fields(string(m[1])) {
		if err := g.PlayToken(tok); err != nil {
			t.Fatalf("seed move %s: %v", tok, err)
		}
	}
	if g.Result() != string(m[2]) || g.Method().String() != string(m[3]) {
		t.Errorf("seed says %s by %s, replay gives %s by %s", m[2], m[3], g.Result(), g.Method())
	}
}
