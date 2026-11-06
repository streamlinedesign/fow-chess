package main

import (
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestPrefsAndAvatars(t *testing.T) {
	site := newTestSite(t)
	ann := site.signup("ann")
	site.signup("ben")

	// Everyone starts with the classic set on both sides and the pawn.
	_, me := ann.do("GET", "/api/me", "")
	if me["white"] != "classic" || me["black"] != "classic" || me["side"] != "" || me["avatar"] != "" {
		t.Fatalf("defaults: %v", me)
	}

	for _, bad := range []string{
		`{"white":"gold"}`,
		`{"black":"red-ink"}`,
		`{"side":"red"}`,
		`{"avatar":"classic/K"}`,
		`{"avatar":"red/Z"}`,
		`{"avatar":"red"}`,
		`{"avatar":"galaxy/K"}`,
		`{"lightTiles":"marble"}`,
	} {
		if code, _ := ann.do("PUT", "/api/prefs", bad); code != 400 {
			t.Errorf("%s: %d", bad, code)
		}
	}

	if code, _ := ann.do("PUT", "/api/prefs", `{"white":"red","black":"red","side":"b","avatar":"violet-light/N"}`); code != 200 {
		t.Fatalf("save: %d", code)
	}
	_, me = ann.do("GET", "/api/me", "")
	if me["white"] != "red" || me["black"] != "red" || me["side"] != "b" || me["avatar"] != "violet-light/N" {
		t.Errorf("saved: %v", me)
	}

	// Themes are locked until you beat the bot that wears them (holy: until
	// you reach a 2000 rating).
	themes := `{"white":"holy","black":"galaxy","lightTiles":"holy","darkTiles":"galaxy","plain":true,"avatar":"ancient-dark/N"}`
	if code, _ := ann.do("PUT", "/api/prefs", themes); code != 403 {
		t.Fatalf("locked themes: %d", code)
	}
	for _, reason := range []string{lockRating(2000), "bot:solomon", "bot:ramesses"} {
		if err := site.store.Unlock(ann.id(t), reason); err != nil {
			t.Fatal(err)
		}
	}

	// Themes, tiles per square colour, and no skins at all.
	if code, _ := ann.do("PUT", "/api/prefs", `{"white":"holy","black":"galaxy","lightTiles":"holy","darkTiles":"galaxy","plain":true,"avatar":"ancient-dark/N"}`); code != 200 {
		t.Fatalf("themes: %d", code)
	}
	_, me = ann.do("GET", "/api/me", "")
	if me["white"] != "holy" || me["black"] != "galaxy" || me["lightTiles"] != "holy" || me["plain"] != true || me["avatar"] != "ancient-dark/N" {
		t.Errorf("themes saved: %v", me)
	}
	ann.do("PUT", "/api/prefs", `{"white":"red","black":"blue","side":"b","avatar":"violet-ink/N"}`)

	// Anyone can see avatars; players on the pawn are left out.
	_, got := ann.do("GET", "/api/avatars?u=ANN,ben,nobody", "")
	if len(got) != 1 || got["ann"] != "violet-ink/N" {
		t.Errorf("avatars: %v", got)
	}
}

// Choices saved before White and Black sets carry over: the set you had for
// your own pieces becomes both.
func TestOldPrefsCarryOver(t *testing.T) {
	p := prefs{Primary: "galaxy-dark"}
	p.fill()
	if p.White != "galaxy" || p.Black != "galaxy" || p.Primary != "" {
		t.Errorf("galaxy-dark: %+v", p)
	}
	p = prefs{Primary: "ink"}
	p.fill()
	if p.White != "classic" || p.Black != "classic" {
		t.Errorf("ink: %+v", p)
	}
	p = prefs{Primary: "blue-ink", skin: skin{White: "holy"}}
	p.fill()
	if p.White != "holy" || p.Black != "classic" {
		t.Errorf("new choices win: %+v", p)
	}
}

// Each player's skin goes to everyone in the match, follows them into the
// rematch, and a plain or unknown one is dropped.
func TestMatchSkins(t *testing.T) {
	hub := NewHub(openTestStore(t, filepath.Join(t.TempDir(), "chess.db")))
	alice, bob, carol := guestPlayer("alice"), guestPlayer("bob"), guestPlayer("carol")
	m := hub.Create(alice, Settings{})
	cw, cb, cs := newClient(alice.key), newClient(bob.key), newClient(carol.key)
	m.join(cw, alice)
	m.join(cb, bob)
	m.join(cs, carol)

	seen := func(c *client) map[string]seatState {
		t.Helper()
		var st stateMsg
		msgs := c.drain()
		for _, b := range msgs {
			var probe struct{ Type string }
			json.Unmarshal(b, &probe)
			if probe.Type == "state" {
				json.Unmarshal(b, &st)
			}
		}
		return st.Players
	}
	// A guest hasn't beaten any bot: the galaxy and holy parts are dropped.
	m.handle(cw, inMsg{Type: "skin", Skin: &skin{White: "galaxy", Black: "red", LightTiles: "holy"}})
	if p := seen(cs); p["w"].Skin == nil || p["w"].Skin.White != "classic" || p["w"].Skin.LightTiles != "" || p["w"].Skin.Black != "red" {
		t.Errorf("locked parts kept: %+v", p["w"].Skin)
	}
	m.handle(cw, inMsg{Type: "skin", Skin: &skin{White: "blue", Black: "red"}})
	m.handle(cs, inMsg{Type: "skin", Skin: &skin{White: "green", Black: "green"}}) // spectators have no seat
	m.handle(cb, inMsg{Type: "skin", Skin: &skin{White: "gold", Black: "red"}})
	p := seen(cs)
	if s := p["w"].Skin; s == nil || s.White != "blue" || s.Black != "red" {
		t.Errorf("white's skin: %+v", s)
	}
	if p["b"].Skin != nil {
		t.Errorf("black's unknown set kept: %+v", p["b"].Skin)
	}

	m.handle(cb, inMsg{Type: "skin", Skin: &skin{White: "classic", Black: "violet"}})
	m.handle(cw, inMsg{Type: "move", UCI: "e2e4"})
	m.handle(cb, inMsg{Type: "resign"})
	m.handle(cw, inMsg{Type: "rematch"})
	m.handle(cb, inMsg{Type: "rematch"})
	p = seen(cb)
	if s := p["w"].Skin; s == nil || s.Black != "violet" {
		t.Errorf("after the rematch bob (white) has %+v", s)
	}
	if s := p["b"].Skin; s == nil || s.White != "blue" {
		t.Errorf("after the rematch alice (black) has %+v", s)
	}

	m.handle(cw, inMsg{Type: "skin", Skin: &skin{White: "classic", Black: "classic"}})
	if p = seen(cs); p["b"].Skin != nil {
		t.Errorf("plain skin kept: %+v", p["b"].Skin)
	}
}
