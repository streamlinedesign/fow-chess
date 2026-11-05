package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"slices"
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

func TestFourSyncRounds(t *testing.T) {
	hub := NewFourHub(nil)
	ann := guestPlayer("ann-browser")
	m := hub.Create(ann, fourSettings{Fog: true, Sync: true, Time: TimeControl{Base: 60, Inc: 2}})
	_, cs := fourPlayers(m)
	for i, c := range cs {
		s := lastFour(t, c)
		if len(s.Legal) == 0 || len(s.Ticking) != 4 {
			t.Fatalf("player %d: %d legal moves, ticking %v", i, len(s.Legal), s.Ticking)
		}
	}
	m.handle(cs[0], fourIn{Type: "move", UCI: "e2e4"})
	m.handle(cs[0], fourIn{Type: "move", UCI: "f2f4"}) // too late: the first move is locked in
	m.handle(cs[1], fourIn{Type: "move", UCI: "b7d7"})
	m.handle(cs[2], fourIn{Type: "move", UCI: "e2e4"}) // not Green's piece: ignored
	s := lastFour(t, cs[0])
	if s.Planned != "e2e4" || len(s.Legal) != 0 || !s.Players["b"].Ready || s.Players["g"].Ready || s.Ply != 0 {
		t.Fatalf("planned %q ready %+v ply %d", s.Planned, s.Players, s.Ply)
	}
	if slices.Contains(s.Ticking, "r") || !slices.Contains(s.Ticking, "g") {
		t.Errorf("ticking %v", s.Ticking)
	}
	m.handle(cs[2], fourIn{Type: "move", UCI: "g13g11"})
	m.handle(cs[3], fourIn{Type: "move", UCI: "m8k8"})
	s = lastFour(t, cs[0])
	if s.Ply != 4 || s.Planned != "" || s.Players["b"].Ready || len(s.Ticking) != 4 {
		t.Fatalf("after the round: ply %d planned %q ticking %v", s.Ply, s.Planned, s.Ticking)
	}
	if s.History[0].SAN != "e2-e4" || s.History[1].SAN != "?" || s.History[1].Round != 1 {
		t.Errorf("history %+v", s.History)
	}
	if s.Clock.Left["r"] < 61000 {
		t.Errorf("no increment: %d", s.Clock.Left["r"])
	}
	if m.game.Tokens() != "s:e2e4,b7d7,g13g11,m8k8" {
		t.Errorf("tokens %q", m.game.Tokens())
	}
	// Violet runs out while the others have sent theirs: the round is played without Violet.
	m.handle(cs[0], fourIn{Type: "move", UCI: "f2f3"})
	m.handle(cs[1], fourIn{Type: "move", UCI: "b8c8"})
	m.handle(cs[2], fourIn{Type: "move", UCI: "h13h12"})
	m.mu.Lock()
	m.turnStart = time.Now().Add(-90 * time.Second)
	m.mu.Unlock()
	m.checkFlag()
	s = lastFour(t, cs[0])
	if s.Players["v"].Out != "time" || s.Ply != 8 || s.Clock.Left["r"] < 60000 {
		t.Fatalf("violet out %q ply %d left %d", s.Players["v"].Out, s.Ply, s.Clock.Left["r"])
	}
	// A spectator of a fog game is two rounds behind.
	sp := newClient("watcher")
	m.join(sp, guestPlayer("watcher"))
	if w := lastFour(t, sp); w.Behind != 4 { // Violet's timeout and the last round
		t.Errorf("spectator behind %d", w.Behind)
	}
}

func TestFourSyncBots(t *testing.T) {
	hub := NewFourHub(nil)
	hub.botPause = func() time.Duration { return 0 }
	ann := guestPlayer("ann-browser")
	m := hub.Create(ann, fourSettings{Sync: true, Hill: true})
	c := newClient(ann.key)
	m.join(c, ann)
	m.handle(c, fourIn{Type: "fill"})
	m.handle(c, fourIn{Type: "resign"})
	// Bots dodge each other for a long time in Sync Move; it's enough that rounds go by.
	deadline := time.Now().Add(20 * time.Second)
	for {
		m.mu.Lock()
		over, n := m.game.Over(), len(m.game.History())
		m.mu.Unlock()
		if over || n >= 3*30 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("bots stopped after %d moves", n)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestFourSetup(t *testing.T) {
	store := openTestStore(t, filepath.Join(t.TempDir(), "four.db"))
	hub := NewFourHub(store)
	ann := guestPlayer("ann-browser")
	m := hub.Create(ann, fourSettings{Setup: "backRow", Time: TimeControl{Base: 60}})
	_, cs := fourPlayers(m)
	s := lastFour(t, cs[0])
	if s.Status != "setup" || s.Clock.Running != "" || len(s.Legal) != 0 || slices.Contains(s.Board, "rK") {
		t.Fatalf("status %s running %q legal %d", s.Status, s.Clock.Running, len(s.Legal))
	}
	mine := "KRRQNNBB" + "PPPPPPPP" + "........"
	m.handle(cs[0], fourIn{Type: "setup", Board: "KKRQNNBB" + "PPPPPPPP" + "........"}) // refused
	if lastFour(t, cs[0]).Arranged != "" {
		t.Fatal("bad setup taken")
	}
	m.handle(cs[0], fourIn{Type: "setup", Board: mine})
	m.handle(cs[1], fourIn{Type: "setup", Board: four.StandardSetup})
	m.handle(cs[2], fourIn{Type: "resign"})
	s = lastFour(t, cs[0])
	if s.Arranged != mine || !s.Players["b"].Ready || s.Players["v"].Ready || s.Players["g"].Out != "resignation" {
		t.Fatalf("arranged %q players %+v", s.Arranged, s.Players)
	}
	if lastFour(t, cs[1]).Arranged != four.StandardSetup {
		t.Fatal("blue's arrangement")
	}
	hub.flush()
	if again := NewFourHub(store).Get(m.id); again == nil || again.start != m.start || !again.arranging() {
		t.Fatalf("arranging not restored")
	}
	m.handle(cs[3], fourIn{Type: "setup", Board: four.StandardSetup})
	s = lastFour(t, cs[0])
	if s.Status != "playing" || s.Board[0*14+3] != "rK" || s.Clock.Running != "r" || s.Players["g"].Out != "resignation" {
		t.Fatalf("status %s d1 %s running %q", s.Status, s.Board[3], s.Clock.Running)
	}
	m.handle(cs[0], fourIn{Type: "move", UCI: "e2e4"})
	hub.flush()
	again := NewFourHub(store).Get(m.id)
	if again == nil || again.game.Tokens() != m.game.Tokens() || again.game.Position().Board != m.game.Position().Board {
		t.Fatalf("game with a setup not restored")
	}
}

func TestFourSetupBots(t *testing.T) {
	hub := NewFourHub(nil)
	ann := guestPlayer("ann-browser")
	m := hub.Create(ann, fourSettings{Setup: "battleship"})
	c := newClient(ann.key)
	m.join(c, ann)
	m.handle(c, fourIn{Type: "fill"})
	if s := lastFour(t, c); s.Status != "setup" || !s.Players["b"].Ready || s.Players["r"].Ready {
		t.Fatalf("status %s players %+v", s.Status, s.Players)
	}
	m.handle(c, fourIn{Type: "setup", Board: "PPPPPPPP" + "RRBBNN.." + "...KQ..."})
	if s := lastFour(t, c); s.Status != "playing" || len(s.Legal) == 0 {
		t.Fatalf("status %s", s.Status)
	}
}
