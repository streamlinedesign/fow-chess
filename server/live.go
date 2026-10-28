package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// Live pushes events to signed-in players' open tabs over server-sent events:
// new notifications, messages, offers and friends. A player with a tab open is
// online.
type Live struct {
	mu    sync.Mutex
	conns map[int64]map[chan []byte]struct{}
	store *Store
	// playing, if set, is the match a player is playing right now, or "".
	playing func(userID int64) string
}

func NewLive(store *Store) *Live {
	return &Live{conns: map[int64]map[chan []byte]struct{}{}, store: store}
}

// liveEvent is one event: its type says what changed, so the page knows what to reload.
type liveEvent struct {
	Type string `json:"type"` // notification, message, friends, presence, offer
	// Who it's about, when that's someone other than the receiver.
	User    string `json:"user,omitempty"`
	MatchID string `json:"matchId,omitempty"`
	Online  *bool  `json:"online,omitempty"`
}

// send pushes an event to every tab userID has open. A tab that's too far
// behind misses it, and catches up the next time it loads.
func (l *Live) send(userID int64, ev liveEvent) {
	if l == nil || userID == 0 {
		return
	}
	b, _ := json.Marshal(ev)
	l.mu.Lock()
	defer l.mu.Unlock()
	for ch := range l.conns[userID] {
		select {
		case ch <- b:
		default:
		}
	}
}

// online is whether userID has the site open.
func (l *Live) online(userID int64) bool {
	if l == nil {
		return false
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.conns[userID]) > 0
}

// card is how a player shows up in lists: online only if they have the site
// open and don't hide it.
func (l *Live) card(p person) userCard {
	c := userCard{Username: p.Username, Rating: int(p.Rating + 0.5)}
	if !p.HideOnline {
		c.Online = l.online(p.ID)
		if c.Online && l.playing != nil {
			c.Playing = l.playing(p.ID)
		}
		if !p.LastSeen.IsZero() {
			c.LastSeen = p.LastSeen.UnixMilli()
		}
	}
	return c
}

// userCard is another player in a list or a chat.
type userCard struct {
	Username string `json:"username"`
	Rating   int    `json:"rating"`
	Online   bool   `json:"online"`
	LastSeen int64  `json:"lastSeen,omitempty"` // unix milliseconds; left out when hidden
	// Playing is the match they're playing right now, so friends can watch.
	Playing string `json:"playing,omitempty"`
}

func (l *Live) cards(ps []person) []userCard {
	out := make([]userCard, len(ps))
	for i, p := range ps {
		out[i] = l.card(p)
	}
	return out
}

// serve streams events to one tab until it closes.
func (l *Live) serve(w http.ResponseWriter, r *http.Request, u *User) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Accel-Buffering", "no")

	ch := make(chan []byte, 32)
	l.mu.Lock()
	first := len(l.conns[u.ID]) == 0
	if first {
		l.conns[u.ID] = map[chan []byte]struct{}{}
	}
	l.conns[u.ID][ch] = struct{}{}
	l.mu.Unlock()
	if first {
		l.presence(u.ID)
	}
	defer func() {
		l.mu.Lock()
		delete(l.conns[u.ID], ch)
		last := len(l.conns[u.ID]) == 0
		if last {
			delete(l.conns, u.ID)
		}
		l.mu.Unlock()
		if last {
			l.presence(u.ID)
		}
	}()

	fmt.Fprint(w, "retry: 3000\n\n")
	flusher.Flush()
	ping := time.NewTicker(25 * time.Second)
	defer ping.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case b := <-ch:
			fmt.Fprintf(w, "data: %s\n\n", b)
			flusher.Flush()
		case <-ping.C:
			// Keeps proxies from closing an idle stream.
			fmt.Fprint(w, ": ping\n\n")
			flusher.Flush()
		}
	}
}

// presence records when a player comes and goes, and tells their friends who are online.
func (l *Live) presence(userID int64) {
	go func() {
		if err := l.store.TouchLastSeen(userID); err != nil {
			log.Printf("last seen: %v", err)
		}
		p, err := l.store.Person(userID)
		if err != nil || p == nil || p.HideOnline {
			return
		}
		ids, err := l.store.FriendIDs(userID)
		if err != nil {
			return
		}
		// Recheck: a quick reload closes and opens a tab, and the friends
		// should see where it ended up.
		now := l.online(userID)
		for _, id := range ids {
			l.send(id, liveEvent{Type: "presence", User: p.Username, Online: &now})
		}
	}()
}
