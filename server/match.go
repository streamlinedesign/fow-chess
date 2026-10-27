package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

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
}

func NewHub(store *Store) *Hub {
	h := &Hub{matches: make(map[string]*Match), store: store}
	go h.sweep()
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
		m.seat(white, creator)
		m.save()
		h.matches[id] = m
		return m
	}
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
	m, err := restoreMatch(h.store, rec)
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
			m.mu.Unlock()
			if idle {
				delete(h.matches, id)
			}
		}
		h.mu.Unlock()
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
	mu       sync.Mutex
	latest   []byte        // newest unsent state; older unsent states are dropped
	pongs    [][]byte      // ping replies, always sent
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

func (c *client) pong(b []byte) {
	c.mu.Lock()
	c.pongs = append(c.pongs, b)
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
	out := c.pongs
	c.pongs = nil
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
	round    int
	started  time.Time // when this round began
	touched  time.Time
}

func newMatch(store *Store, id string, settings Settings) *Match {
	now := time.Now()
	return &Match{
		store: store, id: id, settings: settings, game: engine.NewGameWithRules(settings.rules()),
		clients: make(map[*client]struct{}), round: 1, started: now, touched: now,
	}
}

// restoreMatch rebuilds a match from its newest saved round by replaying the moves.
func restoreMatch(store *Store, rec *gameRecord) (*Match, error) {
	m := newMatch(store, rec.MatchID, rec.Settings)
	m.round, m.started = rec.Round, rec.CreatedAt
	m.seats, m.users = rec.Seats, rec.Users
	m.names = [2]string{rec.whiteName, rec.blackName}
	g, err := replay(rec)
	if err != nil {
		return nil, err
	}
	m.game = g
	return m, nil
}

// replay plays a saved game's moves and applies a result the moves don't show
// (a resignation or an agreed draw).
func replay(rec *gameRecord) (*engine.Game, error) {
	g := engine.NewGameWithRules(rec.Settings.rules())
	for _, uci := range strings.Fields(rec.Moves) {
		if _, err := g.PlayUCI(uci); err != nil {
			return nil, fmt.Errorf("move %s: %w", uci, err)
		}
	}
	if !g.Over() && rec.Result != "" {
		switch {
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
		MatchID: m.id, Round: m.round, Settings: m.settings, Seats: m.seats, Users: m.users,
		Moves: moves.String(), CreatedAt: m.started, UpdatedAt: time.Now(),
	}
	if m.game.Over() {
		rec.Result, rec.Reason = m.game.Outcome().String(), m.game.Method().String()
	}
	m.store.SaveGame(rec)
}

type seatState struct {
	Taken  bool   `json:"taken"`
	Online bool   `json:"online"`
	Name   string `json:"name,omitempty"` // username; empty for a guest
}

// protocolVersion changes whenever the state message changes shape, so a web app
// talking to a server from another version can say so instead of breaking.
const protocolVersion = 5

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
}

type inMsg struct {
	Type string  `json:"type"`
	UCI  string  `json:"uci"`
	T    float64 `json:"t"` // ping timestamp, echoed back
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
					m.save()
					break
				}
			}
		}
	}
	m.clients[c] = struct{}{}
	m.touched = time.Now()
	m.broadcast()
}

func (m *Match) leave(c *client) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.clients, c)
	m.touched = time.Now()
	m.broadcast()
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
		return
	}
	over, plies, round := m.game.Over(), len(m.game.History()), m.round
	m.touched = time.Now()

	switch msg.Type {
	case "move":
		if over || m.seats[black] == "" || int(m.game.Turn()) != color {
			break
		}
		if _, err := m.game.PlayUCI(msg.UCI); err != nil {
			break
		}
		m.draw = [2]bool{}
	case "resign":
		if !over && len(m.game.History()) > 0 {
			m.game.Resign(engine.Color(color))
		}
	case "draw":
		if over || m.seats[black] == "" {
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
			m.round++
			m.started = time.Now()
		}
	}
	if len(m.game.History()) != plies || m.game.Over() != over || m.round != round {
		m.save()
	}
	m.broadcast()
}

// broadcast sends every client its own view of the match. Callers hold m.mu.
func (m *Match) broadcast() {
	status, result, reason := "playing", "", ""
	switch {
	case m.game.Over():
		status, result, reason = "over", m.game.Outcome().String(), m.game.Method().String()
	case m.seats[black] == "":
		status = "waiting"
	}
	base := stateMsg{
		Type: "state", Protocol: protocolVersion, ID: m.id, Round: m.round, Settings: m.settings,
		Status: status, Result: result, Reason: reason,
		Players: map[string]seatState{},
		Rematch: map[string]bool{"w": m.rematch[white], "b": m.rematch[black]},
		Draw:    map[string]bool{"w": m.draw[white], "b": m.draw[black]},
	}
	for i, name := range colorNames {
		base.Players[name] = seatState{Taken: m.seats[i] != "", Online: m.seats[i] != "" && m.online(i), Name: m.names[i]}
	}
	playable := status == "playing"
	var views [3][]byte // white, black, spectator; built once per broadcast
	for c := range m.clients {
		color := m.colorOf(c.playerID)
		slot := color
		if slot < 0 {
			slot = 2
		}
		if views[slot] == nil {
			msg := base
			if color >= 0 {
				msg.You = colorNames[color]
			}
			msg.gameView = viewFor(m.game, color, playable, m.settings.Fog)
			views[slot], _ = json.Marshal(msg)
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

	conn.SetReadLimit(1024)
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
			c.pong(pong)
			continue
		}
		start := time.Now()
		m.handle(c, msg)
		if *verbose {
			log.Printf("match %s: %s %s handled in %s", m.id, msg.Type, msg.UCI, time.Since(start))
		}
	}
}
