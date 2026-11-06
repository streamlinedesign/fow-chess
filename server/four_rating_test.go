package main

import (
	"path/filepath"
	"testing"
	"time"

	"chess/bots"
	"chess/four"
)

func TestFourQueueGroups(t *testing.T) {
	q := NewFourQueue()
	users := []*User{{ID: 1}, {ID: 2}, {ID: 3}, {ID: 4}, {ID: 5}}
	ratings := []int{1500, 1520, 1480, 1510, 2400}
	for i := range 3 {
		q.join(users[i], ratings[i])
		if g := q.group(users[i]); g != nil {
			t.Fatalf("grouped with %d waiting", i+1)
		}
	}
	q.join(users[4], ratings[4])
	if g := q.group(users[4]); g != nil {
		t.Fatal("a far rating grouped at once")
	}
	q.join(users[3], ratings[3])
	g := q.group(users[3])
	if len(g) != 4 {
		t.Fatalf("group of %d", len(g))
	}
	for _, e := range g {
		if e.user.ID == 5 {
			t.Error("the far rating was picked over a close one")
		}
	}
	if q.Size() != 1 {
		t.Errorf("%d left waiting", q.Size())
	}
}

func TestFourRanked(t *testing.T) {
	store := openTestStore(t, filepath.Join(t.TempDir(), "four.db"))
	hub := NewFourHub(store)
	var ps [4]*player
	var cs [4]*client
	for i, name := range []string{"ann", "ben", "cat", "dan"} {
		u, err := store.CreateUser(name, []byte("x"))
		if err != nil {
			t.Fatal(err)
		}
		ps[i] = userPlayer(u, "")
	}
	m := hub.CreateRanked(ps, [4]int{1500, 1500, 1500, 1500})
	for i, p := range ps {
		cs[i] = newClient(p.key)
		m.join(cs[i], p)
	}
	s := lastFour(t, cs[0])
	if s.Status != "setup" || !s.Settings.Rated || s.Settings.Board != "extended" || !s.Settings.Fog || s.Settings.CenterVision != "any" || s.Settings.Sync || !s.Settings.JesterPawns {
		t.Fatalf("status %s settings %+v", s.Status, s.Settings)
	}
	if s.Players["r"].Rating != 1500 {
		t.Errorf("rating %d", s.Players["r"].Rating)
	}
	m.handle(cs[0], fourIn{Type: "fill"}) // a full ranked game takes no bots
	for _, c := range cs {
		m.handle(c, fourIn{Type: "setup", Board: four.Extended.StandardSetup()})
	}
	// War Mode is played in the fog: most of the board is hidden at the start.
	if s := lastFour(t, cs[0]); s.Status != "playing" || slicesCount(s.Board, "?") == 0 {
		t.Fatalf("status %s, %d hidden squares", s.Status, slicesCount(s.Board, "?"))
	}
	// Blue resigns first, then Green, then Violet: Red wins.
	for _, i := range []int{1, 2, 3} {
		m.handle(cs[i], fourIn{Type: "resign"})
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		s = lastFour(t, cs[0])
		if s.Players["r"].RatingDiff != nil || time.Now().After(deadline) {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if s.Status != "over" || s.Result != "r" {
		t.Fatalf("status %s result %s", s.Status, s.Result)
	}
	diff := func(seat string) int {
		d := s.Players[seat].RatingDiff
		if d == nil {
			t.Fatalf("%s not rated", seat)
		}
		return *d
	}
	if !(diff("r") > diff("v") && diff("v") > diff("g") && diff("g") > diff("b")) || diff("r") <= 0 || diff("b") >= 0 {
		t.Errorf("diffs r %d v %d g %d b %d", diff("r"), diff("v"), diff("g"), diff("b"))
	}
	top, err := store.Leaderboard(10, true)
	if err != nil || len(top) != 4 || top[0].Username != "ann" || top[3].Username != "ben" {
		t.Errorf("four-player leaderboard %+v %v", top, err)
	}
	if one, _ := store.Leaderboard(10, false); len(one) != 0 {
		t.Errorf("the one-against-one leaderboard moved: %+v", one)
	}
	// Restoring doesn't rate it twice.
	hub.flush()
	again := NewFourHub(store).Get(m.id)
	time.Sleep(100 * time.Millisecond)
	if again == nil || again.ratingDiff == nil || *again.ratingDiff != *m.ratingDiff {
		t.Fatal("rating not restored")
	}
	if r, _ := store.Rank(ps[0].userID, true); r == nil || r.Games != 1 {
		t.Errorf("rated twice: %+v", r)
	}
}

func slicesCount(xs []string, v string) int {
	n := 0
	for _, x := range xs {
		if x == v {
			n++
		}
	}
	return n
}

func TestFourDemo(t *testing.T) {
	hub := NewFourHub(nil)
	hub.botPause = func() time.Duration { return 0 }
	if warMode.Time != (TimeControl{Base: 1800, Inc: 10}) || rankedFour.Time != warMode.Time || !rankedFour.Rated || warMode.Rated {
		t.Fatalf("war mode %+v ranked %+v", warMode, rankedFour)
	}
	p := &player{key: "guest:1", name: "Guest"}
	m := hub.Create(p, demoFour(bots.Ramesses))
	c := newClient(p.key)
	m.join(c, p)
	s := lastFour(t, c)
	if s.Status != "setup" || !s.Settings.Demo || s.Settings.Rated || s.Settings.Time.clocked() || s.Settings.Setup != "battleship" {
		t.Fatalf("status %s settings %+v", s.Status, s.Settings)
	}
	for _, seat := range []string{"b", "g", "v"} {
		if pl := s.Players[seat]; !pl.Bot || pl.Name != "Ramesses II" || !pl.Ready {
			t.Errorf("seat %s: %+v", seat, pl)
		}
	}
	m.handle(c, fourIn{Type: "setup", Board: four.Extended.StandardSetup()})
	if s := lastFour(t, c); s.Status != "playing" || len(s.Legal) == 0 {
		t.Fatalf("status %s", s.Status)
	}
	if demoFour("nobody").Bot != string(demoBot) {
		t.Error("an unknown bot should fall back to the demo bot")
	}
}
