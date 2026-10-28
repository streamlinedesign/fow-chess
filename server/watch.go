package main

import (
	"cmp"
	"log"
	"net/http"
	"slices"
)

// liveGame is a game being played right now, as the Watch page lists it.
type liveGame struct {
	ID       string              `json:"id"`
	Settings Settings            `json:"settings"`
	Players  map[string]liveSeat `json:"players"`
	// Board is what a spectator seeing it as white sees; see spectatorView.
	Board      string `json:"board"`
	Ply        int    `json:"ply"`
	Turn       string `json:"turn"`
	Spectators int    `json:"spectators"`
	Rated      bool   `json:"rated"`
	// Friend is whether one of the players is the viewer's friend.
	Friend bool `json:"friend"`
	users  [2]int64
}

type liveSeat struct {
	Name   string `json:"name,omitempty"` // empty for a guest
	Rating int    `json:"rating,omitempty"`
}

// matchList is the matches in memory, so callers can lock each one without holding h.mu.
func (h *Hub) matchList() []*Match {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]*Match, 0, len(h.matches))
	for _, m := range h.matches {
		out = append(out, m)
	}
	return out
}

// LiveGames lists the games being played right now.
func (h *Hub) LiveGames() []liveGame {
	out := []liveGame{}
	for _, m := range h.matchList() {
		m.mu.Lock()
		if m.live() {
			v := spectatorView(m.game, m.settings.rules(), white, m.settings.Fog)
			out = append(out, liveGame{
				ID: m.id, Settings: m.settings, Board: v.Board, Ply: v.Ply, Turn: v.Turn,
				Spectators: m.spectators(), Rated: m.rated(), users: m.users,
				Players: map[string]liveSeat{"w": {Name: m.names[white]}, "b": {Name: m.names[black]}},
			})
		}
		m.mu.Unlock()
	}
	return out
}

// Playing is the match userID is playing right now, or "" if none: a live
// match where they hold a seat and have it open.
func (h *Hub) Playing(userID int64) string {
	for _, m := range h.matchList() {
		m.mu.Lock()
		playing := m.live() && (m.users[white] == userID && m.online(white) || m.users[black] == userID && m.online(black))
		m.mu.Unlock()
		if playing {
			return m.id
		}
	}
	return ""
}

// maxWatch is how many live games the Watch page lists.
const maxWatch = 60

// watchRoutes serves the Watch page's list. Anyone can watch, signed in or not.
func (s *Social) watchRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/watch", func(w http.ResponseWriter, r *http.Request) {
		games := s.hub.LiveGames()
		var ids []int64
		for _, g := range games {
			for _, id := range g.users {
				if id != 0 {
					ids = append(ids, id)
				}
			}
		}
		ratings := map[int64]int{}
		if len(ids) > 0 {
			ps, err := s.store.people(`WHERE u.id = ANY($1)`, ids)
			if err != nil {
				log.Printf("watch ratings: %v", err)
			}
			for _, p := range ps {
				ratings[p.ID] = int(p.Rating + 0.5)
			}
		}
		friends := map[int64]bool{}
		me := s.auth.user(r)
		if me != nil {
			ids, _ := s.store.FriendIDs(me.ID)
			for _, id := range ids {
				friends[id] = true
			}
		}

		out := []liveGame{}
		for _, g := range games {
			// Your own games are on the home page; you can't watch yourself.
			if me != nil && (g.users[white] == me.ID || g.users[black] == me.ID) {
				continue
			}
			for c, name := range colorNames {
				seat := g.Players[name]
				seat.Rating = ratings[g.users[c]]
				g.Players[name] = seat
			}
			g.Friend = friends[g.users[white]] || friends[g.users[black]]
			out = append(out, g)
		}
		// Friends' games first, then the strongest players, then the longest games.
		best := func(g liveGame) int { return max(g.Players["w"].Rating, g.Players["b"].Rating) }
		slices.SortStableFunc(out, func(a, b liveGame) int {
			if a.Friend != b.Friend {
				if a.Friend {
					return -1
				}
				return 1
			}
			return cmp.Or(cmp.Compare(best(b), best(a)), cmp.Compare(b.Ply, a.Ply), cmp.Compare(a.ID, b.ID))
		})
		if len(out) > maxWatch {
			out = out[:maxWatch]
		}
		writeJSON(w, map[string]any{"games": out})
	})
}
