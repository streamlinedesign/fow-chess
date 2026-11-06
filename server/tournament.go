package main

import (
	"log"
	"math/rand/v2"
	"net/http"
	"slices"
	"sort"
	"strconv"
	"sync"
	"time"
	_ "time/tzdata"

	"chess/engine"
)

// Arenas: a tournament at a fixed time every day. Players join before or
// during it, are paired as soon as they're free with whoever is nearest in
// score, and score 2 for a win and 1 for a draw until the arena ends. Games
// are ranked like any other.

// arenaKind is one of the daily arenas.
type arenaKind struct {
	Kind         string
	Hour, Minute int // local time in arenaZone
	Length       time.Duration
	Time         TimeControl
}

var arenaKinds = []arenaKind{
	{Kind: "rapid", Hour: 19, Minute: 0, Length: 45 * time.Minute, Time: TimeControl{Base: 600}},
	{Kind: "blitz", Hour: 20, Minute: 0, Length: 30 * time.Minute, Time: TimeControl{Base: 300}},
}

const (
	arenaZoneName = "Europe/Zagreb"
	// minArenaPlayers is how many must have joined when an arena starts, or it's called off.
	minArenaPlayers = 4
	// arenaDaysAhead is how far ahead arenas are put on the calendar.
	arenaDaysAhead = 2
	// Points for a win and a draw.
	winPoints, drawPoints = 2, 1

	notifyArenaCancelled = "arena_cancelled"
	notifyArenaResult    = "arena_result"
)

var arenaZone = func() *time.Location {
	z, err := time.LoadLocation(arenaZoneName)
	if err != nil {
		log.Printf("arena zone %s: %v; using UTC", arenaZoneName, err)
		return time.UTC
	}
	return z
}()

func arenaKindOf(kind string) (arenaKind, bool) {
	for _, k := range arenaKinds {
		if k.Kind == kind {
			return k, true
		}
	}
	return arenaKind{}, false
}

// startsOn is when k starts on the day of d, in the arena's zone.
func (k arenaKind) startsOn(d time.Time) time.Time {
	d = d.In(arenaZone)
	return time.Date(d.Year(), d.Month(), d.Day(), k.Hour, k.Minute, 0, 0, arenaZone)
}

// arenaPlayer is a player in a running arena, as the pairing sees them.
type arenaPlayer struct {
	user    *User
	score   int
	games   int
	wins    int
	paused  bool
	busy    bool  // in an arena game right now
	lastOpp int64 // who they played last, to vary pairings
	lastW   bool  // played White last
}

// arena is a running tournament's state in memory.
type arena struct {
	row     tournamentRow
	kind    arenaKind
	players map[int64]*arenaPlayer
	games   map[string]tournamentGame // by match id + round
}

type Tournaments struct {
	mu     sync.Mutex
	store  *Store
	hub    *Hub
	live   *Live
	notify func(notification)
	arenas map[int64]*arena
	// now is the clock and online is presence, so tests can control both.
	now    func() time.Time
	online func(userID int64) bool
}

func NewTournaments(store *Store, hub *Hub, live *Live, notify func(notification)) *Tournaments {
	return &Tournaments{store: store, hub: hub, live: live, notify: notify, arenas: map[int64]*arena{}, now: time.Now, online: live.online}
}

// run keeps the calendar filled and the arenas moving.
func (t *Tournaments) run() {
	for {
		t.tick()
		time.Sleep(3 * time.Second)
	}
}

// tick does one round of housekeeping: schedule, start, pair, finish.
func (t *Tournaments) tick() {
	t.mu.Lock()
	defer t.mu.Unlock()
	now := t.now()
	for day := 0; day <= arenaDaysAhead; day++ {
		for _, k := range arenaKinds {
			start := k.startsOn(now.AddDate(0, 0, day))
			if start.Before(now) {
				continue
			}
			if err := t.store.EnsureTournament(k.Kind, start, start.Add(k.Length)); err != nil {
				log.Printf("arena calendar: %v", err)
			}
		}
	}
	rows, err := t.store.UpcomingTournaments()
	if err != nil {
		log.Printf("arenas: %v", err)
		return
	}
	for _, row := range rows {
		switch {
		case row.Status == "scheduled" && !now.Before(row.StartsAt):
			t.start(row)
		case row.Status == "running" && t.arenas[row.ID] == nil:
			t.load(row)
		}
		if a := t.arenas[row.ID]; a != nil {
			if !now.Before(a.row.EndsAt) {
				t.finish(a)
			} else {
				t.pair(a)
			}
		}
	}
}

// start opens a scheduled arena, or calls it off when too few joined.
func (t *Tournaments) start(row tournamentRow) {
	players, err := t.store.TournamentPlayers(row.ID)
	if err != nil {
		log.Printf("arena %d: %v", row.ID, err)
		return
	}
	if len(players) < minArenaPlayers {
		if err := t.store.SetTournamentStatus(row.ID, "cancelled"); err != nil {
			log.Printf("arena %d: %v", row.ID, err)
			return
		}
		for _, p := range players {
			t.notify(notification{UserID: p.UserID, Kind: notifyArenaCancelled, Data: map[string]any{
				"kind": row.Kind, "players": len(players), "needed": minArenaPlayers, "startsAt": row.StartsAt.UnixMilli(),
			}})
		}
		return
	}
	if err := t.store.SetTournamentStatus(row.ID, "running"); err != nil {
		log.Printf("arena %d: %v", row.ID, err)
		return
	}
	row.Status = "running"
	t.load(row)
}

// load brings a running arena into memory, after a start or a restart.
func (t *Tournaments) load(row tournamentRow) {
	kind, ok := arenaKindOf(row.Kind)
	if !ok {
		return
	}
	a := &arena{row: row, kind: kind, players: map[int64]*arenaPlayer{}, games: map[string]tournamentGame{}}
	players, err := t.store.TournamentPlayers(row.ID)
	if err != nil {
		log.Printf("arena %d: %v", row.ID, err)
		return
	}
	for _, p := range players {
		u, err := t.store.UserByID(p.UserID)
		if err != nil || u == nil {
			continue
		}
		a.players[p.UserID] = &arenaPlayer{user: u, score: p.Score, games: p.Games, wins: p.Wins, paused: p.Paused}
	}
	open, err := t.store.OpenTournamentGames(row.ID)
	if err != nil {
		log.Printf("arena %d: %v", row.ID, err)
	}
	for _, g := range open {
		a.games[g.MatchID+"/"+strconv.Itoa(g.Round)] = g
		for _, id := range []int64{g.White, g.Black} {
			if p := a.players[id]; p != nil {
				p.busy = true
			}
		}
	}
	t.arenas[row.ID] = a
}

// ready are the players who can be paired right now: in, not sitting out,
// not playing, and here. Callers hold t.mu.
func (t *Tournaments) ready(a *arena) []*arenaPlayer {
	var out []*arenaPlayer
	for _, p := range a.players {
		if !p.paused && !p.busy && t.online(p.user.ID) {
			out = append(out, p)
		}
	}
	// Best first, so neighbours in the list are neighbours in score.
	sort.Slice(out, func(i, j int) bool {
		if out[i].score != out[j].score {
			return out[i].score > out[j].score
		}
		return out[i].user.Rating > out[j].user.Rating
	})
	return out
}

// pair starts games between the ready players, nearest in score first,
// avoiding an immediate rematch when there's someone else to play.
func (t *Tournaments) pair(a *arena) {
	ready := t.ready(a)
	for len(ready) >= 2 {
		p := ready[0]
		j := 1
		if ready[j].user.ID == p.lastOpp && len(ready) > 2 {
			j = 2
		}
		q := ready[j]
		ready = slices.Delete(ready, j, j+1)
		ready = ready[1:]
		t.startGame(a, p, q)
	}
}

// startGame seats two arena players, giving the colours they didn't have last time where it can.
func (t *Tournaments) startGame(a *arena, p, q *arenaPlayer) {
	w, b := p, q
	if p.lastW && !q.lastW || (p.lastW == q.lastW && rand.IntN(2) == 1) {
		w, b = q, p
	}
	settings := Settings{Fog: true, KingCapture: true, CenterVision: "any", Time: a.kind.Time, Tournament: a.row.ID}
	m := t.hub.CreateBetween(userPlayer(w.user, ""), userPlayer(b.user, ""), settings)
	g := tournamentGame{MatchID: m.id, Round: 1, White: w.user.ID, Black: b.user.ID}
	if err := t.store.AddTournamentGame(a.row.ID, g); err != nil {
		log.Printf("arena %d: %v", a.row.ID, err)
	}
	a.games[g.MatchID+"/1"] = g
	for _, x := range []*arenaPlayer{w, b} {
		x.busy = true
	}
	p.lastOpp, q.lastOpp = q.user.ID, p.user.ID
	w.lastW, b.lastW = true, false
	t.live.send(w.user.ID, liveEvent{Type: "match", MatchID: m.id, User: b.user.Username})
	t.live.send(b.user.ID, liveEvent{Type: "match", MatchID: m.id, User: w.user.Username})
}

// gameOver scores a finished arena game. Games that end after the arena
// has finished don't count, but the players are free again either way.
func (t *Tournaments) gameOver(r gameResult) {
	if r.Settings.Tournament == 0 {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	a := t.arenas[r.Settings.Tournament]
	if a == nil {
		return
	}
	key := r.MatchID + "/" + strconv.Itoa(r.Round)
	g, ok := a.games[key]
	if !ok {
		return
	}
	delete(a.games, key)
	var points [2]int
	var wins [2]bool
	switch r.Outcome {
	case engine.WhiteWins:
		points, wins = [2]int{winPoints, 0}, [2]bool{true, false}
	case engine.BlackWins:
		points, wins = [2]int{0, winPoints}, [2]bool{false, true}
	default:
		points = [2]int{drawPoints, drawPoints}
	}
	if err := t.store.ScoreTournamentGame(a.row.ID, g, r.Outcome.String(), points, wins); err != nil {
		log.Printf("arena %d: %v", a.row.ID, err)
	}
	for i, id := range []int64{g.White, g.Black} {
		if p := a.players[id]; p != nil {
			p.busy = false
			p.score += points[i]
			p.games++
			if wins[i] {
				p.wins++
			}
		}
	}
	t.pair(a)
}

// finish closes an arena and tells everyone where they came.
func (t *Tournaments) finish(a *arena) {
	if err := t.store.SetTournamentStatus(a.row.ID, "finished"); err != nil {
		log.Printf("arena %d: %v", a.row.ID, err)
		return
	}
	delete(t.arenas, a.row.ID)
	standings := a.standings()
	for i, p := range standings {
		t.notify(notification{UserID: p.UserID, Kind: notifyArenaResult, Data: map[string]any{
			"kind": a.row.Kind, "place": i + 1, "players": len(standings), "score": p.Score, "startsAt": a.row.StartsAt.UnixMilli(),
		}})
	}
}

// standings is the arena's players best first. Callers hold t.mu.
func (a *arena) standings() []tournamentPlayer {
	out := make([]tournamentPlayer, 0, len(a.players))
	for _, p := range a.players {
		out = append(out, tournamentPlayer{UserID: p.user.ID, Username: p.user.Username, Rating: int(p.user.Rating + 0.5),
			Score: p.score, Games: p.games, Wins: p.wins, Paused: p.paused})
	}
	sort.Slice(out, func(i, j int) bool {
		x, y := out[i], out[j]
		if x.Score != y.Score {
			return x.Score > y.Score
		}
		if x.Wins != y.Wins {
			return x.Wins > y.Wins
		}
		if x.Games != y.Games {
			return x.Games < y.Games
		}
		return x.Username < y.Username
	})
	return out
}

// join puts a player in a tournament, running or not.
func (t *Tournaments) join(row *tournamentRow, u *User) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if err := t.store.JoinTournament(row.ID, u.ID); err != nil {
		return err
	}
	if a := t.arenas[row.ID]; a != nil {
		if p := a.players[u.ID]; p != nil {
			p.paused = false
		} else {
			a.players[u.ID] = &arenaPlayer{user: u}
		}
		t.pair(a)
	}
	return nil
}

// leave takes a player out of a scheduled tournament, or sits them out of a
// running one: their games so far still count.
func (t *Tournaments) leave(row *tournamentRow, u *User) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if a := t.arenas[row.ID]; a != nil {
		if p := a.players[u.ID]; p != nil {
			p.paused = true
		}
		return t.store.PauseTournament(row.ID, u.ID, true)
	}
	return t.store.LeaveTournament(row.ID, u.ID)
}

// standingsOf is a running arena's table, or nil.
func (t *Tournaments) standingsOf(id int64) []tournamentPlayer {
	t.mu.Lock()
	defer t.mu.Unlock()
	if a := t.arenas[id]; a != nil {
		return a.standings()
	}
	return nil
}

// A tournament as the page shows it.
type tournamentView struct {
	tournamentRow
	Time      TimeControl        `json:"time"`
	Standings []tournamentPlayer `json:"standings,omitempty"`
	Joined    bool               `json:"joined,omitempty"`
	Paused    bool               `json:"paused,omitempty"`
}

func (t *Tournaments) view(row tournamentRow, me *User) tournamentView {
	v := tournamentView{tournamentRow: row}
	if k, ok := arenaKindOf(row.Kind); ok {
		v.Time = k.Time
	}
	if row.Status == "running" {
		v.Standings = t.standingsOf(row.ID)
	}
	if row.Status == "finished" {
		if players, err := t.store.TournamentPlayers(row.ID); err == nil {
			v.Standings = players
		}
	}
	if me != nil {
		for _, p := range v.Standings {
			if p.UserID == me.ID {
				v.Joined, v.Paused = true, p.Paused
			}
		}
		if row.Status == "scheduled" {
			if players, err := t.store.TournamentPlayers(row.ID); err == nil {
				for _, p := range players {
					v.Joined = v.Joined || p.UserID == me.ID
				}
			}
		}
	}
	return v
}

func (s *Social) tournamentRoutes(mux *http.ServeMux, t *Tournaments) {
	// What's coming, what's on, and what just happened.
	mux.HandleFunc("GET /api/tournaments", func(w http.ResponseWriter, r *http.Request) {
		me := s.auth.user(r)
		upcoming, err := s.store.UpcomingTournaments()
		if serverError(w, err) {
			return
		}
		recent, err := s.store.RecentTournaments(5)
		if serverError(w, err) {
			return
		}
		views := func(rows []tournamentRow) []tournamentView {
			out := []tournamentView{}
			for _, row := range rows {
				out = append(out, t.view(row, me))
			}
			return out
		}
		writeJSON(w, map[string]any{
			"upcoming": views(upcoming), "recent": views(recent),
			"zone": arenaZoneName, "minPlayers": minArenaPlayers,
		})
	})

	tournament := func(w http.ResponseWriter, r *http.Request) *tournamentRow {
		id, _ := strconv.ParseInt(r.PathValue("id"), 10, 64)
		row, err := s.store.Tournament(id)
		if serverError(w, err) {
			return nil
		}
		if row == nil {
			jsonError(w, "No such tournament.", http.StatusNotFound)
			return nil
		}
		return row
	}

	mux.HandleFunc("GET /api/tournaments/{id}", func(w http.ResponseWriter, r *http.Request) {
		row := tournament(w, r)
		if row == nil {
			return
		}
		writeJSON(w, t.view(*row, s.auth.user(r)))
	})

	mux.HandleFunc("POST /api/tournaments/{id}/join", func(w http.ResponseWriter, r *http.Request) {
		me := s.signedIn(w, r)
		if me == nil {
			return
		}
		row := tournament(w, r)
		if row == nil {
			return
		}
		if row.Status != "scheduled" && row.Status != "running" {
			jsonError(w, "That tournament is over.", http.StatusBadRequest)
			return
		}
		if serverError(w, t.join(row, me)) {
			return
		}
		writeJSON(w, t.view(*row, me))
	})

	mux.HandleFunc("DELETE /api/tournaments/{id}/join", func(w http.ResponseWriter, r *http.Request) {
		me := s.signedIn(w, r)
		if me == nil {
			return
		}
		row := tournament(w, r)
		if row == nil {
			return
		}
		if serverError(w, t.leave(row, me)) {
			return
		}
		writeJSON(w, t.view(*row, me))
	})
}
