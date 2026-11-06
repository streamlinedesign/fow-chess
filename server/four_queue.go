package main

import (
	"math"
	"math/rand/v2"
	"net/http"
	"slices"
	"sync"
	"time"
)

// The four-player queue: signed-in players wait for ranked four-player
// games, and the four closest four-player ratings are seated together once
// they're within the window of whoever has waited longest. A player stays
// in the queue only while their page keeps asking.

type fourEntry struct {
	user   *User
	rating int
	// since is when they joined; seen is their last poll.
	since, seen time.Time
}

type FourQueue struct {
	mu      sync.Mutex
	waiting map[int64]*fourEntry
	found   map[int64]foundMatch
}

func NewFourQueue() *FourQueue {
	return &FourQueue{waiting: map[int64]*fourEntry{}, found: map[int64]foundMatch{}}
}

func (q *FourQueue) join(u *User, rating int) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.sweepLocked()
	delete(q.found, u.ID)
	if e := q.waiting[u.ID]; e != nil {
		e.seen = time.Now()
		return
	}
	now := time.Now()
	q.waiting[u.ID] = &fourEntry{user: u, rating: rating, since: now, seen: now}
}

// group takes four waiting players including u out of the queue, the
// closest ratings first, once their spread is within the window of the one
// who has waited longest. Nil while there's no such four.
func (q *FourQueue) group(u *User) []*fourEntry {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.sweepLocked()
	if q.waiting[u.ID] == nil || len(q.waiting) < 4 {
		return nil
	}
	all := make([]*fourEntry, 0, len(q.waiting))
	for _, e := range q.waiting {
		all = append(all, e)
	}
	slices.SortFunc(all, func(a, b *fourEntry) int { return a.rating - b.rating })
	now := time.Now()
	var best []*fourEntry
	bestSpread := math.MaxInt
	for i := 0; i+4 <= len(all); i++ {
		four := all[i : i+4]
		if !slices.ContainsFunc(four, func(e *fourEntry) bool { return e.user.ID == u.ID }) {
			continue
		}
		spread := four[3].rating - four[0].rating
		var wait time.Duration
		for _, e := range four {
			wait = max(wait, now.Sub(e.since))
		}
		if float64(spread) <= window(wait) && spread < bestSpread {
			best, bestSpread = four, spread
		}
	}
	for _, e := range best {
		delete(q.waiting, e.user.ID)
	}
	return best
}

// touch keeps u in the queue and reports whether they're still waiting, or
// the match they were seated in.
func (q *FourQueue) touch(u *User) (waiting bool, matchID string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.sweepLocked()
	if f, ok := q.found[u.ID]; ok {
		return false, f.id
	}
	if e := q.waiting[u.ID]; e != nil {
		e.seen = time.Now()
		return true, ""
	}
	return false, ""
}

func (q *FourQueue) leave(u *User) {
	q.mu.Lock()
	defer q.mu.Unlock()
	delete(q.waiting, u.ID)
	delete(q.found, u.ID)
}

func (q *FourQueue) seated(userID int64, matchID string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.found[userID] = foundMatch{id: matchID, at: time.Now()}
}

func (q *FourQueue) Size() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.sweepLocked()
	return len(q.waiting)
}

func (q *FourQueue) sweepLocked() {
	now := time.Now()
	for id, e := range q.waiting {
		if now.Sub(e.seen) > queueStale {
			delete(q.waiting, id)
		}
	}
	for id, f := range q.found {
		if now.Sub(f.at) > time.Minute {
			delete(q.found, id)
		}
	}
}

// seatFour starts the ranked game four queued players were grouped into, in
// a random order round the board, and tells the ones who were waiting.
func (s *Social) seatFour(me *User, group []*fourEntry) string {
	rand.Shuffle(len(group), func(i, j int) { group[i], group[j] = group[j], group[i] })
	var ps [4]*player
	var ratings [4]int
	for c, e := range group {
		ps[c], ratings[c] = userPlayer(e.user, ""), e.rating
	}
	m := s.fours.CreateRanked(ps, ratings)
	for _, e := range group {
		if e.user.ID != me.ID {
			s.fourQueue.seated(e.user.ID, m.id)
			s.live.send(e.user.ID, liveEvent{Type: "four", MatchID: m.id})
		}
	}
	return m.id
}

func (s *Social) fourQueueRoutes(mux *http.ServeMux) {
	reply := func(w http.ResponseWriter, waiting bool, matchID string) {
		resp := map[string]any{"waiting": waiting, "size": s.fourQueue.Size()}
		if matchID != "" {
			resp["matchId"] = matchID
		}
		writeJSON(w, resp)
	}

	// Join the four-player queue, or get seated at once.
	mux.HandleFunc("POST /api/four/queue", func(w http.ResponseWriter, r *http.Request) {
		me := s.signedIn(w, r)
		if me == nil {
			return
		}
		rating := s.fours.fourRatings([4]int64{me.ID})[0]
		s.fourQueue.join(me, rating)
		if g := s.fourQueue.group(me); g != nil {
			reply(w, false, s.seatFour(me, g))
			return
		}
		reply(w, true, "")
	})

	// Still waiting? Also what keeps a player in the queue, and where a
	// group is made once the window has widened.
	mux.HandleFunc("GET /api/four/queue", func(w http.ResponseWriter, r *http.Request) {
		me := s.signedIn(w, r)
		if me == nil {
			return
		}
		waiting, matchID := s.fourQueue.touch(me)
		if waiting {
			if g := s.fourQueue.group(me); g != nil {
				waiting, matchID = false, s.seatFour(me, g)
			}
		}
		reply(w, waiting, matchID)
	})

	mux.HandleFunc("DELETE /api/four/queue", func(w http.ResponseWriter, r *http.Request) {
		if me := s.signedIn(w, r); me != nil {
			s.fourQueue.leave(me)
			w.WriteHeader(http.StatusNoContent)
		}
	})
}
