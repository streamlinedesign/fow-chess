package main

import (
	"testing"
	"time"
)

func TestQueuePairsClosestRating(t *testing.T) {
	q := NewQueue()
	blitz := Settings{Time: TimeControl{Base: 300}}
	rapid := Settings{Time: TimeControl{Base: 600}}
	ann := &User{ID: 1, Rating: 1500}
	ben := &User{ID: 2, Rating: 1700}
	eve := &User{ID: 3, Rating: 1600}
	cat := &User{ID: 4, Rating: 1650}
	dan := &User{ID: 5, Rating: 1200}

	join := func(u *User, s Settings) *queueEntry {
		q.join(u, s)
		return q.pair(u)
	}
	// Nobody is close enough yet, so four players wait.
	for _, j := range []struct {
		u *User
		s Settings
	}{{ann, blitz}, {ben, blitz}, {eve, rapid}, {dan, rapid}} {
		if got := join(j.u, j.s); got != nil {
			t.Fatalf("user %d paired with %d, who is too far away for now", j.u.ID, got.user.ID)
		}
	}
	if got := join(cat, blitz); got == nil || got.user != ben {
		t.Fatalf("cat (1650) should pair with ben (1700), the closest blitz player in range, got %+v", got)
	}
	if waiting, _ := q.touch(ann); !waiting {
		t.Fatal("ann should still be waiting")
	}
	// Half a minute later anyone wanting the same game will do.
	q.mu.Lock()
	q.waiting[eve.ID].since = time.Now().Add(-40 * time.Second)
	q.mu.Unlock()
	if got := q.pair(eve); got == nil || got.user != dan {
		t.Fatalf("eve has waited long enough to pair with dan, got %+v", got)
	}
	if q.Size() != 1 {
		t.Fatalf("size %d, want 1 (ann)", q.Size())
	}

	// The paired player learns the match from a poll, once.
	q.paired(ben.ID, "ABC123")
	if waiting, id := q.touch(ben); waiting || id != "ABC123" {
		t.Fatalf("ben: waiting %v match %q", waiting, id)
	}

	// A player whose page stopped asking drops out.
	q.mu.Lock()
	q.waiting[ann.ID].seen = time.Now().Add(-2 * queueStale)
	q.mu.Unlock()
	if waiting, _ := q.touch(ann); waiting {
		t.Fatal("ann should have been swept")
	}
	if q.Size() != 0 {
		t.Fatalf("size %d, want 0", q.Size())
	}
}

func TestSettingsNormalize(t *testing.T) {
	s := Settings{Fog: false, KingCapture: false, Hill: true, Time: TimeControl{Base: 300}}
	s.normalize()
	if !s.Fog || !s.KingCapture || s.Hill || !s.valid() {
		t.Errorf("a plain game must be fog + king capture with no twists: %+v", s)
	}
	m := Settings{Mods: true, Fog: false, KingCapture: false, Hill: true, Time: TimeControl{}}
	m.normalize()
	if m.Fog || m.KingCapture || !m.Hill || !m.Unrated || !m.valid() {
		t.Errorf("a mods game keeps its options, has any clock and is never ranked: %+v", m)
	}
	if (Settings{Time: TimeControl{}}).valid() {
		t.Error("no clock is only for mods games")
	}
}
