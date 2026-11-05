package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"math/rand/v2"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"chess/four"
	"github.com/coder/websocket"
)

// Four-player chess is a mods game of its own: four seats on the cross-shaped
// board, played clockwise, last king standing wins. It has its own matches,
// table and socket, and leaves the two-player Match alone.

// fourSettings are the options a four-player game can have: the mods that make
// sense with four players, and a clock. Eating the king is always the rule.
type fourSettings struct {
	Fog          bool        `json:"fog"`
	Time         TimeControl `json:"time"`
	JumpKing     bool        `json:"jumpKing,omitempty"`
	BackPawns    bool        `json:"backPawns,omitempty"`
	JesterPawns  bool        `json:"jesterPawns,omitempty"`
	Hill         bool        `json:"hill,omitempty"`
	CenterVision string      `json:"centerVision,omitempty"`
}

func (s fourSettings) rules() four.Rules {
	return four.Rules{JumpKing: s.JumpKing, BackPawns: s.BackPawns, JesterPawns: s.JesterPawns, Hill: s.Hill,
		CenterVision: four.CenterVision(centerVisions[s.CenterVision])}
}

func (s fourSettings) valid() bool {
	_, center := centerVisions[s.CenterVision]
	return center && (s.Time.valid() || s.Time.modsValid())
}

// fourProtocol changes whenever the four-player state message changes shape.
// Must match FOUR_PROTOCOL in web/src/useFourMatch.ts.
const fourProtocol = 1

// fourBotName is who fills an empty seat.
const fourBotName = "Rex inutilis"

// fourSpectatorDelay is how many events spectators of a fog game are kept
// behind: two rounds of four.
const fourSpectatorDelay = 8

// FourHub holds the four-player matches in play, and saves them.
type FourHub struct {
	mu      sync.Mutex
	matches map[string]*FourMatch
	db      *sql.DB // nil keeps games only in memory, as in tests
	writes  chan fourRecord
	// botPause is how long a bot waits before moving, so its moves can be
	// seen one by one. Tests set it to nothing.
	botPause func() time.Duration
}

func NewFourHub(store *Store) *FourHub {
	h := &FourHub{matches: map[string]*FourMatch{}, writes: make(chan fourRecord, 1024),
		botPause: func() time.Duration { return time.Duration(350+rand.IntN(500)) * time.Millisecond }}
	if store != nil {
		h.db = store.db
	}
	go h.writer()
	go h.sweep()
	return h
}

func (h *FourHub) Create(creator *player, settings fourSettings) *FourMatch {
	h.mu.Lock()
	defer h.mu.Unlock()
	for {
		id := newID(6)
		if _, taken := h.matches[id]; taken {
			continue
		}
		if rec, _ := h.load(id); rec != nil {
			continue
		}
		now := time.Now()
		m := &FourMatch{hub: h, id: id, settings: settings, game: four.NewGame(settings.rules()),
			clients: map[*client]struct{}{}, created: now, touched: now}
		m.resetClock()
		m.seat(four.Red, creator)
		m.save()
		h.matches[id] = m
		return m
	}
}

// Get returns a match from memory or the database; nil if there's none.
func (h *FourHub) Get(id string) *FourMatch {
	h.mu.Lock()
	defer h.mu.Unlock()
	if m := h.matches[id]; m != nil {
		return m
	}
	rec, err := h.load(id)
	if err != nil || rec == nil {
		return nil
	}
	m, err := restoreFour(h, rec)
	if err != nil {
		log.Printf("restoring four-player match %s: %v", id, err)
		return nil
	}
	h.matches[id] = m
	return m
}

// sweep drops matches nobody has had open for a while; they load again on demand.
func (h *FourHub) sweep() {
	for range time.Tick(5 * time.Minute) {
		h.mu.Lock()
		for id, m := range h.matches {
			m.mu.Lock()
			idle := len(m.clients) == 0 && time.Since(m.touched) > 2*time.Hour && h.db != nil
			if idle {
				m.stopTimer()
			}
			m.mu.Unlock()
			if idle {
				delete(h.matches, id)
			}
		}
		h.mu.Unlock()
	}
}

// fourRecord is a match as saved.
type fourRecord struct {
	ID        string
	Settings  fourSettings
	Seats     [4]string
	Users     [4]int64
	Names     [4]string
	Moves     string
	Result    string
	Reason    string
	Clock     [4]time.Duration
	TurnStart time.Time
	Created   time.Time
	Updated   time.Time
	// done, when set, makes this record a marker that is closed once every
	// write before it is done; see flush.
	done chan struct{}
}

// flush waits until every queued write is in the database.
func (h *FourHub) flush() {
	done := make(chan struct{})
	h.writes <- fourRecord{done: done}
	<-done
}

func (h *FourHub) writer() {
	for r := range h.writes {
		if r.done != nil {
			close(r.done)
			continue
		}
		if h.db == nil {
			continue
		}
		settings, _ := json.Marshal(r.Settings)
		seats, _ := json.Marshal(r.Seats)
		names, _ := json.Marshal(r.Names)
		users, _ := json.Marshal(r.Users)
		var ms [4]int64
		for i, d := range r.Clock {
			ms[i] = d.Milliseconds()
		}
		clock, _ := json.Marshal(ms)
		var turnStart any
		if !r.TurnStart.IsZero() {
			turnStart = r.TurnStart
		}
		_, err := h.db.Exec(`
			INSERT INTO four_games (match_id, settings, seats, users, names, moves, result, reason, clock_ms, turn_started_at, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
			ON CONFLICT (match_id) DO UPDATE SET
				seats = excluded.seats, users = excluded.users, names = excluded.names, moves = excluded.moves,
				result = excluded.result, reason = excluded.reason, clock_ms = excluded.clock_ms,
				turn_started_at = excluded.turn_started_at, updated_at = excluded.updated_at`,
			r.ID, string(settings), string(seats), string(users), string(names), r.Moves, r.Result, r.Reason,
			string(clock), turnStart, r.Created, r.Updated)
		if err != nil {
			log.Printf("saving four-player game %s: %v", r.ID, err)
		}
	}
}

// load reads a saved match; nil if there's none.
func (h *FourHub) load(id string) (*fourRecord, error) {
	if h.db == nil {
		return nil, nil
	}
	var r fourRecord
	var settings, seats, names, users, clock string
	var turnStart sql.NullTime
	err := h.db.QueryRow(`SELECT match_id, settings, seats, users, names, moves, result, reason, clock_ms, turn_started_at, created_at, updated_at
		FROM four_games WHERE match_id = $1`, id).
		Scan(&r.ID, &settings, &seats, &users, &names, &r.Moves, &r.Result, &r.Reason, &clock, &turnStart, &r.Created, &r.Updated)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(settings), &r.Settings)
	json.Unmarshal([]byte(seats), &r.Seats)
	json.Unmarshal([]byte(names), &r.Names)
	json.Unmarshal([]byte(users), &r.Users)
	var ms [4]int64
	json.Unmarshal([]byte(clock), &ms)
	for i, n := range ms {
		r.Clock[i] = time.Duration(n) * time.Millisecond
	}
	if turnStart.Valid {
		r.TurnStart = turnStart.Time
	}
	return &r, nil
}

func restoreFour(h *FourHub, r *fourRecord) (*FourMatch, error) {
	m := &FourMatch{hub: h, id: r.ID, settings: r.Settings, game: four.NewGame(r.Settings.rules()),
		clients: map[*client]struct{}{}, created: r.Created, touched: time.Now(),
		seats: r.Seats, users: r.Users, names: r.Names, clock: r.Clock, turnStart: r.TurnStart}
	for i, key := range r.Seats {
		m.bots[i] = strings.HasPrefix(key, "bot:")
	}
	for _, tok := range strings.Fields(r.Moves) {
		if err := m.game.PlayToken(tok); err != nil {
			return nil, err
		}
	}
	if !m.game.Over() && r.Result == "draw" {
		m.game.AgreeDraw()
	}
	if m.game.Over() {
		m.turnStart = time.Time{}
	}
	if m.running() {
		m.mu.Lock()
		if !m.flagged(time.Now()) {
			m.armTimer()
		}
		m.mu.Unlock()
	}
	m.mu.Lock()
	m.pokeBot()
	m.mu.Unlock()
	return m, nil
}

// FourMatch is one four-player game.
type FourMatch struct {
	mu       sync.Mutex
	hub      *FourHub
	id       string
	settings fourSettings
	game     *four.Game
	seats    [4]string // player keys, Red Blue Green Violet; "" while free
	users    [4]int64
	names    [4]string
	bots     [4]bool
	clients  map[*client]struct{}
	created  time.Time
	touched  time.Time
	// clock is each seat's time left as of turnStart, when the player to
	// move's clock started; a zero turnStart means no clock is running.
	clock       [4]time.Duration
	turnStart   time.Time
	timer       *time.Timer
	botThinking bool
	chat        []chatLine
	chatID      int
}

func (m *FourMatch) seat(c four.Color, p *player) {
	m.seats[c], m.users[c], m.names[c] = p.key, p.userID, p.name
	m.bots[c] = strings.HasPrefix(p.key, "bot:")
}

// full is whether all four seats are taken, which is when the game starts.
func (m *FourMatch) full() bool { return !slices.Contains(m.seats[:], "") }

func (m *FourMatch) colorOf(key string) int {
	for i, s := range m.seats {
		if s != "" && s == key {
			return i
		}
	}
	return -1
}

func (m *FourMatch) online(c int) bool {
	if m.bots[c] {
		return true
	}
	for cl := range m.clients {
		if cl.playerID == m.seats[c] {
			return true
		}
	}
	return false
}

// save queues the match for the database. Callers hold m.mu.
func (m *FourMatch) save() {
	m.hub.writes <- fourRecord{ID: m.id, Settings: m.settings, Seats: m.seats, Users: m.users, Names: m.names,
		Moves: m.game.Tokens(), Result: m.game.Result(), Reason: m.game.Method().String(),
		Clock: m.clock, TurnStart: m.turnStart, Created: m.created, Updated: time.Now()}
}

func (m *FourMatch) resetClock() {
	t := m.settings.Time
	start := time.Duration(t.Base+t.PerMove) * time.Second
	m.clock = [4]time.Duration{start, start, start, start}
	m.turnStart = time.Time{}
	m.stopTimer()
}

func (m *FourMatch) running() bool { return !m.turnStart.IsZero() }

func (m *FourMatch) left(c int, now time.Time) time.Duration {
	d := m.clock[c]
	if m.running() && int(m.game.Turn()) == c {
		d -= now.Sub(m.turnStart)
	}
	return d
}

func (m *FourMatch) armTimer() {
	m.stopTimer()
	m.timer = time.AfterFunc(max(m.left(int(m.game.Turn()), time.Now()), 0), m.checkFlag)
}

func (m *FourMatch) stopTimer() {
	if m.timer != nil {
		m.timer.Stop()
		m.timer = nil
	}
}

// startTurn starts the clock of the player to move, if the game has one and is
// on. Callers hold m.mu.
func (m *FourMatch) startTurn(now time.Time) {
	m.turnStart = time.Time{}
	m.stopTimer()
	if !m.settings.Time.clocked() || !m.full() || m.game.Over() {
		return
	}
	if pm := m.settings.Time.PerMove; pm > 0 {
		m.clock[m.game.Turn()] = time.Duration(pm) * time.Second
	}
	m.turnStart = now
	m.armTimer()
}

func (m *FourMatch) checkFlag() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.flagged(time.Now()) {
		m.save()
		m.pokeBot()
		m.broadcast()
	} else if m.running() {
		m.armTimer()
	}
}

// flagged takes the player to move out of the game if their time is up, and
// reports whether it did. Callers hold m.mu.
func (m *FourMatch) flagged(now time.Time) bool {
	if m.game.Over() || !m.running() {
		return false
	}
	c := int(m.game.Turn())
	if m.left(c, now) > 0 {
		return false
	}
	m.clock[c] = 0
	m.game.Leave(four.Color(c), four.Timeout)
	m.startTurn(now)
	return true
}

// join seats the player in the first free seat while the game hasn't
// started; anyone else watches.
func (m *FourMatch) join(c *client, p *player) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.colorOf(p.key) < 0 {
		if i := m.colorOf(p.guest); i >= 0 && p.userID != 0 {
			m.seat(four.Color(i), p)
			m.save()
		} else if i := slices.Index(m.seats[:], ""); i >= 0 {
			m.seat(four.Color(i), p)
			if m.full() {
				m.startTurn(time.Now())
			}
			m.save()
		}
	}
	m.clients[c] = struct{}{}
	m.touched = time.Now()
	if m.colorOf(c.playerID) >= 0 {
		c.send(m.chatJSON(true, m.chat))
	}
	m.pokeBot()
	m.broadcast()
}

func (m *FourMatch) leave(c *client) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.clients, c)
	m.touched = time.Now()
	m.broadcast()
}

type fourIn struct {
	Type string  `json:"type"`
	UCI  string  `json:"uci"`
	Text string  `json:"text"`
	Side string  `json:"side"`
	T    float64 `json:"t"`
}

func (m *FourMatch) handle(c *client, msg fourIn) {
	m.mu.Lock()
	defer m.mu.Unlock()
	color := m.colorOf(c.playerID)
	if color < 0 {
		if side, ok := four.ParseColor(msg.Side); msg.Type == "watch" && ok && c.side != int(side) {
			c.side = int(side)
			c.push(m.state(c))
		}
		return
	}
	now := time.Now()
	m.touched = now
	events := len(m.game.History())
	switch msg.Type {
	case "fill":
		// Bots take the empty seats, and the game starts.
		if m.full() {
			return
		}
		for i, s := range m.seats {
			if s == "" {
				m.seat(four.Color(i), &player{key: "bot:four:" + four.Color(i).String(), name: fourBotName})
			}
		}
		m.startTurn(now)
		m.save()
	case "move":
		if !m.full() || m.game.Over() || int(m.game.Turn()) != color || m.flagged(now) {
			break
		}
		m.play(msg.UCI, now)
	case "resign":
		if !m.full() || m.game.Over() || m.game.Position().Out[color] {
			break
		}
		onTurn := int(m.game.Turn()) == color
		if onTurn && m.running() {
			m.clock[color] = max(m.left(color, now), 0)
		}
		m.game.Leave(four.Color(color), four.Resignation)
		if onTurn || m.game.Over() {
			m.startTurn(now)
		}
	case "chat":
		text := strings.TrimSpace(msg.Text)
		if text == "" || !utf8.ValidString(text) {
			return
		}
		if utf8.RuneCountInString(text) > chatMaxLen {
			text = string([]rune(text)[:chatMaxLen])
		}
		m.chatID++
		line := chatLine{ID: m.chatID, From: four.Color(color).String(), Text: text, At: now.UnixMilli()}
		m.chat = append(m.chat, line)
		if len(m.chat) > chatMaxLines {
			m.chat = slices.Clone(m.chat[len(m.chat)-chatMaxLines:])
		}
		b := m.chatJSON(false, []chatLine{line})
		for cl := range m.clients {
			if m.colorOf(cl.playerID) >= 0 {
				cl.send(b)
			}
		}
		return
	}
	if len(m.game.History()) != events {
		m.save()
	}
	m.pokeBot()
	m.broadcast()
}

// play plays the move of the player to move and runs the clocks. Callers hold m.mu.
func (m *FourMatch) play(uci string, now time.Time) bool {
	c := int(m.game.Turn())
	left := m.left(c, now)
	if _, err := m.game.PlayUCI(uci); err != nil {
		return false
	}
	if m.running() {
		m.clock[c] = left + time.Duration(m.settings.Time.Inc)*time.Second
	}
	m.startTurn(now)
	return true
}

// pokeBot has a bot move after a short pause when it's a bot's turn. Callers hold m.mu.
func (m *FourMatch) pokeBot() {
	if !m.full() || m.game.Over() || !m.bots[m.game.Turn()] || m.botThinking {
		return
	}
	m.botThinking = true
	events := len(m.game.History())
	go func() {
		time.Sleep(m.hub.botPause())
		m.mu.Lock()
		defer m.mu.Unlock()
		m.botThinking = false
		if m.game.Over() || len(m.game.History()) != events || !m.bots[m.game.Turn()] {
			m.pokeBot()
			return
		}
		pos := m.game.Position()
		if mv, ok := four.BotMove(&pos, rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))); ok {
			m.play(mv.UCI(), time.Now())
			m.save()
		}
		m.pokeBot()
		m.broadcast()
	}()
}

func (m *FourMatch) chatJSON(reset bool, lines []chatLine) []byte {
	if lines == nil {
		lines = []chatLine{}
	}
	b, _ := json.Marshal(chatMsg{Type: "chat", Reset: reset, Lines: lines})
	return b
}

type fourSeat struct {
	Taken  bool   `json:"taken"`
	Online bool   `json:"online"`
	Name   string `json:"name,omitempty"`
	Bot    bool   `json:"bot,omitempty"`
	// Out is how the player left the game: "capturing the king", "resignation" or "time".
	Out string `json:"out,omitempty"`
}

type fourEvent struct {
	By  string `json:"by"`
	SAN string `json:"san"` // "?" for a move the viewer can't know
	Out string `json:"out,omitempty"`
}

type fourView struct {
	// Board is every square from the bottom left, row by row: " " off the
	// board, "." empty, "?" hidden, or the piece's colour and kind, like "rK".
	Board        []string            `json:"board"`
	Turn         string              `json:"turn"`
	Ply          int                 `json:"ply"`
	Legal        []string            `json:"legal"`
	Last         *lastMove           `json:"last"`
	Danger       string              `json:"danger,omitempty"`
	DangerHidden bool                `json:"dangerHidden,omitempty"`
	FullVision   bool                `json:"fullVision,omitempty"`
	History      []fourEvent         `json:"history"`
	Captured     map[string][]string `json:"captured"`
	Behind       int                 `json:"behind,omitempty"`
}

type fourState struct {
	Type     string       `json:"type"`
	Protocol int          `json:"protocol"`
	ID       string       `json:"id"`
	You      string       `json:"you"`
	Settings fourSettings `json:"settings"`
	fourView
	Status     string              `json:"status"` // waiting, playing, over
	Result     string              `json:"result,omitempty"`
	Reason     string              `json:"reason,omitempty"`
	Players    map[string]fourSeat `json:"players"`
	Clock      *clockView          `json:"clock,omitempty"`
	Watching   string              `json:"watching,omitempty"`
	Spectators int                 `json:"spectators"`
}

// fourViewFor is the game as viewer sees it (-1 sees everything). Under fog a
// player sees what their pieces see, their own moves, and who is out, but not
// the other moves; a player who is out sees everything.
func fourViewFor(g *four.Game, viewer int, playable, fog bool) fourView {
	pos := g.Position()
	hist := g.History()
	v := fourView{Turn: pos.Turn.String(), Ply: len(hist), Legal: []string{}, History: make([]fourEvent, len(hist)),
		Captured: map[string][]string{}, Board: make([]string, four.NumSquares)}
	hidden := fog && !g.Over() && viewer >= 0 && !pos.Out[viewer]
	var seen *[four.NumSquares]bool
	if hidden {
		vision := pos.Vision(four.Color(viewer))
		seen = &vision
		v.FullVision = pos.Outpost(four.Color(viewer)) != four.NoSquare
	}
	for i, pc := range pos.Board {
		sq := four.Square(i)
		switch {
		case !sq.Valid():
			v.Board[i] = " "
		case seen != nil && !seen[i]:
			v.Board[i] = "?"
		case pc.IsEmpty():
			v.Board[i] = "."
		default:
			v.Board[i] = pc.Code()
		}
	}
	if playable && !g.Over() && viewer == int(pos.Turn) {
		for _, mv := range g.LegalMoves() {
			v.Legal = append(v.Legal, mv.UCI())
		}
	}
	if viewer >= 0 && !g.Over() && !pos.Out[viewer] {
		if k := pos.KingSquare(four.Color(viewer)); k != four.NoSquare && pos.Attacked(k, four.Color(viewer), nil) {
			v.Danger = k.String()
			v.DangerHidden = seen != nil && !pos.Attacked(k, four.Color(viewer), seen)
		}
	}
	for _, c := range four.Colors {
		v.Captured[c.String()] = []string{}
	}
	for i, e := range hist {
		mine := !hidden || int(e.By) == viewer
		v.History[i] = fourEvent{By: e.By.String(), SAN: "?"}
		if e.Out != four.NoMethod {
			v.History[i] = fourEvent{By: e.By.String(), Out: e.Out.String()}
			continue
		}
		if mine {
			v.History[i].SAN = e.SAN
			if !e.Captured.IsEmpty() {
				v.Captured[e.By.String()] = append(v.Captured[e.By.String()], e.Captured.Code())
			}
		}
	}
	// The last move the viewer knows: the last one played, or under fog their own.
	for i := len(hist) - 1; i >= 0; i-- {
		if hist[i].Out == four.NoMethod && (!hidden || int(hist[i].By) == viewer) {
			v.Last = &lastMove{From: hist[i].Move.From.String(), To: hist[i].Move.To.String()}
			break
		}
	}
	return v
}

// base is the part of the state that's the same for everyone. Callers hold m.mu.
func (m *FourMatch) base() fourState {
	s := fourState{Type: "state", Protocol: fourProtocol, ID: m.id, Settings: m.settings, Status: "playing",
		Players: map[string]fourSeat{}}
	switch {
	case m.game.Over():
		s.Status, s.Result, s.Reason = "over", m.game.Result(), m.game.Method().String()
	case !m.full():
		s.Status = "waiting"
	}
	for _, c := range four.Colors {
		seat := fourSeat{Taken: m.seats[c] != "", Name: m.names[c], Bot: m.bots[c]}
		seat.Online = seat.Taken && m.online(int(c))
		if f := m.game.Fell(c); f != four.NoMethod {
			seat.Out = f.String()
		}
		s.Players[c.String()] = seat
	}
	if m.settings.Time.clocked() {
		now := time.Now()
		s.Clock = &clockView{Left: map[string]int64{}}
		for _, c := range four.Colors {
			s.Clock.Left[c.String()] = max(m.left(int(c), now), 0).Milliseconds()
		}
		if m.running() {
			s.Clock.Running = m.game.Turn().String()
		}
	}
	seen := map[string]bool{}
	for cl := range m.clients {
		if m.colorOf(cl.playerID) < 0 {
			seen[cl.playerID] = true
		}
	}
	s.Spectators = len(seen)
	return s
}

// view fills in the state for a player (color 0 to 3) or a spectator who
// watches as side. Callers hold m.mu.
func (m *FourMatch) view(s fourState, color, side int) []byte {
	if color >= 0 {
		s.You = four.Color(color).String()
		s.fourView = fourViewFor(m.game, color, s.Status == "playing", m.settings.Fog)
	} else {
		s.Watching = four.Color(side).String()
		s.fourView = m.spectatorView(side)
	}
	b, _ := json.Marshal(s)
	return b
}

// spectatorView keeps spectators of a fog game a couple of rounds behind, seeing
// what side saw then, so nobody watching can tell a player what's hidden.
func (m *FourMatch) spectatorView(side int) fourView {
	if !m.settings.Fog || m.game.Over() {
		return fourViewFor(m.game, -1, false, m.settings.Fog)
	}
	hist := m.game.History()
	shown := max(0, len(hist)-fourSpectatorDelay)
	past := four.NewGame(m.settings.rules())
	for _, e := range hist[:shown] {
		past.PlayToken(e.Token())
	}
	v := fourViewFor(past, side, false, true)
	v.Turn, v.Ply, v.Behind = m.game.Turn().String(), len(hist), len(hist)-shown
	return v
}

func (m *FourMatch) state(c *client) []byte {
	return m.view(m.base(), m.colorOf(c.playerID), c.side)
}

// broadcast sends every client its own view. Callers hold m.mu.
func (m *FourMatch) broadcast() {
	base := m.base()
	views := map[int][]byte{}
	for c := range m.clients {
		slot := m.colorOf(c.playerID)
		if slot < 0 {
			slot = 4 + c.side
		}
		if views[slot] == nil {
			views[slot] = m.view(base, m.colorOf(c.playerID), c.side)
		}
		c.push(views[slot])
	}
}

// serve runs one websocket connection until it closes.
func (m *FourMatch) serve(ctx context.Context, conn *websocket.Conn, p *player) {
	c := newClient(p.key)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		ping := time.NewTicker(25 * time.Second)
		defer ping.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-c.wake:
				for _, b := range c.drain() {
					wctx, wcancel := context.WithTimeout(ctx, 10*time.Second)
					err := conn.Write(wctx, websocket.MessageText, b)
					wcancel()
					if err != nil {
						cancel()
						return
					}
				}
			case <-ping.C:
				if err := conn.Ping(ctx); err != nil {
					cancel()
					return
				}
			}
		}
	}()
	m.join(c, p)
	defer m.leave(c)
	conn.SetReadLimit(4096)
	for {
		_, b, err := conn.Read(ctx)
		if err != nil {
			if !errors.Is(err, context.Canceled) {
				conn.Close(websocket.StatusNormalClosure, "")
			}
			return
		}
		var msg fourIn
		if json.Unmarshal(b, &msg) != nil {
			continue
		}
		if msg.Type == "ping" {
			pong, _ := json.Marshal(map[string]any{"type": "pong", "t": msg.T})
			c.send(pong)
			continue
		}
		m.handle(c, msg)
	}
}
