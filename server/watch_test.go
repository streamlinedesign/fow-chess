package main

import (
	"encoding/json"
	"testing"
)

// lastState is the newest state a client was sent.
func lastState(t *testing.T, c *client) map[string]any {
	t.Helper()
	c.mu.Lock()
	b := c.latest
	c.mu.Unlock()
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("state %q: %v", b, err)
	}
	return out
}

func TestWatching(t *testing.T) {
	site := newTestSite(t)
	ann, ben, cat := site.signup("ann"), site.signup("ben"), site.signup("cat")
	ann.do("POST", "/api/friends/ben", "")
	ben.do("POST", "/api/friends/ann", "")
	cat.do("POST", "/api/friends/ann", "")
	ann.do("POST", "/api/friends/cat", "")
	_, res := ann.do("POST", "/api/offers", `{"to":"ben","settings":{"fog":true},"color":"w"}`)
	_, res = ben.do("POST", "/api/offers/"+jsonNumber(res["id"].(float64))+"/accept", "")
	id := res["matchId"].(string)
	m := site.hub.Get(id)
	m.mu.Lock()
	seats, users := m.seats, m.users
	m.mu.Unlock()
	cw, cb := newClient(seats[white]), newClient(seats[black])
	m.join(cw, &player{key: seats[white], userID: users[white]})
	if got := site.hub.Playing(ann.id(t)); got != "" {
		t.Errorf("ann is playing %q before ben is there", got)
	}
	m.join(cb, &player{key: seats[black], userID: users[black]})
	if got := site.hub.Playing(ann.id(t)); got != id {
		t.Errorf("ann is playing %q, want %s", got, id)
	}

	// Two guests playing a game with no fog.
	other := site.hub.Create(guestPlayer("g1"), Settings{})
	other.join(newClient("g1"), guestPlayer("g1"))
	other.join(newClient("g2"), guestPlayer("g2"))

	for i, uci := range []string{"e2e4", "e7e5", "g1f3", "b8c6", "f1c4", "g8f6", "d2d3", "f8c5"} {
		c := cw
		if i%2 == 1 {
			c = cb
		}
		m.handle(c, inMsg{Type: "move", UCI: uci})
	}

	// A spectator sees the game as white saw it six moves ago, and can switch sides.
	spec := newClient("someone")
	m.join(spec, guestPlayer("someone"))
	s := lastState(t, spec)
	if s["watching"] != "w" || s["behind"] != 6.0 || s["ply"] != 8.0 || len(s["history"].([]any)) != 2 || s["spectators"] != 1.0 {
		t.Errorf("spectator state: watching %v behind %v ply %v history %v spectators %v",
			s["watching"], s["behind"], s["ply"], s["history"], s["spectators"])
	}
	m.handle(spec, inMsg{Type: "watch", Side: "b"})
	if s := lastState(t, spec); s["watching"] != "b" || s["history"].([]any)[0] != "?" {
		t.Errorf("after switching: watching %v history %v", s["watching"], s["history"])
	}
	// Players still see their own game as it is, and know someone is watching.
	if s := lastState(t, cw); s["you"] != "w" || s["behind"] != nil || s["spectators"] != 1.0 {
		t.Errorf("white's state: you %v behind %v spectators %v", s["you"], s["behind"], s["spectators"])
	}

	// The Watch page puts a friend's game first, and leaves out your own.
	_, res = cat.do("GET", "/api/watch", "")
	games := res["games"].([]any)
	if len(games) != 2 {
		t.Fatalf("cat sees %d games: %v", len(games), games)
	}
	first := games[0].(map[string]any)
	players := first["players"].(map[string]any)
	if first["id"] != id || first["friend"] != true || players["w"].(map[string]any)["name"] != "ann" ||
		players["w"].(map[string]any)["rating"] != 1500.0 || first["ply"] != 8.0 || first["spectators"] != 1.0 {
		t.Errorf("cat's first game: %v", first)
	}
	if second := games[1].(map[string]any); second["friend"] != false || second["players"].(map[string]any)["w"].(map[string]any)["name"] != nil {
		t.Errorf("cat's second game: %v", second)
	}
	_, res = ann.do("GET", "/api/watch", "")
	if games := res["games"].([]any); len(games) != 1 || games[0].(map[string]any)["id"] == id {
		t.Errorf("ann's watch list has her own game: %v", games)
	}

	// Once both players leave, it isn't being played right now.
	m.leave(cw)
	m.leave(cb)
	if got := site.hub.Playing(ann.id(t)); got != "" {
		t.Errorf("ann is still playing %q after leaving", got)
	}
	if _, res = cat.do("GET", "/api/watch", ""); len(res["games"].([]any)) != 1 {
		t.Errorf("cat still sees the left game: %v", res["games"])
	}
}
