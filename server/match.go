package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"chess/engine"
	"github.com/coder/websocket"
)

// idAlphabet skips look-alike characters (0/O, 1/I/L) so IDs are easy to read aloud.
const idAlphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"

const (
	white = 0
	black = 1
)

var colorNames = [2]string{"w", "b"}

// Hub holds the matches in play. Every match is also saved to the store, so one
// that was dropped from memory, or lived through a restart, loads again on demand.
type Hub struct {
	mu      sync.Mutex
	matches map[string]*Match
	store   *Store
	hooks   matchHooks
}

// matchHooks tell the rest of the site what happens in matches.
type matchHooks struct {
	// notify, if set, puts a game result or a slow move in a player's bell.
	notify func(notification)
	// playing, if set, is told when a signed-in player starts or stops playing
	// a game, so friends see them in a game.
	playing func(userID int64)
}

func NewHub(store *Store) *Hub {
	h := &Hub{matches: make(map[string]*Match), store: store}
	go h.sweep()
	go h.flagExpired()
	return h
}

func (h *Hub) Create(creator *player, settings Settings) *Match {
	h.mu.Lock()
	defer h.mu.Unlock()
	for {
		id := newID(6)
		if _, taken := h.matches[id]; taken {
			continue
		}
		if g, _ := h.store.LatestGame(id); g != nil {
			continue
		}
		m := newMatch(h.store, id, settings)
		m.hooks = h.hooks
		m.seat(white, creator)
		m.save()
		h.matches[id] = m
		return m
	}
}

// SetHooks sets where game results, slow moves and who's playing go.
func (h *Hub) SetHooks(hooks matchHooks) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.hooks = hooks
}

// CreateBetween starts a match from an accepted offer, both seats taken. The
// clock starts once both players have it open.
func (h *Hub) CreateBetween(w, b *player, settings Settings) *Match {
	m := h.Create(w, settings)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.seat(black, b)
	m.turnBegan = time.Now()
	m.save()
	return m
}

// Get returns a match from memory, or loads it from the store; nil if there's no such match.
func (h *Hub) Get(id string) *Match {
	h.mu.Lock()
	defer h.mu.Unlock()
	if m := h.matches[id]; m != nil {
		return m
	}
	rec, err := h.store.LatestGame(id)
	if err != nil || rec == nil {
		return nil
	}
	m, err := restoreMatch(h.store, rec, h.hooks)
	if err != nil {
		log.Printf("restoring match %s: %v", id, err)
		return nil
	}
	h.matches[id] = m
	return m
}

// sweep drops matches nobody has been connected to for a while.
func (h *Hub) sweep() {
	for range time.Tick(5 * time.Minute) {
		h.mu.Lock()
		for id, m := range h.matches {
			m.mu.Lock()
			idle := len(m.clients) == 0 && time.Since(m.touched) > 2*time.Hour
			if idle && m.timer != nil {
				// flagExpired catches this game running out of time from now on.
				m.timer.Stop()
				m.timer = nil
			}
			m.mu.Unlock()
			if idle {
				delete(h.matches, id)
			}
		}
		h.mu.Unlock()
	}
}

// flagExpired ends games whose clock ran out while nobody had them loaded, like
// a Daily game left alone for a day, or any game when the server restarts.
func (h *Hub) flagExpired() {
	for {
		ids, err := h.store.ExpiredGames()
		if err != nil {
			log.Printf("looking for games out of time: %v", err)
		}
		for _, id := range ids {
			if m := h.Get(id); m != nil {
				m.checkFlag()
			}
		}
		time.Sleep(30 * time.Second)
	}
}

func newID(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	for i := range b {
		b[i] = idAlphabet[int(b[i])%len(idAlphabet)]
	}
	return string(b)
}

// player is who is behind a connection: a signed-in user, or a guest known by a
// random ID their browser keeps.
type player struct {
	key    string // seat key: "u:<user id>" or the guest's browser ID
	userID int64  // 0 for guests
	name   string // username, empty for guests
	guest  string // the browser ID, so a guest who signs in keeps their seat
}

func guestPlayer(browserID string) *player { return &player{key: browserID, guest: browserID} }

func userPlayer(u *User, browserID string) *player {
	return &player{key: "u:" + strconv.FormatInt(u.ID, 10), userID: u.ID, name: u.Username, guest: browserID}
}

type client struct {
	playerID string
	side     int // which side a spectator sees the game as; guarded by the match's mu
	mu       sync.Mutex
	latest   []byte        // newest unsent state; older unsent states are dropped
	queue    [][]byte      // ping replies and chat, always sent in order
	wake     chan struct{} // signals the writer that something is queued
}

func newClient(playerID string) *client {
	return &client{playerID: playerID, wake: make(chan struct{}, 1)}
}

// push queues a state message. A slow client skips straight to the newest state
// instead of falling behind or missing one.
func (c *client) push(b []byte) {
	c.mu.Lock()
	c.latest = b
	c.mu.Unlock()
	c.signal()
}

// send queues a message that must not be skipped, unlike a state.
func (c *client) send(b []byte) {
	c.mu.Lock()
	c.queue = append(c.queue, b)
	c.mu.Unlock()
	c.signal()
}

func (c *client) signal() {
	select {
	case c.wake <- struct{}{}:
	default:
	}
}

func (c *client) drain() [][]byte {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := c.queue
	c.queue = nil
	if c.latest != nil {
		out = append(out, c.latest)
		c.latest = nil
	}
	return out
}

type Match struct {
	mu       sync.Mutex
	store    *Store
	id       string
	settings Settings
	game     *engine.Game
	seats    [2]string // player keys holding white and black
	users    [2]int64  // user IDs of signed-in players, 0 for guests
	names    [2]string // their usernames
	clients  map[*client]struct{}
	rematch  [2]bool
	draw     [2]bool
	// setUp is whether this round's pieces are in place. In a setup mode it's
	// false until both players have sent their arrangement; game until then is
	// only a placeholder.
	setUp bool
	// placed is each player's arrangement while setting up, "" until they're ready.
	placed  [2]string
	round   int
	started time.Time // when this round began
	touched time.Time
	// clock is each side's time left as of turnStart, when the side to move's
	// clock started. A zero turnStart means no clock is running: there's no
	// clock, the opponent hasn't joined yet, or the game is over.
	clock      [2]time.Duration
	turnStart  time.Time
	timer      *time.Timer // ends the game when the side to move runs out
	ratingDiff *[2]int     // set once a rated game's ratings are saved
	// turnBegan is when the side to move got the turn, clock or not.
	turnBegan time.Time
	hooks     matchHooks
	// overNotified is the round whose result has gone to the players' bells.
	overNotified int
	// chat between the two players in this round. Kept only in memory, and
	// cleared when the next round starts or both players leave a finished game.
	chat   []chatLine
	chatID int
}

const (
	chatMaxLen   = 500 // characters per message
	chatMaxLines = 200 // older lines are dropped
)

type chatLine struct {
	ID   int    `json:"id"`
	From string `json:"from"` // "w" or "b"
	Text string `json:"text"`
	At   int64  `json:"at"` // unix milliseconds
}

type chatMsg struct {
	Type  string     `json:"type"`
	Reset bool       `json:"reset,omitempty"` // replace the chat instead of adding to it
	Lines []chatLine `json:"lines"`
}

func newMatch(store *Store, id string, settings Settings) *Match {
	now := time.Now()
	m := &Match{
		store: store, id: id, settings: settings, game: engine.NewGameWithRules(settings.rules()),
		clients: make(map[*client]struct{}), round: 1, started: now, touched: now,
		setUp: settings.Setup == "",
	}
	m.resetClock()
	return m
}

// resetClock puts both clocks at the starting time, stopped.
func (m *Match) resetClock() {
	t := m.settings.Time
	start := time.Duration(t.Base+t.PerMove) * time.Second
	m.clock = [2]time.Duration{start, start}
	m.turnStart = time.Time{}
	m.stopTimer()
}

// restoreMatch rebuilds a match from its newest saved round by replaying the moves.
func restoreMatch(store *Store, rec *gameRecord, hooks matchHooks) (*Match, error) {
	m := newMatch(store, rec.MatchID, rec.Settings)
	m.hooks = hooks
	m.round, m.started, m.turnBegan = rec.Round, rec.CreatedAt, rec.UpdatedAt
	m.seats, m.users = rec.Seats, rec.Users
	// Setups are kept only in memory, so a setup that didn't finish starts over.
	m.setUp = rec.Settings.Setup == "" || rec.StartFEN != ""
	m.names = [2]string{rec.whiteName, rec.blackName}
	g, err := replay(rec)
	if err != nil {
		return nil, err
	}
	m.game = g
	if g.Over() {
		m.overNotified = m.round
	}
	m.clock, m.turnStart = rec.Clock, rec.TurnStart
	if d := rec.RatingDiff; d[white] != nil && d[black] != nil {
		m.ratingDiff = &[2]int{*d[white], *d[black]}
	}
	if m.running() {
		if m.flagged(time.Now()) {
			m.save()
		} else {
			m.armTimer()
		}
	}
	return m, nil
}

// replay plays a saved game's moves and applies a result the moves don't show
// (a resignation or an agreed draw).
func replay(rec *gameRecord) (*engine.Game, error) {
	g, err := engine.NewGameFromFEN(startFrom(rec.StartFEN), rec.Settings.rules())
	if err != nil {
		return nil, err
	}
	for _, uci := range strings.Fields(rec.Moves) {
		if _, err := g.PlayUCI(uci); err != nil {
			return nil, fmt.Errorf("move %s: %w", uci, err)
		}
	}
	if !g.Over() && rec.Result != "" {
		switch {
		case rec.Reason == engine.Timeout.String() || rec.Reason == engine.TimeoutVsInsufficientMaterial.String():
			g.TimeOut()
		case rec.Reason == engine.Agreement.String():
			g.AgreeDraw()
		case rec.Result == engine.WhiteWins.String():
			g.Resign(engine.Black)
		case rec.Result == engine.BlackWins.String():
			g.Resign(engine.White)
		}
	}
	return g, nil
}

// seat puts a player in a seat. Callers hold m.mu (or own the match exclusively).
func (m *Match) seat(color int, p *player) {
	m.seats[color], m.users[color], m.names[color] = p.key, p.userID, p.name
}

// save queues this round for the store. Callers hold m.mu.
func (m *Match) save() {
	var moves strings.Builder
	for i, pm := range m.game.History() {
		if i > 0 {
			moves.WriteByte(' ')
		}
		moves.WriteString(pm.Move.UCI())
	}
	rec := gameRecord{
		MatchID: m.id, Round: m.round, Settings: m.settings, Seats: m.seats, Users: m.users, StartFEN: m.startFEN(),
		Moves: moves.String(), CreatedAt: m.started, UpdatedAt: time.Now(),
		Clock: m.clock, TurnStart: m.turnStart, Rated: m.rated(),
	}
	round := m.round
	rec.onRated = func(diff [2]int) {
		// Called from the store's writer, which must not wait on this match.
		go func() {
			m.mu.Lock()
			defer m.mu.Unlock()
			if m.round == round {
				m.ratingDiff = &diff
				m.broadcast()
			}
		}()
	}
	if m.game.Over() {
		rec.Result, rec.Reason = m.game.Outcome().String(), m.game.Method().String()
		if m.overNotified != m.round {
			m.overNotified = m.round
			m.notifyOver()
			m.playingChanged()
		}
	}
	m.store.SaveGame(rec)
}

// startFEN is the start position to save: the players' setups, or "" for the usual one.
func (m *Match) startFEN() string {
	if m.settings.Setup == "" || !m.setUp {
		return ""
	}
	return m.game.StartFEN()
}

// slowMove is how long a move in a Daily or No clock game has to take before
// the opponent hears about it in their bell.
const slowMove = 4 * time.Hour

// notifyOver tells signed-in players who aren't watching how their game ended.
// Callers hold m.mu.
func (m *Match) notifyOver() {
	if m.hooks.notify == nil {
		return
	}
	outcome := m.game.Outcome().String()
	for c := range 2 {
		if m.users[c] == 0 || m.online(c) {
			continue
		}
		result := "draw"
		if outcome == "1-0" || outcome == "0-1" {
			result = "lost"
			if (outcome == "1-0") == (c == white) {
				result = "won"
			}
		}
		go m.hooks.notify(notification{
			UserID: m.users[c], Kind: notifyGameOver, ActorID: m.users[1-c], MatchID: m.id, Round: m.round,
			Data: map[string]any{
				"result": result, "reason": m.game.Method().String(), "kingCapture": m.settings.KingCapture,
				"sniper": sniperWin(outcome, m.game.Method().String(), len(m.game.History())),
			},
		})
	}
}

// rated is whether this game counts toward rating: two different signed-in
// players, a clock, and not played as unrated.
func (m *Match) rated() bool {
	return !m.settings.Unrated && m.settings.Time.clocked() && m.users[white] != 0 && m.users[black] != 0 && m.users[white] != m.users[black]
}

func (m *Match) running() bool { return !m.turnStart.IsZero() }

// left is how much time c has at now.
func (m *Match) left(c int, now time.Time) time.Duration {
	d := m.clock[c]
	if m.running() && int(m.game.Turn()) == c {
		d -= now.Sub(m.turnStart)
	}
	return d
}

// startClock starts the side to move's clock once both players are seated.
// Callers hold m.mu.
func (m *Match) startClock(now time.Time) {
	if !m.settings.Time.clocked() || !m.setUp || m.game.Over() || m.seats[white] == "" || m.seats[black] == "" || m.running() {
		return
	}
	if pm := m.settings.Time.PerMove; pm > 0 {
		m.clock[m.game.Turn()] = time.Duration(pm) * time.Second
	}
	m.turnStart = now
	m.armTimer()
}

// stopClock charges the side to move for the time they used and stops the
// clocks. Callers hold m.mu.
func (m *Match) stopClock(now time.Time) {
	if !m.running() {
		return
	}
	c := int(m.game.Turn())
	m.clock[c] = max(m.left(c, now), 0)
	m.turnStart = time.Time{}
	m.stopTimer()
}

func (m *Match) armTimer() {
	m.stopTimer()
	m.timer = time.AfterFunc(max(m.left(int(m.game.Turn()), time.Now()), 0), m.checkFlag)
}

func (m *Match) stopTimer() {
	if m.timer != nil {
		m.timer.Stop()
		m.timer = nil
	}
}

// checkFlag ends the game if the side to move is out of time.
func (m *Match) checkFlag() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.flagged(time.Now()) {
		m.save()
		m.broadcast()
	} else if m.running() {
		m.armTimer() // woke a moment early
	}
}

// flagged ends the game if the side to move has run out of time, and reports
// whether it did. Callers hold m.mu.
func (m *Match) flagged(now time.Time) bool {
	if m.game.Over() || !m.running() || m.left(int(m.game.Turn()), now) > 0 {
		return false
	}
	m.stopClock(now)
	m.game.TimeOut()
	return true
}

type seatState struct {
	Taken  bool   `json:"taken"`
	Online bool   `json:"online"`
	Name   string `json:"name,omitempty"` // username; empty for a guest
}

// protocolVersion changes whenever the state message changes shape, so a web app
// talking to a server from another version can say so instead of breaking.
const protocolVersion = 9

type stateMsg struct {
	Type     string   `json:"type"`
	Protocol int      `json:"protocol"`
	ID       string   `json:"id"`
	Round    int      `json:"round"`
	You      string   `json:"you"`
	Settings Settings `json:"settings"`
	gameView
	Status  string               `json:"status"`
	Result  string               `json:"result,omitempty"`
	Reason  string               `json:"reason,omitempty"`
	Players map[string]seatState `json:"players"`
	Rematch map[string]bool      `json:"rematch"`
	Draw    map[string]bool      `json:"draw"`
	Clock   *clockView           `json:"clock,omitempty"` // nil for No clock games
	Rated   bool                 `json:"rated"`
	// RatingDiff is each side's rating change, once a rated game is over and saved.
	RatingDiff map[string]int `json:"ratingDiff,omitempty"`
	// Watching is the side a spectator sees the game as; empty for players.
	Watching string `json:"watching,omitempty"`
	// Spectators is how many people are watching, not counting the players.
	Spectators int `json:"spectators"`
	// Setup is who's ready while the players arrange their pieces.
	Setup *setupState `json:"setup,omitempty"`
	// Sniper marks a game won on the winner's first move; see sniperWin.
	Sniper bool `json:"sniper,omitempty"`
}

// setupState is the setup as one viewer may know it.
type setupState struct {
	Ready map[string]bool `json:"ready"`
	// Mine is the arrangement this player sent, so a reload picks it up.
	Mine string `json:"mine,omitempty"`
	// Theirs is the opponent's arrangement once they're ready, in games
	// without fog. Under fog you never see it.
	Theirs string `json:"theirs,omitempty"`
}

// clockView is the clocks as of the moment the state is sent; the browser
// counts the running one down from there.
type clockView struct {
	Left    map[string]int64 `json:"left"`              // milliseconds
	Running string           `json:"running,omitempty"` // whose clock is ticking
}

type inMsg struct {
	Type string `json:"type"`
	UCI  string `json:"uci"`
	Text string `json:"text"` // chat message
	// Board is a player's setup, 64 characters from a1 to h8; empty to take it back.
	Board string  `json:"board"`
	Side  string  `json:"side"` // "w" or "b": the side a spectator wants to see the game as
	T     float64 `json:"t"`    // ping timestamp, echoed back
}

// join seats the player if a seat is free (white first); anyone else watches.
// A guest who signs in mid-game keeps the seat their browser held.
func (m *Match) join(c *client, p *player) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.colorOf(p.key) < 0 {
		if i := m.colorOf(p.guest); i >= 0 && p.userID != 0 {
			m.seat(i, p)
			m.save()
		} else {
			for i := range m.seats {
				if m.seats[i] == "" {
					m.seat(i, p)
					m.turnBegan = time.Now()
					// White's clock starts as soon as an opponent sits down.
					m.startClock(time.Now())
					m.save()
					break
				}
			}
		}
	}
	m.clients[c] = struct{}{}
	m.touched = time.Now()
	// A match from an accepted offer starts its clock once both players are here.
	if !m.running() && m.online(white) && m.online(black) && len(m.game.History()) == 0 {
		m.startClock(time.Now())
		if m.running() {
			m.save()
		}
	}
	if m.colorOf(c.playerID) >= 0 {
		c.send(m.chatJSON(true, m.chat))
		m.playingChanged()
	}
	m.broadcast()
}

func (m *Match) leave(c *client) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.clients, c)
	m.touched = time.Now()
	if m.game.Over() && !m.online(white) && !m.online(black) {
		m.chat = nil
	}
	if i := m.colorOf(c.playerID); i >= 0 && !m.online(i) {
		m.playingChanged()
	}
	m.broadcast()
}

// playingChanged tells the site that whether this match's players are playing
// may have changed. Callers hold m.mu.
func (m *Match) playingChanged() {
	if m.hooks.playing == nil {
		return
	}
	for _, id := range m.users {
		if id != 0 {
			go m.hooks.playing(id)
		}
	}
}

// live is whether the match is being played right now: both seats taken, the
// game not over, and a player here. A game nobody has moved in yet needs both
// players, so a match from an offer isn't listed while it waits for one.
// Callers hold m.mu.
func (m *Match) live() bool {
	if m.seats[white] == "" || m.seats[black] == "" || m.game.Over() {
		return false
	}
	if len(m.game.History()) == 0 {
		return m.online(white) && m.online(black)
	}
	return m.online(white) || m.online(black)
}

// spectators counts the people watching who aren't playing; two tabs count once.
// Callers hold m.mu.
func (m *Match) spectators() int {
	seen := map[string]bool{}
	for c := range m.clients {
		if m.colorOf(c.playerID) < 0 {
			seen[c.playerID] = true
		}
	}
	return len(seen)
}

func (m *Match) colorOf(playerID string) int {
	for i, p := range m.seats {
		if p != "" && p == playerID {
			return i
		}
	}
	return -1
}

func (m *Match) online(color int) bool {
	for c := range m.clients {
		if c.playerID == m.seats[color] {
			return true
		}
	}
	return false
}

func (m *Match) handle(c *client, msg inMsg) {
	m.mu.Lock()
	defer m.mu.Unlock()
	color := m.colorOf(c.playerID)
	if color < 0 {
		if msg.Type == "watch" {
			side := white
			if msg.Side == "b" {
				side = black
			}
			if c.side != side {
				c.side = side
				c.push(m.state(c))
			}
		}
		return
	}
	over, plies, round := m.game.Over(), len(m.game.History()), m.round
	now := time.Now()
	m.touched = now

	switch msg.Type {
	case "setup":
		if m.setUp || m.seats[black] == "" {
			break
		}
		if msg.Board != "" && checkSetup(setupModes[m.settings.Setup], color, msg.Board) != nil {
			break
		}
		m.placed[color] = msg.Board
		if m.placed[white] != "" && m.placed[black] != "" {
			m.startSetUp(now)
		}
	case "move":
		if over || !m.setUp || m.seats[black] == "" || int(m.game.Turn()) != color {
			break
		}
		// A move that arrives after the flag fell is too late.
		if m.flagged(now) {
			break
		}
		running, spent := m.running(), now.Sub(m.turnStart)
		if _, err := m.game.PlayUCI(msg.UCI); err != nil {
			break
		}
		m.draw = [2]bool{}
		t := m.settings.Time
		if opp := 1 - color; m.hooks.notify != nil && (t.PerMove > 0 || !t.clocked()) && now.Sub(m.turnBegan) > slowMove &&
			m.users[opp] != 0 && !m.online(opp) && !m.game.Over() {
			go m.hooks.notify(notification{UserID: m.users[opp], Kind: notifyMove, ActorID: m.users[color], MatchID: m.id, Round: m.round})
		}
		m.turnBegan = now
		if running {
			m.clock[color] += time.Duration(t.Inc)*time.Second - spent
			if t.PerMove > 0 {
				m.clock[color] = time.Duration(t.PerMove) * time.Second
			}
			m.turnStart = time.Time{}
			m.stopTimer()
		}
		m.startClock(now)
	case "resign":
		if !over && len(m.game.History()) > 0 {
			m.game.Resign(engine.Color(color))
		}
	case "draw":
		if over || !m.setUp || m.seats[black] == "" {
			break
		}
		m.draw[color] = !m.draw[color]
		if m.draw[white] && m.draw[black] {
			m.game.AgreeDraw()
		}
	case "rematch":
		if !over {
			break
		}
		m.rematch[color] = !m.rematch[color]
		if m.rematch[white] && m.rematch[black] {
			m.game = engine.NewGameWithRules(m.settings.rules())
			m.seats[white], m.seats[black] = m.seats[black], m.seats[white]
			m.users[white], m.users[black] = m.users[black], m.users[white]
			m.names[white], m.names[black] = m.names[black], m.names[white]
			m.rematch = [2]bool{}
			m.draw = [2]bool{}
			m.setUp, m.placed = m.settings.Setup == "", [2]string{}
			m.round++
			m.started = now
			m.turnBegan = now
			m.ratingDiff = nil
			m.resetClock()
			m.startClock(now)
			m.chat = nil
			m.sendChat(true, nil)
		}
	case "chat":
		text := strings.TrimSpace(msg.Text)
		if text == "" || !utf8.ValidString(text) {
			break
		}
		if utf8.RuneCountInString(text) > chatMaxLen {
			text = string([]rune(text)[:chatMaxLen])
		}
		m.chatID++
		line := chatLine{ID: m.chatID, From: colorNames[color], Text: text, At: time.Now().UnixMilli()}
		m.chat = append(m.chat, line)
		if len(m.chat) > chatMaxLines {
			m.chat = slices.Clone(m.chat[len(m.chat)-chatMaxLines:])
		}
		m.sendChat(false, []chatLine{line})
		return // the game didn't change
	}
	if m.game.Over() {
		m.stopClock(now)
	}
	if len(m.game.History()) != plies || m.game.Over() != over || m.round != round {
		m.save()
	}
	m.broadcast()
}

// startSetUp starts the game from both players' setups. A king that starts
// under attack is left as it is: the game is played from there. Callers hold m.mu.
func (m *Match) startSetUp(now time.Time) {
	g, err := engine.NewGameFromFEN(setupFEN(m.placed), m.settings.rules())
	if err != nil {
		log.Printf("match %s: setup: %v", m.id, err)
		m.placed = [2]string{}
		return
	}
	m.game, m.setUp, m.placed = g, true, [2]string{}
	m.turnBegan = now
	m.startClock(now)
	m.save()
}

func (m *Match) chatJSON(reset bool, lines []chatLine) []byte {
	if lines == nil {
		lines = []chatLine{}
	}
	b, _ := json.Marshal(chatMsg{Type: "chat", Reset: reset, Lines: lines})
	return b
}

// sendChat sends chat lines to the players' connections; spectators don't see
// the chat. Callers hold m.mu.
func (m *Match) sendChat(reset bool, lines []chatLine) {
	b := m.chatJSON(reset, lines)
	for c := range m.clients {
		if m.colorOf(c.playerID) >= 0 {
			c.send(b)
		}
	}
}

// base is the part of the state message that's the same for everyone. Callers hold m.mu.
func (m *Match) base() stateMsg {
	status, result, reason := "playing", "", ""
	switch {
	case m.game.Over():
		status, result, reason = "over", m.game.Outcome().String(), m.game.Method().String()
	case m.seats[black] == "":
		status = "waiting"
	case !m.setUp:
		status = "setup"
	}
	base := stateMsg{
		Type: "state", Protocol: protocolVersion, ID: m.id, Round: m.round, Settings: m.settings,
		Status: status, Result: result, Reason: reason,
		Players:    map[string]seatState{},
		Rematch:    map[string]bool{"w": m.rematch[white], "b": m.rematch[black]},
		Draw:       map[string]bool{"w": m.draw[white], "b": m.draw[black]},
		Rated:      m.rated(),
		Spectators: m.spectators(),
	}
	if m.settings.Time.clocked() {
		now := time.Now()
		base.Clock = &clockView{Left: map[string]int64{}}
		for i, name := range colorNames {
			base.Clock.Left[name] = max(m.left(i, now), 0).Milliseconds()
		}
		if m.running() {
			base.Clock.Running = m.game.Turn().String()
		}
	}
	if d := m.ratingDiff; d != nil {
		base.RatingDiff = map[string]int{"w": d[white], "b": d[black]}
	}
	if status == "over" {
		base.Sniper = sniperWin(result, reason, len(m.game.History()))
	}
	for i, name := range colorNames {
		base.Players[name] = seatState{Taken: m.seats[i] != "", Online: m.seats[i] != "" && m.online(i), Name: m.names[i]}
	}
	return base
}

// view fills in a state message for a player (color 0 or 1) or for a spectator
// (color -1) who sees the game as side. Callers hold m.mu.
func (m *Match) view(msg stateMsg, color, side int) []byte {
	if msg.Status == "setup" {
		// Nothing is on the board until both players are ready.
		msg.gameView = gameView{Board: strings.Repeat(".", 64), Turn: "w", Legal: []string{}, History: []string{}, Captured: map[string]string{"w": "", "b": ""}}
		msg.Setup = &setupState{Ready: map[string]bool{"w": m.placed[white] != "", "b": m.placed[black] != ""}}
		if color >= 0 {
			msg.You = colorNames[color]
			msg.Setup.Mine = m.placed[color]
			if !m.settings.Fog {
				msg.Setup.Theirs = m.placed[1-color]
			}
		} else {
			msg.Watching = colorNames[side]
		}
		b, _ := json.Marshal(msg)
		return b
	}
	if color >= 0 {
		msg.You = colorNames[color]
		msg.gameView = viewFor(m.game, color, msg.Status == "playing", m.settings.Fog)
	} else {
		msg.Watching = colorNames[side]
		msg.gameView = spectatorView(m.game, m.settings.rules(), side, m.settings.Fog)
	}
	b, _ := json.Marshal(msg)
	return b
}

// state is the state message for one client. Callers hold m.mu.
func (m *Match) state(c *client) []byte {
	return m.view(m.base(), m.colorOf(c.playerID), c.side)
}

// broadcast sends every client its own view of the match. Callers hold m.mu.
func (m *Match) broadcast() {
	base := m.base()
	// White, black, and spectators seeing it as white or black; built once per broadcast.
	var views [4][]byte
	for c := range m.clients {
		color := m.colorOf(c.playerID)
		slot := color
		if slot < 0 {
			slot = 2 + c.side
		}
		if views[slot] == nil {
			views[slot] = m.view(base, color, c.side)
		}
		c.push(views[slot])
	}
}

// serve runs one websocket connection until it closes.
func (m *Match) serve(ctx context.Context, conn *websocket.Conn, p *player) {
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

	conn.SetReadLimit(4096) // room for a chat message, even one escaped in full
	for {
		_, b, err := conn.Read(ctx)
		if err != nil {
			if !errors.Is(err, context.Canceled) {
				conn.Close(websocket.StatusNormalClosure, "")
			}
			return
		}
		var msg inMsg
		if json.Unmarshal(b, &msg) != nil {
			continue
		}
		if msg.Type == "ping" {
			pong, _ := json.Marshal(map[string]any{"type": "pong", "t": msg.T})
			c.send(pong)
			continue
		}
		start := time.Now()
		m.handle(c, msg)
		if *verbose {
			log.Printf("match %s: %s %s handled in %s", m.id, msg.Type, msg.UCI, time.Since(start))
		}
	}
}
