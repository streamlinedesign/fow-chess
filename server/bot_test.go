package main

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"chess/bots"
)

// waitFor polls until cond holds under the match lock, or fails.
func waitFor(t *testing.T, m *Match, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		m.mu.Lock()
		ok := cond()
		m.mu.Unlock()
		if ok {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func TestBotPlaysItsSeat(t *testing.T) {
	store := openTestStore(t, filepath.Join(t.TempDir(), "x"))
	hub := NewHub(store)
	ann := guestPlayer("ann-browser")
	bot := botPlayer(bots.Vercingetorix)
	settings := Settings{Fog: true, KingCapture: true, Time: TimeControl{Base: 300}, Bot: "vercingetorix"}
	settings.normalize()
	if !settings.valid() || !settings.Unrated {
		t.Fatalf("bot settings should be valid and unranked: %+v", settings)
	}

	// The bot is White: it moves first, as soon as the human sits down.
	m := hub.CreateBetween(bot, ann, settings)
	c := newClient(ann.key)
	m.join(c, ann)
	waitFor(t, m, "the bot's first move", func() bool { return len(m.game.History()) == 1 })
	m.mu.Lock()
	if !m.online(white) || m.names[white] != "Vercingetorix" || m.rated() {
		t.Errorf("bot seat: online %v name %q rated %v", m.online(white), m.names[white], m.rated())
	}
	m.mu.Unlock()
	m.handle(c, inMsg{Type: "move", UCI: "e7e5"})
	waitFor(t, m, "the bot's reply", func() bool { return len(m.game.History()) == 3 })

	// The bot accepts a rematch and, now Black, waits for the human.
	m.handle(c, inMsg{Type: "resign"})
	m.handle(c, inMsg{Type: "rematch"})
	m.mu.Lock()
	round, turn := m.round, m.game.Turn()
	m.mu.Unlock()
	if round != 2 || turn != 0 || m.colorOf(bot.key) != black {
		t.Fatalf("after rematch: round %d turn %v bot seat %d", round, turn, m.colorOf(bot.key))
	}
	m.handle(c, inMsg{Type: "move", UCI: "d2d4"})
	waitFor(t, m, "the bot's reply as Black", func() bool { return len(m.game.History()) == 2 })
}

func TestBotSetsUpInBattleship(t *testing.T) {
	store := openTestStore(t, filepath.Join(t.TempDir(), "x"))
	hub := NewHub(store)
	ann := guestPlayer("ann-browser")
	settings := Settings{Fog: true, KingCapture: true, Setup: "battleship", Time: TimeControl{Base: 300}, Bot: "rex"}
	m := hub.CreateBetween(ann, botPlayer(bots.Rex), settings)
	c := newClient(ann.key)
	m.join(c, ann)
	m.mu.Lock()
	placed := m.placed[black]
	m.mu.Unlock()
	if err := checkSetup(setupModes["battleship"], black, placed, false); err != nil {
		t.Fatalf("bot setup %q: %v", placed, err)
	}
	m.handle(c, inMsg{Type: "setup", Board: ownSetup(white)})
	waitFor(t, m, "the game to start", func() bool { return m.setUp })
	m.handle(c, inMsg{Type: "move", UCI: "e2e4"})
	waitFor(t, m, "the bot's reply", func() bool { return len(m.game.History()) == 2 })
}

func TestRandomSetupIsValid(t *testing.T) {
	for i := 0; i < 50; i++ {
		for name, mode := range setupModes {
			if mode.rows == 0 {
				continue
			}
			for _, color := range []int{white, black} {
				if err := checkSetup(mode, color, randomSetup(mode, color, true), true); err != nil {
					t.Fatalf("%s for %d: %v", name, color, err)
				}
			}
		}
	}
}

// The demo board's game: the bot you pick (Solomon by default), no clock, never saved, never listed.
func TestDemoGame(t *testing.T) {
	store := openTestStore(t, filepath.Join(t.TempDir(), "x"))
	hub := NewHub(store)
	def := Settings{Demo: true}
	def.normalize()
	if def.Bot != "solomon" {
		t.Errorf("default demo bot: %q", def.Bot)
	}
	bad := Settings{Demo: true, Bot: "deep-blue"}
	bad.normalize()
	if bad.Bot != "solomon" {
		t.Errorf("unknown demo bot became %q", bad.Bot)
	}
	settings := Settings{Demo: true, Time: TimeControl{Base: 300}, Setup: "battleship", Bot: "ramesses"}
	settings.normalize()
	if !settings.valid() || settings.Bot != "ramesses" || settings.Time.clocked() || settings.Setup != "" || !settings.Fog || !settings.Unrated {
		t.Fatalf("demo settings: %+v", settings)
	}
	ann := guestPlayer("ann-browser")
	m := hub.CreateBetween(botPlayer(bots.Ramesses), ann, settings)
	c := newClient(ann.key)
	m.join(c, ann)
	waitFor(t, m, "the bot's first move", func() bool { return len(m.game.History()) == 1 })
	m.mu.Lock()
	live := m.live()
	m.mu.Unlock()
	if live {
		t.Error("a demo game is listed as live")
	}
	time.Sleep(300 * time.Millisecond) // give the store's writer a chance
	if rec, _ := store.LatestGame(m.id); rec != nil {
		t.Error("a demo game was saved")
	}
}

// A bot's Battleship setup keeps its king guarded.
func TestBotSetupGuardsTheKing(t *testing.T) {
	for color := range 2 {
		for range 50 {
			b := botSetup(setupModes["battleship"], color, false)
			k := strings.IndexByte(b, "Kk"[color])
			if row := ownRow(color, k); row > 1 {
				t.Fatalf("king on its own row %d: %s", row, b)
			}
		}
	}
}
