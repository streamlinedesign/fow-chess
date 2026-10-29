package main

import (
	"math"
	"math/rand/v2"
	"net/http"
	"sync"
	"time"
)

// Matchmaking: signed-in players wait in a queue with the clock and mode they
// want, and the two closest ratings waiting for the same game are paired.
// A player stays in the queue only while their page keeps asking.

// queueStale is how long a player stays in the queue without polling.
const queueStale = 15 * time.Second

type queueEntry struct {
	user     *User
	settings Settings
	// since is when they joined; seen is their last poll.
	since, seen time.Time
}

type Queue struct {
	mu      sync.Mutex
	waiting map[int64]*queueEntry
	// found remembers the match a waiting player was paired into, for the
	// poll that follows the live event, or replaces it.
	found map[int64]foundMatch
}

type foundMatch struct {
	id string
	at time.Time
}

func NewQueue() *Queue {
	return &Queue{waiting: map[int64]*queueEntry{}, found: map[int64]foundMatch{}}
}

func sameGame(a, b Settings) bool {
	return a.Time == b.Time && a.Setup == b.Setup
}

// window is how far apart two ratings may be to pair, after waiting for d:
// close at first, anyone after half a minute.
func window(d time.Duration) float64 {
	return 100 + 20*d.Seconds()
}

// join puts u in the queue with the game they want, or renews their entry.
func (q *Queue) join(u *User, settings Settings) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.sweepLocked()
	delete(q.found, u.ID)
	if e := q.waiting[u.ID]; e != nil && sameGame(e.settings, settings) {
		e.seen = time.Now()
		return
	}
	q.waiting[u.ID] = &queueEntry{user: u, settings: settings, seen: time.Now(), since: time.Now()}
}

// pair takes u and the closest waiting player who wants the same game and
// is within either one's window out of the queue, and returns that player.
func (q *Queue) pair(u *User) *queueEntry {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.sweepLocked()
	me := q.waiting[u.ID]
	if me == nil {
		return nil
	}
	now := time.Now()
	var best *queueEntry
	for id, e := range q.waiting {
		if id == u.ID || !sameGame(e.settings, me.settings) {
			continue
		}
		gap := math.Abs(e.user.Rating - u.Rating)
		if gap > window(now.Sub(me.since)) && gap > window(now.Sub(e.since)) {
			continue
		}
		if best == nil || gap < math.Abs(best.user.Rating-u.Rating) {
			best = e
		}
	}
	if best != nil {
		delete(q.waiting, best.user.ID)
		delete(q.waiting, u.ID)
	}
	return best
}

// touch keeps u in the queue and reports whether they're still waiting, or
// the match they were paired into.
func (q *Queue) touch(u *User) (waiting bool, matchID string) {
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

func (q *Queue) leave(u *User) {
	q.mu.Lock()
	defer q.mu.Unlock()
	delete(q.waiting, u.ID)
	delete(q.found, u.ID)
}

func (q *Queue) paired(userID int64, matchID string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.found[userID] = foundMatch{id: matchID, at: time.Now()}
}

// Size is how many players are waiting right now.
func (q *Queue) Size() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.sweepLocked()
	return len(q.waiting)
}

func (q *Queue) sweepLocked() {
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

// pairUp starts the match two queued players were paired into and tells the
// one who was waiting.
func (s *Social) pairUp(me *User, other *queueEntry) string {
	pw, pb := userPlayer(me, ""), userPlayer(other.user, "")
	if rand.IntN(2) == 1 {
		pw, pb = pb, pw
	}
	m := s.hub.CreateBetween(pw, pb, other.settings)
	s.queue.paired(other.user.ID, m.id)
	s.live.send(other.user.ID, liveEvent{Type: "match", MatchID: m.id, User: me.Username})
	return m.id
}

func (s *Social) queueRoutes(mux *http.ServeMux) {
	// Join the queue, or get paired at once.
	mux.HandleFunc("POST /api/queue", func(w http.ResponseWriter, r *http.Request) {
		me := s.signedIn(w, r)
		if me == nil {
			return
		}
		var settings Settings
		if !readJSON(w, r, &settings) {
			return
		}
		settings.Mods, settings.Unrated, settings.Bot = false, false, ""
		settings.normalize()
		if !settings.valid() {
			jsonError(w, "Those match settings aren't available.", http.StatusBadRequest)
			return
		}
		s.queue.join(me, settings)
		if other := s.queue.pair(me); other != nil {
			writeJSON(w, map[string]any{"matchId": s.pairUp(me, other)})
			return
		}
		writeJSON(w, map[string]any{"waiting": true, "size": s.queue.Size()})
	})

	// Still waiting? Also what keeps a player in the queue, and where a
	// pair is made once the windows have widened.
	mux.HandleFunc("GET /api/queue", func(w http.ResponseWriter, r *http.Request) {
		me := s.signedIn(w, r)
		if me == nil {
			return
		}
		waiting, matchID := s.queue.touch(me)
		if waiting {
			if other := s.queue.pair(me); other != nil {
				waiting, matchID = false, s.pairUp(me, other)
			}
		}
		resp := map[string]any{"waiting": waiting, "size": s.queue.Size()}
		if matchID != "" {
			resp["matchId"] = matchID
		}
		writeJSON(w, resp)
	})

	mux.HandleFunc("DELETE /api/queue", func(w http.ResponseWriter, r *http.Request) {
		me := s.signedIn(w, r)
		if me == nil {
			return
		}
		s.queue.leave(me)
		w.WriteHeader(http.StatusNoContent)
	})
}
