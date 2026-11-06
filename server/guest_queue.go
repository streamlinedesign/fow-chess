package main

import (
	"encoding/json"
	"math/rand/v2"
	"net/http"
	"slices"
	"sync"
	"time"
)

// Play as guest: browsers without an account wait in a queue of their own for
// one fixed, unranked game each, Eat the King at Rapid 10+0 or War Mode at 30
// minutes plus 10 seconds a move. Whoever has waited longest is seated first;
// there are no ratings to match. Signed-in players use the ranked queues.

// guestGames are the two games a guest can queue for, and how many players each seats.
var guestGames = map[string]int{"etk": 2, "war": 4}

// guestRapid is the guests' Eat the King clock: Rapid 10+0.
var guestRapid = TimeControl{Base: 600}

type guestEntry struct {
	id   string // the browser's guest ID
	game string
	// since is when they joined; seen is their last poll.
	since, seen time.Time
}

type GuestQueue struct {
	mu      sync.Mutex
	waiting map[string]*guestEntry
	found   map[string]foundMatch
}

func NewGuestQueue() *GuestQueue {
	return &GuestQueue{waiting: map[string]*guestEntry{}, found: map[string]foundMatch{}}
}

// join puts a guest in the queue for game, or renews their entry.
func (q *GuestQueue) join(id, game string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.sweepLocked()
	delete(q.found, id)
	if e := q.waiting[id]; e != nil && e.game == game {
		e.seen = time.Now()
		return
	}
	now := time.Now()
	q.waiting[id] = &guestEntry{id: id, game: game, since: now, seen: now}
}

// group takes id and the guests who have waited longest for the same game out
// of the queue once there are enough for one, and returns them all, id among
// them. Nil while there aren't enough.
func (q *GuestQueue) group(id string) []*guestEntry {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.sweepLocked()
	me := q.waiting[id]
	if me == nil {
		return nil
	}
	n := guestGames[me.game]
	var others []*guestEntry
	for _, e := range q.waiting {
		if e.id != id && e.game == me.game {
			others = append(others, e)
		}
	}
	if len(others) < n-1 {
		return nil
	}
	slices.SortFunc(others, func(a, b *guestEntry) int { return a.since.Compare(b.since) })
	g := append([]*guestEntry{me}, others[:n-1]...)
	for _, e := range g {
		delete(q.waiting, e.id)
	}
	return g
}

// touch keeps a guest in the queue and reports whether they're still waiting,
// or the match they were seated in.
func (q *GuestQueue) touch(id string) (waiting bool, matchID string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.sweepLocked()
	if f, ok := q.found[id]; ok {
		return false, f.id
	}
	if e := q.waiting[id]; e != nil {
		e.seen = time.Now()
		return true, ""
	}
	return false, ""
}

func (q *GuestQueue) leave(id string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	delete(q.waiting, id)
	delete(q.found, id)
}

func (q *GuestQueue) seated(id, matchID string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.found[id] = foundMatch{id: matchID, at: time.Now()}
}

// size is how many guests are waiting for the game id waits for.
func (q *GuestQueue) size(id string) int {
	q.mu.Lock()
	defer q.mu.Unlock()
	me := q.waiting[id]
	if me == nil {
		return 0
	}
	n := 0
	for _, e := range q.waiting {
		if e.game == me.game {
			n++
		}
	}
	return n
}

func (q *GuestQueue) sweepLocked() {
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

// seatGuests starts the unranked game a group of guests was put together for,
// in a random order, and remembers it for the ones who were waiting.
func (s *Social) seatGuests(me string, g []*guestEntry) string {
	rand.Shuffle(len(g), func(i, j int) { g[i], g[j] = g[j], g[i] })
	var id string
	if g[0].game == "war" {
		var ps [4]*player
		for c, e := range g {
			ps[c] = guestPlayer(e.id)
		}
		id = s.fours.CreateSeated(ps, warFour(warTimes[0]), [4]int{}).id
	} else {
		settings := Settings{Time: guestRapid}
		settings.normalize()
		settings.Unrated = true // even if both sign in before it ends
		id = s.hub.CreateBetween(guestPlayer(g[0].id), guestPlayer(g[1].id), settings).id
	}
	for _, e := range g {
		if e.id != me {
			s.guestQueue.seated(e.id, id)
		}
	}
	return id
}

func (s *Social) guestQueueRoutes(mux *http.ServeMux) {
	// guest is the browser asking, or nil (with the error written) for a
	// signed-in player or a request without a guest ID.
	guest := func(w http.ResponseWriter, r *http.Request) string {
		if s.auth.user(r) != nil {
			jsonError(w, "You're signed in: use Play for a ranked game.", http.StatusBadRequest)
			return ""
		}
		p := playerFor(r, nil)
		if p == nil {
			jsonError(w, "Missing player.", http.StatusBadRequest)
			return ""
		}
		return p.key
	}
	reply := func(w http.ResponseWriter, id string, waiting bool, matchID string) {
		resp := map[string]any{"waiting": waiting, "size": s.guestQueue.size(id)}
		if matchID != "" {
			resp["matchId"] = matchID
		}
		writeJSON(w, resp)
	}

	// Join the guest queue for "etk" or "war", or get seated at once.
	mux.HandleFunc("POST /api/guest/queue", func(w http.ResponseWriter, r *http.Request) {
		id := guest(w, r)
		if id == "" {
			return
		}
		var body struct {
			Game string `json:"game"`
		}
		json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024)).Decode(&body)
		if guestGames[body.Game] == 0 {
			jsonError(w, "Guests can play etk or war.", http.StatusBadRequest)
			return
		}
		s.guestQueue.join(id, body.Game)
		if g := s.guestQueue.group(id); g != nil {
			reply(w, id, false, s.seatGuests(id, g))
			return
		}
		reply(w, id, true, "")
	})

	// Still waiting? Also what keeps a guest in the queue.
	mux.HandleFunc("GET /api/guest/queue", func(w http.ResponseWriter, r *http.Request) {
		id := guest(w, r)
		if id == "" {
			return
		}
		waiting, matchID := s.guestQueue.touch(id)
		if waiting {
			if g := s.guestQueue.group(id); g != nil {
				waiting, matchID = false, s.seatGuests(id, g)
			}
		}
		reply(w, id, waiting, matchID)
	})

	mux.HandleFunc("DELETE /api/guest/queue", func(w http.ResponseWriter, r *http.Request) {
		if id := guest(w, r); id != "" {
			s.guestQueue.leave(id)
			w.WriteHeader(http.StatusNoContent)
		}
	})
}
