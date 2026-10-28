package main

import (
	"path/filepath"
	"testing"
	"time"
)

// startGame seats two guests in a new match with the given clock.
func startGame(t *testing.T, store *Store, tc TimeControl) (m *Match, cw, cb *client) {
	t.Helper()
	alice, bob := guestPlayer("alice"), guestPlayer("bob")
	m = NewHub(store).Create(alice, Settings{Time: tc})
	cw, cb = newClient(alice.key), newClient(bob.key)
	m.join(cw, alice)
	m.join(cb, bob)
	return m, cw, cb
}

// rewind pretends the side to move has been thinking for d already.
func rewind(m *Match, d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.turnStart = m.turnStart.Add(-d)
	m.armTimer()
}

func near(a, b time.Duration) bool { return (a - b).Abs() < 200*time.Millisecond }

func TestClockRunsAndAddsIncrement(t *testing.T) {
	m, cw, cb := startGame(t, openTestStore(t, filepath.Join(t.TempDir(), "x")), TimeControl{Base: 180, Inc: 2})
	if !m.running() || m.game.Turn().String() != "w" {
		t.Fatal("white's clock should start when black sits down")
	}
	rewind(m, 10*time.Second)
	m.handle(cw, inMsg{Type: "move", UCI: "e2e4"})
	if !near(m.clock[white], 172*time.Second) {
		t.Errorf("white has %v after 10s with +2, want 2m52s", m.clock[white])
	}
	rewind(m, 30*time.Second)
	if !near(m.left(black, time.Now()), 150*time.Second) {
		t.Errorf("black has %v while thinking 30s, want 2m30s", m.left(black, time.Now()))
	}
	m.handle(cb, inMsg{Type: "resign"})
	if m.running() || !near(m.clock[black], 150*time.Second) {
		t.Errorf("after resigning, running %v and black has %v", m.running(), m.clock[black])
	}
}

func TestClockWaitsForOpponent(t *testing.T) {
	alice := guestPlayer("alice")
	m := NewHub(openTestStore(t, filepath.Join(t.TempDir(), "x"))).Create(alice, Settings{Time: TimeControl{Base: 60}})
	m.join(newClient(alice.key), alice)
	if m.running() {
		t.Error("clock running with nobody to play")
	}
}

func TestFlagFalls(t *testing.T) {
	m, cw, cb := startGame(t, openTestStore(t, filepath.Join(t.TempDir(), "x")), TimeControl{Base: 60})
	m.handle(cw, inMsg{Type: "move", UCI: "e2e4"})
	rewind(m, 61*time.Second)
	m.handle(cb, inMsg{Type: "move", UCI: "e7e5"}) // too late
	if !m.game.Over() || m.game.Outcome().String() != "1-0" || m.game.Method().String() != "time" {
		t.Fatalf("got %s by %q", m.game.Outcome(), m.game.Method())
	}
	if len(m.game.History()) != 1 || m.clock[black] != 0 {
		t.Errorf("late move played, or black left with %v", m.clock[black])
	}
}

func TestTimerEndsGame(t *testing.T) {
	m, cw, _ := startGame(t, openTestStore(t, filepath.Join(t.TempDir(), "x")), TimeControl{Base: 60})
	m.handle(cw, inMsg{Type: "move", UCI: "e2e4"})
	rewind(m, 60*time.Second-50*time.Millisecond)
	time.Sleep(300 * time.Millisecond)
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.game.Over() || m.game.Method().String() != "time" {
		t.Errorf("nobody moved and the game is %s by %q", m.game.Outcome(), m.game.Method())
	}
}

func TestDailyResetsEveryMove(t *testing.T) {
	m, cw, cb := startGame(t, openTestStore(t, filepath.Join(t.TempDir(), "x")), TimeControl{PerMove: 86400})
	rewind(m, 20*time.Hour)
	m.handle(cw, inMsg{Type: "move", UCI: "e2e4"})
	rewind(m, 23*time.Hour)
	m.handle(cb, inMsg{Type: "move", UCI: "e7e5"})
	if m.game.Over() || !near(m.left(white, time.Now()), 24*time.Hour) {
		t.Errorf("white has %v, want a fresh 24h", m.left(white, time.Now()))
	}
}

func TestClockSurvivesRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x")
	store := openTestStore(t, path)
	m, cw, _ := startGame(t, store, TimeControl{Base: 300})
	m.handle(cw, inMsg{Type: "move", UCI: "e2e4"})
	rewind(m, 100*time.Second)
	m.mu.Lock()
	m.save()
	m.stopTimer()
	m.mu.Unlock()
	store.Close()

	store = openTestStore(t, path)
	got := NewHub(store).Get(m.id)
	if got == nil || !got.running() || !near(got.left(black, time.Now()), 200*time.Second) {
		t.Fatalf("restored clock %v, want black on 3m20s", got.left(black, time.Now()))
	}
	got.mu.Lock()
	got.stopTimer()
	got.mu.Unlock()
}

func TestGameRunsOutWhileServerIsDown(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x")
	store := openTestStore(t, path)
	m, cw, _ := startGame(t, store, TimeControl{Base: 60})
	m.handle(cw, inMsg{Type: "move", UCI: "e2e4"})
	m.mu.Lock()
	m.turnStart = m.turnStart.Add(-2 * time.Minute)
	m.save()
	m.stopTimer()
	m.mu.Unlock()
	store.Close()

	store = openTestStore(t, path)
	ids, err := store.ExpiredGames()
	if err != nil || len(ids) != 1 || ids[0] != m.id {
		t.Fatalf("expired games %v, %v", ids, err)
	}
	got := NewHub(store).Get(m.id)
	if !got.game.Over() || got.game.Outcome().String() != "1-0" {
		t.Errorf("restored game is %s by %q", got.game.Outcome(), got.game.Method())
	}
	store.Close()
	store = openTestStore(t, path)
	if ids, _ := store.ExpiredGames(); len(ids) != 0 {
		t.Errorf("still expired after flagging: %v", ids)
	}
	if rec, _ := store.LatestGame(m.id); rec.Reason != "time" {
		t.Errorf("saved reason %q", rec.Reason)
	}
}

func TestRating(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x")
	store := openTestStore(t, path)
	ann, _ := store.CreateUser("ann", []byte("x"))
	ben, _ := store.CreateUser("ben", []byte("x"))
	hub := NewHub(store)
	pa, pb := userPlayer(ann, "a"), userPlayer(ben, "b")

	play := func(tc TimeControl) *Match {
		m := hub.Create(pa, Settings{Time: tc})
		cw, cb := newClient(pa.key), newClient(pb.key)
		m.join(cw, pa)
		m.join(cb, pb)
		m.handle(cw, inMsg{Type: "move", UCI: "e2e4"})
		m.handle(cb, inMsg{Type: "resign"})
		m.mu.Lock()
		m.save() // saving a finished game again must not rate it twice
		m.mu.Unlock()
		return m
	}
	rated := play(TimeControl{Base: 300})
	unrated := play(TimeControl{})
	if !rated.rated() || unrated.rated() {
		t.Fatalf("rated %v, no clock rated %v", rated.rated(), unrated.rated())
	}
	store.Close()

	store = openTestStore(t, path)
	ann, _ = store.UserByID(ann.ID)
	ben, _ = store.UserByID(ben.ID)
	if ann.Rating <= 1500 || ben.Rating >= 1500 || int(ann.Rating+ben.Rating+0.5) != 3000 {
		t.Errorf("after one win ann %.1f, ben %.1f", ann.Rating, ben.Rating)
	}
	rec, _ := store.Game(rated.id, 1)
	if rec.RatingDiff[white] == nil || *rec.RatingDiff[white] <= 0 || *rec.RatingDiff[black] >= 0 {
		t.Errorf("rating diffs %v", rec.RatingDiff)
	}
	if rec, _ := store.Game(unrated.id, 1); rec.RatingDiff[white] != nil {
		t.Error("No clock game was rated")
	}

	top, _ := store.Leaderboard(100)
	if len(top) != 2 || top[0].Username != "ann" || top[0].Games != 1 || top[1].Rank != 2 {
		t.Errorf("leaderboard %+v", top)
	}
	if me, _ := store.Rank(ben.ID); me == nil || me.Rank != 2 {
		t.Errorf("ben's rank %+v", me)
	}
}
