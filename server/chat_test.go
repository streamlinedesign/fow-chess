package main

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

// chatLines returns the chat a client would show after reading its queued messages.
func chatLines(t *testing.T, c *client, lines []chatLine) []chatLine {
	t.Helper()
	for _, b := range c.drain() {
		var msg chatMsg
		if json.Unmarshal(b, &msg) != nil || msg.Type != "chat" {
			continue
		}
		if msg.Reset {
			lines = nil
		}
		lines = append(lines, msg.Lines...)
	}
	return lines
}

func TestChat(t *testing.T) {
	hub := NewHub(openTestStore(t, filepath.Join(t.TempDir(), "chess.db")))
	alice, bob, carol := guestPlayer("alice"), guestPlayer("bob"), guestPlayer("carol")
	m := hub.Create(alice, Settings{})
	cw, cb, cs := newClient(alice.key), newClient(bob.key), newClient(carol.key)
	m.join(cw, alice)
	m.join(cb, bob)
	m.join(cs, carol)

	m.handle(cw, inMsg{Type: "chat", Text: "  good luck  "})
	m.handle(cb, inMsg{Type: "chat", Text: "   "})
	m.handle(cb, inMsg{Type: "chat", Text: strings.Repeat("é", chatMaxLen+10)})
	m.handle(cs, inMsg{Type: "chat", Text: "spectators can't talk"})

	var wl, bl []chatLine
	wl, bl = chatLines(t, cw, wl), chatLines(t, cb, bl)
	if len(wl) != 2 || wl[0].Text != "good luck" || wl[0].From != "w" || wl[1].From != "b" {
		t.Fatalf("white sees %+v", wl)
	}
	if len([]rune(wl[1].Text)) != chatMaxLen {
		t.Errorf("long message kept %d characters, want %d", len([]rune(wl[1].Text)), chatMaxLen)
	}
	if len(bl) != 2 {
		t.Errorf("black sees %d lines, want 2", len(bl))
	}
	if sl := chatLines(t, cs, nil); len(sl) != 0 {
		t.Errorf("spectator sees chat %+v", sl)
	}

	// A player who reconnects gets the chat so far.
	m.leave(cb)
	cb = newClient(bob.key)
	m.join(cb, bob)
	if bl = chatLines(t, cb, nil); len(bl) != 2 {
		t.Errorf("after reconnecting black sees %d lines, want 2", len(bl))
	}

	// The chat lasts through the end of the game, and a rematch starts it afresh.
	m.handle(cw, inMsg{Type: "move", UCI: "e2e4"})
	m.handle(cb, inMsg{Type: "resign"})
	m.handle(cw, inMsg{Type: "chat", Text: "gg"})
	if wl = chatLines(t, cw, wl); len(wl) != 3 {
		t.Fatalf("after the game white sees %d lines, want 3", len(wl))
	}
	m.handle(cw, inMsg{Type: "rematch"})
	m.handle(cb, inMsg{Type: "rematch"})
	if wl = chatLines(t, cw, wl); len(wl) != 0 || len(m.chat) != 0 {
		t.Errorf("rematch kept the chat: %+v", wl)
	}

	// Once a finished game has no players left, its chat is gone.
	m.handle(cb, inMsg{Type: "chat", Text: "again?"}) // black is white now
	m.handle(cb, inMsg{Type: "move", UCI: "e2e4"})
	m.handle(cw, inMsg{Type: "resign"})
	m.leave(cw)
	if len(m.chat) == 0 {
		t.Fatal("chat cleared while a player is still here")
	}
	m.leave(cb)
	if len(m.chat) != 0 {
		t.Errorf("chat kept after both players left a finished game: %+v", m.chat)
	}
}
