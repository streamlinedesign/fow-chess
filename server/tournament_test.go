package main

import (
	"path/filepath"
	"testing"
	"time"

	"chess/engine"
)

func TestArenaLifecycle(t *testing.T) {
	store := openTestStore(t, filepath.Join(t.TempDir(), "x"))
	hub := NewHub(store)
	live := NewLive(store)
	var notes []notification
	tm := NewTournaments(store, hub, live, func(n notification) { notes = append(notes, n) })
	tm.online = func(int64) bool { return true }
	// A quiet Tuesday evening, just before the rapid arena.
	now := time.Date(2026, 11, 3, 18, 59, 0, 0, arenaZone)
	tm.now = func() time.Time { return now }

	tm.tick()
	upcoming, _ := store.UpcomingTournaments()
	if len(upcoming) != 2*(arenaDaysAhead+1) {
		t.Fatalf("after the first tick, %d arenas on the calendar, want %d", len(upcoming), 2*(arenaDaysAhead+1))
	}
	rapid, blitz := upcoming[0], upcoming[1]
	if rapid.Kind != "rapid" || !rapid.StartsAt.Equal(time.Date(2026, 11, 3, 19, 0, 0, 0, arenaZone)) || blitz.Kind != "blitz" {
		t.Fatalf("first two arenas: %+v, %+v", rapid, blitz)
	}

	var users []*User
	for _, name := range []string{"ann", "ben", "cat", "dan"} {
		u, err := store.CreateUser(name, []byte("x"))
		if err != nil {
			t.Fatal(err)
		}
		users = append(users, u)
	}
	// Two join the blitz, four the rapid.
	for _, u := range users[:2] {
		if err := tm.join(&blitz, u); err != nil {
			t.Fatal(err)
		}
	}
	for _, u := range users {
		if err := tm.join(&rapid, u); err != nil {
			t.Fatal(err)
		}
	}

	// The rapid starts: four players, two games at once.
	now = rapid.StartsAt.Add(time.Second)
	tm.tick()
	row, _ := store.Tournament(rapid.ID)
	if row.Status != "running" {
		t.Fatalf("rapid status %q", row.Status)
	}
	a := tm.arenas[rapid.ID]
	if a == nil || len(a.games) != 2 {
		t.Fatalf("arena games: %v", a)
	}
	for _, g := range a.games {
		m := hub.Get(g.MatchID)
		if m == nil || m.settings.Tournament != rapid.ID || !m.rated() || m.settings.Time != (TimeControl{Base: 600}) {
			t.Fatalf("arena game %s: %+v", g.MatchID, m.settings)
		}
	}

	// A game ends: the winner scores 2, both are free and get paired again.
	var first tournamentGame
	for _, g := range a.games {
		first = g
		break
	}
	tm.gameOver(gameResult{MatchID: first.MatchID, Round: 1, Settings: Settings{Tournament: rapid.ID}, Outcome: engine.WhiteWins})
	if p := a.players[first.White]; p.score != winPoints || p.games != 1 || p.wins != 1 {
		t.Errorf("winner: %+v", p)
	}
	if p := a.players[first.Black]; p.score != 0 || p.games != 1 || !p.busy {
		t.Errorf("loser: %+v (should be paired again at once)", p)
	}
	if len(a.games) != 2 {
		t.Errorf("after a result, %d games in play, want 2 (the pair played again)", len(a.games))
	}
	standings := tm.standingsOf(rapid.ID)
	if standings[0].UserID != first.White || standings[0].Score != winPoints {
		t.Errorf("standings: %+v", standings)
	}

	// Sitting out keeps you off the pairing list; your points stay.
	loser := a.players[first.Black].user
	if err := tm.leave(&rapid, loser); err != nil {
		t.Fatal(err)
	}
	if !a.players[loser.ID].paused {
		t.Error("leaving a running arena should sit the player out")
	}

	// The rapid ends: finished, and everyone learns their place.
	now = rapid.EndsAt.Add(time.Second)
	notes = nil
	tm.tick()
	row, _ = store.Tournament(rapid.ID)
	if row.Status != "finished" || tm.arenas[rapid.ID] != nil {
		t.Fatalf("rapid status %q", row.Status)
	}
	if len(notes) != 4 || notes[0].Kind != notifyArenaResult || notes[0].UserID != first.White || notes[0].Data["place"] != 1 {
		t.Fatalf("result notes: %+v", notes)
	}
	// The blitz starts with two of four: called off, and both hear about it.
	now = blitz.StartsAt.Add(time.Second)
	notes = nil
	tm.tick()
	row, _ = store.Tournament(blitz.ID)
	if row.Status != "cancelled" || len(notes) != 2 || notes[0].Kind != notifyArenaCancelled {
		t.Fatalf("blitz status %q, notes %+v", row.Status, notes)
	}

	recent, _ := store.RecentTournaments(5)
	if len(recent) != 1 || recent[0].ID != rapid.ID || recent[0].Players != 4 {
		t.Errorf("recent: %+v", recent)
	}
}
